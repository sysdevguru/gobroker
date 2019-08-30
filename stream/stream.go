package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/service/accesskey"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/log"
	"github.com/alpacahq/gopaca/rmq/pubsub"
	"github.com/eapache/channels"
	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"github.com/vmihailenco/msgpack"
)

const (
	// websocket config
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	// streams
	AccountUpdates = "account_updates"
	TradeUpdates   = "trade_updates"
)

var (
	send     *channels.InfiniteChannel
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	authFunc func(keyId, secretKey string) (*models.AccessKey, error)
)

// AccountUpdatesStream returns the stream string for account updates with
// account ID suffix for routing.
func AccountUpdatesStream(id uuid.UUID) string {
	return fmt.Sprintf("%s_%s", AccountUpdates, id.String())
}

// TradeUpdatesStream returns the stream string for trade updates with
// account ID suffix for routing.
func TradeUpdatesStream(id uuid.UUID) string {
	return fmt.Sprintf("%s_%s", TradeUpdates, id.String())
}

// InboundMessage is the standard message sent by clients of the stream interface
type InboundMessage struct {
	Action string                 `json:"action" msgpack:"action"`
	Data   map[string]interface{} `json:"data" msgpack:"data"`
}

// OutboundMessage is the standard message sent by the server to update clients
// of the stream interface
type OutboundMessage struct {
	Stream string      `json:"stream" msgpack:"stream"`
	Data   interface{} `json:"data" msgpack:"data"`
}

type Listener struct {
	sync.Mutex
	c            *websocket.Conn
	done         chan struct{}
	marshal      func(v interface{}) ([]byte, error)
	unmarshal    func(data []byte, v interface{}) error
	auth         atomic.Value
	accountID    string
	keyID        string
	authenticate func(keyId, secretKey string) (*models.AccessKey, error)
}

func (l *Listener) authenticated() bool {
	return l.auth.Load() != nil
}

func (l *Listener) authorize(id, keyID string) {
	l.accountID = id
	l.keyID = keyID
	l.auth.Store(struct{}{})
}

func (l *Listener) handleOutbound(m OutboundMessage) {
	if buf, err := l.marshal(m); err != nil {
		log.Error(
			"stream outbound marshal error",
			"key_id", l.keyID,
			"msg", m,
			"listener", l.c.RemoteAddr().String(),
			"error", err)
	} else {
		// prevents concurrent write to the websocket connection
		log.Debug(
			"stream outbound",
			"key_id", l.keyID,
			"stream", m.Stream,
			"data", string(buf[:20]),
			"listener", l.c.RemoteAddr().String())

		l.Lock()
		defer l.Unlock()

		if err := l.c.WriteMessage(websocket.BinaryMessage, buf); err != nil {
			log.Error(
				"stream outbound write error",
				"key_id", l.keyID,
				"msg", string(buf),
				"listener", l.c.RemoteAddr().String(),
				"error", err)
		}
	}
}

// only support authentication and listen for now
func (l *Listener) handleInbound(m InboundMessage) {
	switch m.Action {
	case "authenticate":
		if v, ok := m.Data["key_id"]; ok {
			keyID := v.(string)
			if v, ok = m.Data["secret_key"]; ok {
				secretKey := v.(string)

				if accessKey, err := l.authenticate(keyID, secretKey); err == nil {
					l.authorize(accessKey.AccountID.String(), keyID)

					l.handleOutbound(OutboundMessage{
						Stream: "authorization",
						Data: map[string]interface{}{
							"status": "authorized",
							"action": "authenticate",
						},
					})
				} // don't notify of error for security reasons

				if !l.authenticated() {
					l.handleOutbound(OutboundMessage{
						Stream: "authorization",
						Data: map[string]interface{}{
							"status": "unauthorized",
							"action": "authenticate",
						},
					})
				}
			}
		}
	case "listen":
		if !l.authenticated() {
			l.handleOutbound(OutboundMessage{
				Stream: "authorization",
				Data: map[string]interface{}{
					"status": "unauthorized",
					"action": "listen",
				},
			})
			return
		}

		streams := l.parseStreams(m.Data)
		router.Update(l, streams)
		strippedStreams := make([]string, len(streams))

		for i, stream := range streams {
			strippedStreams[i] = stripStream(stream)
		}

		l.handleOutbound(OutboundMessage{
			Stream: "listening",
			Data: map[string]interface{}{
				"streams": strippedStreams,
			},
		})
	}
}

func (l *Listener) parseStreams(data map[string]interface{}) (streams []string) {
	if v, ok := data["streams"]; ok {
		for _, s := range v.([]interface{}) {
			stream, ok := s.(string)

			if !ok {
				continue
			}

			if !validStream(stream) {
				continue
			}

			streams = append(streams, decorateStream(stream, l.accountID))
		}
	}
	return streams
}

func validStream(stream string) bool {
	switch stream {
	case AccountUpdates:
		fallthrough
	case TradeUpdates:
		return true
	default:
		return false
	}
}

func decorateStream(stream string, id string) string {
	switch {
	case strings.EqualFold(stream, AccountUpdates):
		fallthrough
	case strings.EqualFold(stream, TradeUpdates):
		stream = fmt.Sprintf("%v_%v", stream, id)
	}

	return stream
}

func stripStream(stream string) string {
	switch {
	case strings.Contains(stream, AccountUpdates):
		return AccountUpdates
	case strings.Contains(stream, TradeUpdates):
		return TradeUpdates
	default:
		return stream
	}
}

func (l *Listener) consume() {
	defer func() {
		// cleanup cache when the connection is closed
		router.Update(l, []string{})
		l.done <- struct{}{}
	}()
	l.c.SetPongHandler(func(string) error {
		return l.c.SetReadDeadline(clock.Now().Add(pongWait))
	})
	for {
		msgType, buf, err := l.c.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				log.Warn(
					"stream unexpected socket failure",
					"listener", l.c.RemoteAddr().String(),
					"error", err)
			}
			return
		}
		switch msgType {
		case websocket.TextMessage:
			fallthrough
		case websocket.BinaryMessage:
			m := InboundMessage{}
			if err = l.unmarshal(buf, &m); err != nil {
				// don't log for security reasons
				continue
			}
			log.Debug(
				"stream inbound",
				"key_id", l.keyID,
				"action", m.Action,
				"data", string(buf[:20]),
				"listener", l.c.RemoteAddr().String())

			l.handleInbound(m)
		case websocket.CloseMessage:
			return
		}
	}
}

func stream() {
	for v := range send.Out() {
		if v == nil {
			continue
		}

		m := v.(OutboundMessage)
		listeners := router.GetListeners(m.Stream)
		m.Stream = stripStream(m.Stream)

		for _, l := range listeners {
			l.handleOutbound(m)
		}
	}
}

func (l *Listener) produce() {
	ticker := time.NewTicker(pingPeriod)
	for {
		select {
		case <-ticker.C:
			l.c.WriteMessage(websocket.PingMessage, []byte{})
		case <-l.done:
			return
		}
	}
}

// pushForTest sends data locally to the stream interface.
// This is mainly for testing, and the mssages across cluster needs to be sent
// through the message queue.
func pushForTest(stream string, data interface{}) {
	send.In() <- OutboundMessage{Stream: stream, Data: data}
}

func rmqSubscribe(c <-chan pubsub.Message, cancel context.CancelFunc) context.CancelFunc {
	go func() {
		for buf := range c {
			msg := OutboundMessage{}
			if err := json.Unmarshal(buf, &msg); err != nil {
				log.Error("stream failed to parse rmq message", "error", err, "message", string(buf))
				continue
			}
			send.In() <- msg
		}
	}()

	return cancel
}

// Initialize builds the send channel as well as the cache, and
// must be called before any data flows over the stream interface
func Initialize(authService accesskey.AccessKeyService, c <-chan pubsub.Message, cancel context.CancelFunc) {
	authFunc = func(keyId, secretKey string) (*models.AccessKey, error) {
		service := authService.WithTx(db.DB())
		return service.Verify(keyId, secretKey)
	}

	send = channels.NewInfiniteChannel()
	router = NewRouter()
	router.cancel = rmqSubscribe(c, cancel)

	go stream()
}

// Handler hooks into the REST interface and handles the incoming
// streaming requests, and upgrades the connection
func Handler(w http.ResponseWriter, r *http.Request) {
	// upgrade the socket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("stream socket upgrade error", "error", err)
		return
	}

	// build the listener
	l := Listener{
		c:            ws,
		done:         make(chan struct{}),
		authenticate: authFunc,
	}

	// check the codec
	switch r.Header.Get("Content-Type") {
	case "application/x-msgpack":
		l.marshal = marshalMsgPack
		l.unmarshal = unmarshalMsgPack
	default:
		l.marshal = json.Marshal
		l.unmarshal = json.Unmarshal
	}

	if l.c != nil {
		log.Info("new stream listener", "listener", ws.RemoteAddr().String())
	}

	// begin streaming
	go l.consume()
	go l.produce()
}

// msgpack marshal wrapper
func marshalMsgPack(v interface{}) ([]byte, error) {
	return msgpack.Marshal(v)
}

// msgpack unmarshal wrapper
func unmarshalMsgPack(data []byte, v interface{}) error {
	return msgpack.Unmarshal(data, v)
}
