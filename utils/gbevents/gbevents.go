package gbevents

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/alpacahq/gobroker/utils/signalman"
	"github.com/alpacahq/gopaca/log"
	"github.com/alpacahq/gopaca/rmq/pubsub"
)

type Event struct {
	Name    string                  `json:"name"`
	Payload *map[string]interface{} `json:"payload"`
}

var (
	once sync.Once

	handlers  []func(*Event)
	handlerMu sync.RWMutex

	cancellerSub context.CancelFunc
	cancellerPub context.CancelFunc
	publisher    chan<- pubsub.Message

	EventAssetRefreshed    = "asset_refreshed"
	EventAccessKeyDisabled = "access_key_disabled"
)

func RegisterFunc(handler func(*Event)) {
	handlerMu.Lock()
	defer handlerMu.Unlock()
	handlers = append(handlers, handler)
}

func RegisterSignalHandler() {
	signalman.RegisterFunc("gbevents", shutdown)
}

func shutdown() error {
	if cancellerSub != nil {
		cancellerSub()
	}
	if cancellerPub != nil {
		cancellerPub()
	}
	return nil
}

func TriggerEvent(evt *Event) {
	once.Do(func() {
		msgs, cancel := pubsub.NewPubSub("gbevents").Publish()
		cancellerPub = cancel
		publisher = msgs
	})

	buf, _ := json.Marshal(evt)

	publisher <- pubsub.Message(buf)
}

func RunForever() {
	c, cancel := pubsub.NewPubSub("gbevents").Subscribe()

	cancellerSub = cancel

	for msg := range c {
		evt := Event{}

		if err := json.Unmarshal(msg, &evt); err != nil {
			log.Error("failed to unmarshal msg")
			continue
		}

		log.Debug("receive gbevents", "event", evt)

		handlerMu.RLock()
		for _, handler := range handlers {
			handler(&evt)
		}

		handlerMu.RUnlock()
	}
}
