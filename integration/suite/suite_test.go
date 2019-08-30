// +build integration

package suite

import (
	"net/url"
	"testing"
	"time"

	"github.com/alpacahq/apex"
	"github.com/alpacahq/gobroker/integration/apiclient"
	"github.com/alpacahq/gobroker/integration/testop"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/rest/api/controller/entities"
	"github.com/alpacahq/gobroker/rest/api/controller/order"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/log"
	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TestSuite struct {
	suite.Suite
	apiKey testop.ApiKey
	id     string
}

func TestIt(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (s *TestSuite) SetupSuite() {
	// get the api key and create the client
	if err := db.DB().First(&s.apiKey).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	log.Info("Start", "time", clock.Now())

	client := apiclient.NewRestClient(s.apiKey.KeyID, s.apiKey.SecretKey)
	client.SetBaseURL("http://nginxsvc")

	// get the relationship & fund the account ahead of time
	acct, err := client.GetAccount()
	require.Nil(s.T(), err)

	s.id = acct.ID

	plaidAcct := "test_plaid_account"

	rel := &models.ACHRelationship{
		ID:             uuid.Must(uuid.NewV4()).String(),
		AccountID:      acct.ID,
		Status:         enum.RelationshipApproved,
		ApprovalMethod: apex.Plaid,
		PlaidAccount:   &plaidAcct,
	}

	require.Nil(s.T(), db.DB().Create(rel).Error)

	nc := apiclient.NewInternalClient(
		"integration-test1@alpaca.markets",
		"test-integration1",
		&s.id,
	)

	nc.SetBaseURL("http://nginxsvc")

	req := entities.TransferRequest{
		RelationshipID: rel.ID,
		Direction:      apex.Incoming,
		Amount:         decimal.New(10000, 0),
	}

	_, err = nc.CreateTransfer(acct.ID, req)

	require.Nil(s.T(), err)
}

func (s *TestSuite) TearDownSuite() {

}

func (s *TestSuite) TestREST() {
	start := clock.Now()
	client := apiclient.NewRestClient(s.apiKey.KeyID, s.apiKey.SecretKey)
	client.SetBaseURL("http://nginxsvc")

	// Get Account
	{
		account, err := client.GetAccount()
		assert.Nil(s.T(), err)
		assert.NotNil(s.T(), account)
	}

	// Get Account (msgpack)
	{
		client.RequestHeaders = map[string]string{
			"Content-Type": api.MIMEApplicationMsgpack,
		}
		account, err := client.GetAccount()
		assert.Nil(s.T(), err)
		assert.NotNil(s.T(), account)
		client.RequestHeaders["Content-Type"] = api.MIMEApplicationJSON
	}

	// Patch Configurations
	{
		param := apiclient.PatchConfigurationsRequest{
			"suspend_trade": true,
		}
		err := client.PatchConfigurations(param)
		assert.Nil(s.T(), err)
		account, err := client.GetAccount()
		assert.Nil(s.T(), err)
		assert.True(s.T(), account.TradeSuspendedByUser)

		param = apiclient.PatchConfigurationsRequest{
			"suspend_trade": false,
		}
		err = client.PatchConfigurations(param)
		assert.Nil(s.T(), err)
		account, err = client.GetAccount()
		assert.Nil(s.T(), err)
		assert.False(s.T(), account.TradeSuspendedByUser)
	}

	// unauthorized case
	{
		client := apiclient.NewRestClient("invalid", "invalid")
		client.SetBaseURL("http://nginxsvc")
		account, err := client.GetAccount()
		assert.NotNil(s.T(), err)
		assert.Equal(s.T(), err.(*apiclient.ApiError).Code, 40110000)
		assert.Nil(s.T(), account)
	}

	// Positions
	{
		positions, err := client.ListPositions()
		assert.Nil(s.T(), err)
		assert.NotNil(s.T(), positions)
	}

	// Clock
	{
		clock, err := client.GetClock()
		assert.Nil(s.T(), err)
		assert.NotNil(s.T(), clock)
	}

	// Calendar
	{
		calendar, err := client.GetCalendar()
		assert.Nil(s.T(), err)
		assert.NotNil(s.T(), calendar)
	}

	s.T().Logf("TestREST complete [%v]", clock.Now().Sub(start))
}

func (s *TestSuite) TestOrders() {
	start := clock.Now()
	client := apiclient.NewRestClient(s.apiKey.KeyID, s.apiKey.SecretKey)
	client.SetBaseURL("http://nginxsvc")

	// connect socket for updates
	c := getStreamSocket()
	defer gracefulClose(c)

	// authenticate the connection
	authRequest := map[string]interface{}{
		"action": "authenticate",
		"data": map[string]interface{}{
			"key_id":     s.apiKey.KeyID,
			"secret_key": s.apiKey.SecretKey,
		},
	}
	err := c.WriteJSON(authRequest)
	assert.Nil(s.T(), err)
	streamMsg := StreamMsg{}
	c.SetReadDeadline(time.Now().Add(time.Minute))
	err = c.ReadJSON(&streamMsg)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "authorized", streamMsg.Data["status"])

	subscribeRequest := map[string]interface{}{
		"action": "listen",
		"data": map[string]interface{}{
			"streams": []interface{}{
				"trade_updates",
			},
		},
	}

	// listen to trade updates
	err = c.WriteJSON(subscribeRequest)
	assert.Nil(s.T(), err)
	streamMsg = StreamMsg{}
	c.SetReadDeadline(time.Now().Add(time.Minute))
	err = c.ReadJSON(&streamMsg)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "listening", streamMsg.Stream)
	assert.Equal(
		s.T(),
		map[string]interface{}{"streams": []interface{}{"trade_updates"}},
		streamMsg.Data,
	)

	// make sure there are no orders
	orders, err := client.ListOrders("all")
	assert.Nil(s.T(), err)
	assert.Len(s.T(), orders, 0)

	// place a market order
	aapl := "AAPL"
	req := order.CreateOrderRequest{
		AssetKey:    &aapl,
		Qty:         decimal.New(int64(1), 0),
		Side:        enum.Buy,
		Type:        enum.Market,
		TimeInForce: enum.GTC,
	}

	o, err := client.PlaceOrder(req)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), o)
	assert.Equal(s.T(), o.Symbol, aapl)
	assert.Equal(s.T(), o.Status, "new")

	// read the new (ack) update
	streamMsg = StreamMsg{}
	c.SetReadDeadline(time.Now().Add(time.Minute))
	err = c.ReadJSON(&streamMsg)
	require.Nil(s.T(), err)
	require.Equal(s.T(), streamMsg.Data["event"].(string), string(enum.ExecutionNew))

	// read the fill update
	streamMsg = StreamMsg{}
	c.SetReadDeadline(time.Now().Add(time.Minute))
	err = c.ReadJSON(&streamMsg)
	require.Nil(s.T(), err)
	require.Equal(s.T(), streamMsg.Data["event"].(string), string(enum.ExecutionFill))

	// wait for commit
	time.Sleep(100 * time.Millisecond)

	// make sure there is a filled order
	orders, err = client.ListOrders("closed")
	require.Nil(s.T(), err)
	require.Len(s.T(), orders, 1)
	assert.Equal(s.T(), string(enum.Market), orders[0].Type)

	// place an unfillable limit order
	limitPx := decimal.NewFromFloat(1.00)
	req = order.CreateOrderRequest{
		AssetKey:    &aapl,
		Qty:         decimal.New(int64(2), 0),
		Side:        enum.Buy,
		Type:        enum.Limit,
		LimitPrice:  &limitPx,
		TimeInForce: enum.GTC,
	}

	o, err = client.PlaceOrder(req)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), o)

	// read the new order update
	streamMsg = StreamMsg{}
	c.SetReadDeadline(time.Now().Add(time.Minute))
	err = c.ReadJSON(&streamMsg)
	assert.Nil(s.T(), err)
	require.NotNil(s.T(), streamMsg.Data["event"])
	assert.Equal(s.T(), string(enum.ExecutionNew), streamMsg.Data["event"].(string))

	// make sure the limit order still hanging around
	orders, err = client.ListOrders("open")
	assert.Nil(s.T(), err)
	assert.Len(s.T(), orders, 1)

	// make sure the filled order is there
	orders, err = client.ListOrders("closed")
	assert.Nil(s.T(), err)
	assert.Len(s.T(), orders, 1)

	// cancel the hanging limit order
	err = client.CancelOrder(o.ID)
	assert.Nil(s.T(), err)

	// read the cancellation update
	c.SetReadDeadline(time.Now().Add(time.Minute))
	err = c.ReadJSON(&streamMsg)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), streamMsg.Data["event"].(string), string(enum.ExecutionCanceled))

	// wait for commit
	time.Sleep(100 * time.Millisecond)

	// make sure the limit order canceled
	orders, err = client.ListOrders("closed")
	assert.Nil(s.T(), err)
	assert.Len(s.T(), orders, 2)

	// check our positions
	positions, err := client.ListPositions()
	assert.Nil(s.T(), err)
	assert.Len(s.T(), positions, 1)

	// place a market order with msgpack
	msgpackClient := apiclient.NewRestClient(s.apiKey.KeyID, s.apiKey.SecretKey)
	msgpackClient.SetBaseURL("http://nginxsvc")
	msgpackClient.RequestHeaders = map[string]string{
		"Content-Type": api.MIMEApplicationMsgpack,
	}
	nvda := "NVDA"
	req = order.CreateOrderRequest{
		AssetKey:    &nvda,
		Qty:         decimal.New(int64(1), 0),
		Side:        enum.Buy,
		Type:        enum.Market,
		TimeInForce: enum.GTC,
	}

	o, err = msgpackClient.PlaceOrder(req)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), o)
	assert.Equal(s.T(), o.Symbol, nvda)
	assert.Equal(s.T(), o.Status, "new")

	s.T().Logf("TestOrders complete [%v]", clock.Now().Sub(start))
}

func (s *TestSuite) TestAssets() {
	start := clock.Now()
	client := apiclient.NewRestClient(s.apiKey.KeyID, s.apiKey.SecretKey)
	client.SetBaseURL("http://nginxsvc")
	// List Assets
	{
		assets, err := client.ListAssets()
		assert.Nil(s.T(), err)
		assert.Len(s.T(), assets, 4)
	}

	// Get Asset by symbol
	{
		asset, err := client.GetAsset("AAPL")
		assert.Nil(s.T(), err)
		assert.Equal(s.T(), "us_equity", asset.Class)
		assert.Equal(s.T(), "NASDAQ", asset.Exchange)
		assert.Equal(s.T(), "AAPL", asset.Symbol)
		assert.Equal(s.T(), "active", asset.Status)
		assert.True(s.T(), asset.Tradable)

		asset, err = client.GetAsset("UVXY:NYSEARCA")
		assert.Nil(s.T(), err)
		assert.Equal(s.T(), "us_equity", asset.Class)
		assert.Equal(s.T(), "NYSEARCA", asset.Exchange)
		assert.Equal(s.T(), "UVXY", asset.Symbol)
	}

	// Get Asset by asset id
	{
		asset, err := client.GetAsset("3b11d285-ba92-45b3-a56a-523d7c89603f")
		assert.Nil(s.T(), err)
		assert.Equal(s.T(), "NYSEARCA", asset.Exchange)
		assert.Equal(s.T(), "SPY", asset.Symbol)
	}

	// Get Asset invalid
	{
		_, err := client.GetAsset("INVALID")
		assert.Equal(s.T(), 40410000, err.(*apiclient.ApiError).Code)
		assert.Equal(s.T(), "asset not found for INVALID", err.(*apiclient.ApiError).Message)

		_, err = client.GetAsset("AAPL:BATS")
		assert.Equal(s.T(), 40410000, err.(*apiclient.ApiError).Code)
		assert.Equal(s.T(), "asset not found for AAPL:BATS", err.(*apiclient.ApiError).Message)
	}

	// List Fundamentals
	{
		fundamentals, err := client.ListFundamentals([]string{
			"AAPL", "INVALID", "SPY",
		})
		assert.Nil(s.T(), err)
		assert.Len(s.T(), fundamentals, 2)
	}

	// Get Single Fundamental
	{
		fundamental, err := client.GetFundamental("AAPL")
		assert.Nil(s.T(), err)
		assert.Equal(s.T(), "Apple Inc", fundamental.FullName)

		_, err = client.GetFundamental("INVALID")
		assert.Equal(s.T(), 40410000, err.(*apiclient.ApiError).Code)
		assert.Equal(s.T(), "asset not found for INVALID", err.(*apiclient.ApiError).Message)
	}
	// List BarLists
	{
		limit := 10
		barLists, err := client.ListBarLists([]string{
			"AAPL", "SPY",
		}, apiclient.BarListParams{
			Limit: &limit,
		})
		assert.Nil(s.T(), err)
		assert.Len(s.T(), barLists, 2)
		assert.Equal(s.T(), "us_equity", barLists[0].Class)
	}

	// Get BarList
	{
		limit := 10
		barList, err := client.GetBarList("AAPL", apiclient.BarListParams{
			Limit: &limit,
		})
		assert.Nil(s.T(), err)
		assert.Len(s.T(), barList.Bars, 10)
		bars := barList.Bars
		barList.Bars = nil
		expected := &apiclient.BarList{
			AssetID:  "fbaa7510-3ea0-4f8e-83b5-d6f527129f48",
			Symbol:   "AAPL",
			Exchange: "NASDAQ",
			Class:    "us_equity",
			Bars:     nil,
		}
		assert.Equal(s.T(), expected, barList)

		expectedBar0 := apiclient.Bar{
			Open:   167.14,
			High:   168.62,
			Low:    166.77,
			Close:  167.84,
			Volume: 1845193,
			Time:   time.Date(2018, 2, 1, 5, 0, 0, 0, time.UTC),
		}
		assert.Equal(s.T(), &expectedBar0, bars[0])
	}

	// List Quotes
	{
		quotes, err := client.ListQuotes([]string{
			"AAPL", "INVALID", "SPY",
		})
		assert.Nil(s.T(), err)
		assert.Len(s.T(), quotes, 2)
		assert.Equal(s.T(), "us_equity", quotes[0].Class)
	}

	// Get Quote
	{
		quote, err := client.GetQuote("AAPL")
		assert.Nil(s.T(), err)
		assert.Equal(s.T(), int64(1528124748), quote.BidTimestamp.Unix())
		assert.Equal(s.T(), float32(192.02), quote.Bid)
		assert.Equal(s.T(), int64(1528124748), quote.AskTimestamp.Unix())
		assert.Equal(s.T(), float32(192.03), quote.Ask)
		assert.Equal(s.T(), int64(1528124748), quote.LastTimestamp.Unix())
		assert.Equal(s.T(), float32(192.02), quote.Last)
		assert.Equal(s.T(), "fbaa7510-3ea0-4f8e-83b5-d6f527129f48", quote.AssetID)
		assert.Equal(s.T(), "AAPL", quote.Symbol)
		assert.Equal(s.T(), "us_equity", quote.Class)
	}

	s.T().Logf("TestAssets complete [%v]", clock.Now().Sub(start))
}

type StreamMsg struct {
	Stream string                 `json:"stream"`
	Data   map[string]interface{} `json:"data"`
}

func getStreamSocket() *websocket.Conn {
	u := url.URL{Scheme: "ws", Host: "nginxsvc", Path: "/stream"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		panic(err)
	}
	return c
}

func gracefulClose(c *websocket.Conn) {
	err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Warn("write closure", "error", err)
		return
	}
	c.Close()
}

func (s *TestSuite) TestInternal() {
	start := clock.Now()

	client := apiclient.NewInternalClient(
		"integration-test1@alpaca.markets",
		"test-integration1",
		&s.id,
	)
	client.SetBaseURL("http://nginxsvc")

	var created *apiclient.AccessKey
	// Create Access Key
	{
		accessKey, err := client.CreateAccessKey(s.apiKey.AccountID)
		assert.Nil(s.T(), err)
		assert.NotEmpty(s.T(), accessKey.ID)
		assert.NotEmpty(s.T(), accessKey.Secret)
		created = accessKey
	}

	// List Access Key
	{
		accessKeys, err := client.ListAccessKeys()
		assert.Nil(s.T(), err)
		assert.Len(s.T(), accessKeys, 2)
	}

	// Delete Access Key
	{
		err := client.DeleteAccessKey(created.ID)
		assert.Nil(s.T(), err)

		accessKeys, err := client.ListAccessKeys()
		assert.Nil(s.T(), err)
		assert.Len(s.T(), accessKeys, 1)
	}

	//
	// owner details
	//

	// Get
	{
		_, err := client.GetOwnerDetails(s.apiKey.AccountID)
		assert.Nil(s.T(), err)
	}

	// Patch
	{
		err := client.PatchOwnerDetails(s.apiKey.AccountID, map[string]interface{}{"state": "CA"})
		assert.Nil(s.T(), err)
		od, err := client.GetOwnerDetails(s.apiKey.AccountID)
		assert.Equal(s.T(), *od.State, "CA")
	}

	//
	// affiliate
	//

	// List
	{
		afs, err := client.GetAffiliates(s.apiKey.AccountID)
		assert.Nil(s.T(), err)
		assert.Equal(s.T(), len(afs), 0)
	}

	// Create
	{
		params := map[string]interface{}{
			"street_address":   []string{"test street address", "test"},
			"city":             "San Mateo",
			"state":            "CA",
			"postal_code":      "00000",
			"country":          "USA",
			"company_name":     "Alpaca",
			"compliance_email": "test@alpaca.markets",
			"type":             enum.FinraFirm,
		}
		_, err := client.CreateAffiliate(s.apiKey.AccountID, params)
		assert.Nil(s.T(), err)
	}

	// Patch
	{
		afs, _ := client.GetAffiliates(s.apiKey.AccountID)
		assert.Equal(s.T(), len(afs), 1)

		params := map[string]interface{}{
			"postal_code": "11111",
		}
		_, err := client.PatchAffiliate(s.apiKey.AccountID, afs[0].ID, params)
		assert.Nil(s.T(), err)

		afs, _ = client.GetAffiliates(s.apiKey.AccountID)
		assert.Equal(s.T(), len(afs), 1)
		assert.Equal(s.T(), afs[0].PostalCode, "11111")
	}

	// Delete
	{
		afs, _ := client.GetAffiliates(s.apiKey.AccountID)
		assert.Equal(s.T(), len(afs), 1)

		err := client.DeleteAffiliate(s.apiKey.AccountID, afs[0].ID)
		assert.Nil(s.T(), err)

		afs, _ = client.GetAffiliates(s.apiKey.AccountID)
		assert.Equal(s.T(), len(afs), 0)
	}

	//
	// trusted contact
	//

	// Get
	{
		_, err := client.GetTrustedContact(s.apiKey.AccountID)
		assert.NotNil(s.T(), err)
		assert.Equal(s.T(), err.(*apiclient.ApiError).Code, 40410000)
	}

	// Create
	{
		params := map[string]interface{}{
			"given_name":    "x",
			"family_name":   "y",
			"email_address": "test@alpaca.markets",
		}
		_, err := client.CreateTrustedContact(s.apiKey.AccountID, params)
		assert.Nil(s.T(), err)

		tc, err := client.GetTrustedContact(s.apiKey.AccountID)
		assert.Nil(s.T(), err)
		assert.Equal(s.T(), tc.GivenName, "x")
	}

	// Patch
	{
		params := map[string]interface{}{
			"given_name": "z",
		}
		_, err := client.PatchTrustedContact(s.apiKey.AccountID, params)
		assert.Nil(s.T(), err)

		tc, err := client.GetTrustedContact(s.apiKey.AccountID)
		assert.Nil(s.T(), err)
		assert.Equal(s.T(), tc.GivenName, "z")
	}

	// Delete
	{
		err := client.DeleteTrustedContact(s.apiKey.AccountID)
		assert.Nil(s.T(), err)

		_, err = client.GetTrustedContact(s.apiKey.AccountID)
		assert.NotNil(s.T(), err)
		assert.Equal(s.T(), err.(*apiclient.ApiError).Code, 40410000)
	}

	// Account

	// Create
	{
		nc := apiclient.NewInternalClient(
			"integration-test-new1@alpaca.markets",
			"test-integration1",
			nil,
		)
		nc.SetBaseURL("http://nginxsvc")

		_, err := nc.CreateAccount()
		assert.Nil(s.T(), err)

		// test duplicate
		_, err = nc.CreateAccount()
		require.NotNil(s.T(), err)
		assert.Equal(s.T(), err.Error(), "duplicate email")
	}

	// Waiting for automatic account approve from account worker. 5 sec + buffer
	time.Sleep(7 * time.Second)

	s.T().Logf("TestInternal complete [%v]", clock.Now().Sub(start))

	// Profit Loss
	{
		_, err := client.GetProfitLoss(s.apiKey.AccountID)
		assert.Nil(s.T(), err)
	}

	// intra API
	// asset
	{
		nc := apiclient.NewPTClient()
		nc.SetBaseURL("http://nginxsvc")
		assets, _ := nc.ListAssets()
		assert.Len(s.T(), assets, 4)
	}

}
