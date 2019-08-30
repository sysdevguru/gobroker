package stream

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/service/assetcache"
	"github.com/eapache/channels"
	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type StreamTestSuite struct {
	suite.Suite
}

func TestStreamTestSuite(t *testing.T) {
	suite.Run(t, new(StreamTestSuite))
}

func loadAssetsMock() ([]*models.Asset, error) {
	return []*models.Asset{
		&models.Asset{
			ID:       "59ffb624-2872-490c-874b-badda1589c1b",
			Class:    "us_equity",
			Exchange: "NASDAQ",
			Symbol:   "AAPL",
			Status:   enum.AssetActive,
			Tradable: true,
		},
	}, nil
}

func InitializeForTest() {
	assetcache.MockLoadAssets(loadAssetsMock)
	send = channels.NewInfiniteChannel()
	router = NewRouter()
	go stream()
}

func (s *StreamTestSuite) TestCache() {
	InitializeForTest()
	l1 := Listener{}
	s1 := "stream_1"
	s2 := "stream_2"
	router.Update(&l1, []string{s1, s2})

	listeners := router.getListeners(s1)
	assert.Len(s.T(), listeners, 1)
	assert.Equal(s.T(), listeners[0], &l1)

	listeners = router.getListeners(s2)
	assert.Len(s.T(), listeners, 1)
	assert.Equal(s.T(), listeners[0], &l1)

	streams := router.getStreams(&l1)
	assert.Len(s.T(), streams, 2)
	for _, st := range streams {
		assert.True(s.T(), st == s1 || st == s2)
	}

	s3 := "stream_3"
	router.Update(&l1, []string{s1, s3})

	listeners = router.getListeners(s3)
	assert.Len(s.T(), listeners, 1)
	assert.Equal(s.T(), listeners[0], &l1)

	listeners = router.getListeners(s2)
	assert.Len(s.T(), listeners, 0)

	router.Update(&l1, []string{s1, s2, s3})

	listeners = router.getListeners(s1)
	assert.Len(s.T(), listeners, 1)
	assert.Equal(s.T(), listeners[0], &l1)

	listeners = router.getListeners(s2)
	assert.Len(s.T(), listeners, 1)
	assert.Equal(s.T(), listeners[0], &l1)

	listeners = router.getListeners(s3)
	assert.Len(s.T(), listeners, 1)
	assert.Equal(s.T(), listeners[0], &l1)

	// multiple listeners
	l2 := Listener{}
	s4 := "stream_4"
	router.Update(&l2, []string{s2, s4})

	listeners = router.getListeners(s4)
	assert.Len(s.T(), listeners, 1)
	assert.Equal(s.T(), listeners[0], &l2)

	listeners = router.getListeners(s2)
	assert.Len(s.T(), listeners, 2)

	streams = router.getStreams(&l2)
	assert.Len(s.T(), streams, 2)
	for _, st := range streams {
		assert.True(s.T(), st == s2 || st == s4)
	}

	router.Update(&l2, []string{s4})
	streams = router.getStreams(&l2)
	assert.Len(s.T(), streams, 1)
	for _, st := range streams {
		assert.True(s.T(), st == s4)
	}

	router.Update(&l1, []string{})
	assert.Len(s.T(), router.getStreams(&l1), 0)
}

func (s *StreamTestSuite) TestCacheConcurrency() {
	InitializeForTest()

	l := Listener{}
	s1 := "stream_1"
	s2 := "stream_2"

	router.Update(&l, []string{s1})

	var wg sync.WaitGroup
	routines := make(chan func(), 2)
	routines <- func() {
		router.Update(&l, []string{})
		wg.Done()
	}
	routines <- func() {
		router.Update(&l, []string{s1, s2})
		wg.Done()
	}
	close(routines)
	for r := range routines {
		wg.Add(1)
		r()
	}
	wg.Wait()
	streams := router.getStreams(&l)
	assert.Len(s.T(), streams, 2)
}

func (s *StreamTestSuite) TestStream() {
	accountID := uuid.Must(uuid.NewV4())
	keyID := uuid.Must(uuid.NewV4())

	authFunc = func(keyId string, secretKey string) (*models.AccessKey, error) {
		return &models.AccessKey{
			ID:        keyId,
			AccountID: accountID,
		}, nil
	}

	InitializeForTest()
	srv := httptest.NewServer(http.HandlerFunc(Handler))
	u, _ := url.Parse(srv.URL)
	u.Scheme = "ws"
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		assert.FailNow(s.T(), fmt.Sprintf("failed to connect to websocket: %v", err))
	}

	// send auth
	secret := uuid.Must(uuid.NewV4())

	if err := conn.WriteJSON(InboundMessage{Action: "authenticate", Data: map[string]interface{}{
		"key_id":     keyID.String(),
		"secret_key": secret.String(),
	}}); err != nil {
		assert.FailNow(s.T(), fmt.Sprintf("failed to authenticate - error: %v", err))
	}

	// receive auth ack
	om := OutboundMessage{}
	_, msg, err := conn.ReadMessage()
	if err != nil {
		assert.FailNow(s.T(), fmt.Sprintf("failed to read auth ack: %v", err))
	}
	if err = json.Unmarshal(msg, &om); err != nil || om.Stream != "authorization" {
		assert.FailNow(s.T(), fmt.Sprintf("invalid auth ack received: %v", string(msg)))
	}

	// listen
	if err := conn.WriteJSON(InboundMessage{Action: "listen", Data: map[string]interface{}{
		"streams": []string{TradeUpdates},
	}}); err != nil {
		assert.FailNow(s.T(), fmt.Sprintf("failed to listen - error: %v", err))
	}

	// read listen ack
	_, msg, err = conn.ReadMessage()
	if err != nil {
		assert.FailNow(s.T(), fmt.Sprintf("failed to read auth ack: %v", err))
	}
	if err = json.Unmarshal(msg, &om); err != nil || om.Stream != "listening" {
		assert.FailNow(s.T(), fmt.Sprintf("invalid auth ack received: %v", string(msg)))
	}

	// stream some data
	for i := 0; i < 5; i++ {
		pushForTest(TradeUpdatesStream(accountID), map[string]interface{}{
			"event":     "fill",
			"qty":       "100",
			"price":     "179.08",
			"timestamp": "2018-02-28T20:38:22Z",
			"order": map[string]interface{}{
				"id":              "7b7653c4-7468-494a-aeb3-d5f255789473",
				"client_order_id": "7b7653c4-7468-494a-aeb3-d5f255789473",
				"asset_id":        "904837e3-3b76-47ec-b432-046db621571b",
				"symbol":          "AAPL",
				"exchange":        "NASDAQ",
				"asset_class":     "us_equity",
				"side":            "buy",
			},
		})
	}

	// receive the data
	for i := 0; i < 5; i++ {
		_, msg, err = conn.ReadMessage()
		if err != nil {
			assert.FailNow(s.T(), fmt.Sprintf("failed to read data stream message - error: %v", err))
		}
		if err = json.Unmarshal(msg, &om); err != nil {
			assert.FailNow(s.T(), fmt.Sprintf("failed to unmarshal data stream message %v - error: %v", string(msg), err))
		}
	}

	if err = conn.Close(); err != nil {
		assert.FailNow(s.T(), fmt.Sprintf("failed to close websocket - error: %v", err))
	}
}
