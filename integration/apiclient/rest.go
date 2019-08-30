package apiclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/rest/api/controller/order"
	"github.com/google/go-querystring/query"
	"github.com/shopspring/decimal"
	"github.com/valyala/fasthttp"
	"github.com/vmihailenco/msgpack"
)

type ApiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *ApiError) Error() string {
	return e.Message
}

type RestClient struct {
	apiKeyId     string
	apiSecretKey string
	baseUrl      string
	client       *fasthttp.Client

	RequestHeaders map[string]string
	setHeaderFunc  func(req *fasthttp.Request)
}

func NewRestClient(keyId, secretKey string) *RestClient {
	c := &RestClient{
		apiKeyId:     keyId,
		apiSecretKey: secretKey,
		baseUrl:      "https://api.alpaca.markets",
		client:       &fasthttp.Client{},
	}
	c.setHeaderFunc = c.setHeader
	return c
}

func (c *RestClient) SetBaseURL(url string) *RestClient {
	c.baseUrl = url
	return c
}

type Account struct {
	ID                   string          `json:"id"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
	DeletedAt            *time.Time      `json:"deleted_at"`
	Status               string          `json:"status"`
	Currency             string          `json:"currency"`
	Cash                 decimal.Decimal `json:"cash"`
	CashWithdrawable     decimal.Decimal `json:"cash_withdrawable"`
	TradingBlocked       bool            `json:"trading_blocked"`
	TransfersBlocked     bool            `json:"transfers_blocked"`
	AccountBlocked       bool            `json:"account_blocked"`
	TradeSuspendedByUser bool            `json:"trade_suspended_by_user"`
}

type Order struct {
	ID            string           `json:"id"`
	ClientOrderID string           `json:"client_order_id"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
	SubmittedAt   time.Time        `json:"submitted_at"`
	FilledAt      *time.Time       `json:"filled_at"`
	ExpiredAt     *time.Time       `json:"expired_at"`
	CanceledAt    *time.Time       `json:"canceled_at"`
	FailedAt      *time.Time       `json:"failed_at"`
	AssetID       string           `json:"asset_id"`
	Symbol        string           `json:"symbol"`
	Exchange      string           `json:"exchange"`
	Class         string           `json:"asset_class"`
	Qty           decimal.Decimal  `json:"qty"`
	Type          string           `json:"type"`
	Side          string           `json:"side"`
	TimeInForce   string           `json:"time_in_force"`
	LimitPrice    *decimal.Decimal `json:"limit_price"`
	StopPrice     *decimal.Decimal `json:"stop_price"`
	Status        string           `json:"status"`
}

type Position struct {
	AssetID    string          `json:"asset_id"`
	Symbol     string          `json:"symbol"`
	Exchange   string          `json:"exchange"`
	Class      string          `json:"asset_class"`
	AccountID  string          `json:"account_id"`
	EntryPrice decimal.Decimal `json:"entry_price"`
	Qty        decimal.Decimal `json:"qty"`
	Side       string          `json:"side"`
}

type Asset struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Exchange string `json:"exchange"`
	Class    string `json:"asset_class"`
	Symbol   string `json:"symbol"`
	Status   string `json:"status"`
	Tradable bool   `json:"tradable"`
}

type Fundamental struct {
	AssetID           string          `json:"asset_id"`
	Symbol            string          `json:"symbol"`
	FullName          string          `json:"full_name"`
	IndustryName      string          `json:"industry_name"`
	IndustryGroup     string          `json:"industry_group"`
	Sector            string          `json:"sector"`
	PERatio           float32         `json:"pe_ratio"`
	PEGRatio          float32         `json:"peg_ratio"`
	Beta              float32         `json:"beta"`
	EPS               float32         `json:"eps"`
	MarketCap         int64           `json:"market_cap"`
	SharesOutstanding int64           `json:"shares_outstanding"`
	AvgVol            int64           `json:"avg_vol"`
	DivRate           float32         `json:"div_rate"`
	ROE               float32         `json:"roe"`
	ROA               float32         `json:"roa"`
	PS                float32         `json:"ps"`
	PC                float32         `json:"pc"`
	GrossMargin       float32         `json:"gross_margin"`
	FiftyTwoWeekHigh  decimal.Decimal `json:"fifty_two_week_high"`
	FiftyTwoWeekLow   decimal.Decimal `json:"fifty_two_week_low"`
	ShortDescription  string          `json:"short_description"`
	LongDescription   string          `json:"long_description"`
}

type BarList struct {
	AssetID  string `json:"asset_id"`
	Symbol   string `json:"symbol"`
	Exchange string `json:"exchange"`
	Class    string `json:"asset_class"`
	Bars     []*Bar `json:"bars"`
}

type Bar struct {
	Open   float32   `json:"open"`
	High   float32   `json:"high"`
	Low    float32   `json:"low"`
	Close  float32   `json:"close"`
	Volume int32     `json:"volume"`
	Time   time.Time `json:"time"`
}

type BarListParams struct {
	Timeframe string     `url:"timeframe,omitempty"`
	StartDt   *time.Time `url:"start_dt,omitempty"`
	EndDt     *time.Time `url:"end_dt,omitempty"`
	Limit     *int       `url:"limit,omitempty"`
}

type Quote struct {
	BidTimestamp  time.Time `json:"bid_timestamp"`
	Bid           float32   `json:"bid"`
	AskTimestamp  time.Time `json:"ask_timestamp"`
	Ask           float32   `json:"ask"`
	LastTimestamp time.Time `json:"last_timestamp"`
	Last          float32   `json:"last"`
	AssetID       string    `json:"asset_id"`
	Symbol        string    `json:"symbol"`
	Class         string    `json:"asset_class"`
}

type CalendarDay struct {
	Date  string `json:"date"`
	Open  string `json:"open"`
	Close string `json:"close"`
}

type Clock struct {
	Timestamp time.Time `json:"timestamp"`
	IsOpen    bool      `json:"is_open"`
	NextOpen  time.Time `json:"next_open"`
	NextClose time.Time `json:"next_close"`
}

type PatchConfigurationsRequest map[string]interface{}

func (c *RestClient) GetAccount() (res *Account, err error) {
	url := fmt.Sprintf("%s/api/v1/account", c.baseUrl)
	_, err = c.call(url, "GET", nil, &res)
	return
}

func (c *RestClient) PatchConfigurations(req PatchConfigurationsRequest) (err error) {
	url := fmt.Sprintf("%s/api/v1/account/configurations", c.baseUrl)
	_, err = c.call(url, "PATCH", &req, nil)
	return
}

func (c *RestClient) ListPositions() (res []*Position, err error) {
	url := fmt.Sprintf("%s/api/v1/positions", c.baseUrl)
	_, err = c.call(url, "GET", nil, &res)
	return
}

func (c *RestClient) GetClock() (res *Clock, err error) {
	url := fmt.Sprintf("%s/api/v1/clock", c.baseUrl)
	_, err = c.call(url, "GET", nil, &res)
	return
}

func (c *RestClient) GetCalendar() (res []*CalendarDay, err error) {
	url := fmt.Sprintf("%s/api/v1/calendar", c.baseUrl)
	_, err = c.call(url, "GET", nil, &res)
	return
}

func (c *RestClient) ListOrders(status string) (res []*Order, err error) {
	url := fmt.Sprintf("%s/api/v1/orders", c.baseUrl)
	url += fmt.Sprintf("?status=%s", status)
	_, err = c.call(url, "GET", nil, &res)
	return
}

func (c *RestClient) PlaceOrder(req order.CreateOrderRequest) (res *Order, err error) {
	url := fmt.Sprintf("%s/api/v1/orders", c.baseUrl)
	_, err = c.call(url, "POST", req, &res)
	return
}

func (c *RestClient) CancelOrder(orderID string) (err error) {
	url := fmt.Sprintf("%s/api/v1/orders/%v", c.baseUrl, orderID)
	_, err = c.call(url, "DELETE", nil, nil)
	return
}

func (c *RestClient) ListAssets() (res []*Asset, err error) {
	url := fmt.Sprintf("%v/api/v1/assets", c.baseUrl)
	_, err = c.call(url, "GET", nil, &res)
	return
}

func (c *RestClient) GetAsset(symbol string) (res *Asset, err error) {
	url := fmt.Sprintf("%v/api/v1/assets/%s", c.baseUrl, symbol)
	_, err = c.call(url, "GET", nil, &res)
	return
}

func (c *RestClient) ListFundamentals(symbols []string) (res []*Fundamental, err error) {
	vals := url.Values{}
	vals.Add("symbols", strings.Join(symbols, ","))
	url := fmt.Sprintf("%v/api/v1/fundamentals?%v", c.baseUrl, vals.Encode())
	_, err = c.call(url, "GET", nil, &res)
	return
}

func (c *RestClient) GetFundamental(symbol string) (res *Fundamental, err error) {
	url := fmt.Sprintf("%v/api/v1/assets/%s/fundamental", c.baseUrl, symbol)
	_, err = c.call(url, "GET", nil, &res)
	return
}

func (c *RestClient) ListBarLists(symbols []string, opts BarListParams) (res []*BarList, err error) {
	vals, _ := query.Values(opts)
	vals.Add("symbols", strings.Join(symbols, ","))
	url := fmt.Sprintf("%v/api/v1/bars?%v", c.baseUrl, vals.Encode())
	_, err = c.call(url, "GET", nil, &res)
	return
}

func (c *RestClient) GetBarList(symbol string, opts BarListParams) (res *BarList, err error) {
	vals, _ := query.Values(opts)
	url := fmt.Sprintf("%v/api/v1/assets/%s/bars?%v", c.baseUrl, symbol, vals.Encode())
	_, err = c.call(url, "GET", nil, &res)
	return
}

func (c *RestClient) ListQuotes(symbols []string) (res []*Quote, err error) {
	vals := url.Values{}
	vals.Add("symbols", strings.Join(symbols, ","))
	url := fmt.Sprintf("%v/api/v1/quotes?%v", c.baseUrl, vals.Encode())
	_, err = c.call(url, "GET", nil, &res)
	return
}

func (c *RestClient) GetQuote(symbol string) (res *Quote, err error) {
	url := fmt.Sprintf("%v/api/v1/assets/%s/quote", c.baseUrl, symbol)
	_, err = c.call(url, "GET", nil, &res)
	return
}

func (c *RestClient) setHeader(req *fasthttp.Request) {
	req.Header.Add("APCA-API-KEY-ID", c.apiKeyId)
	req.Header.Add("APCA-API-SECRET-KEY", c.apiSecretKey)
	req.Header.Add("APCA-TEST-CLIENT", "bypass")
	// Additional headers to add (ie. Content-Type) which can be set per request/test
	for k, v := range c.RequestHeaders {
		req.Header.Add(k, v)
	}
}

func (c *RestClient) call(uri, method string, reqBody, resBody interface{}) (resp *fasthttp.Response, err error) {
	req := fasthttp.AcquireRequest()
	c.setHeaderFunc(req)
	req.SetRequestURI(uri)
	req.Header.SetMethod(method)
	contentType := strings.ToLower(string(req.Header.ContentType()))
	if method != "GET" && reqBody != nil {
		switch contentType {
		case api.MIMEApplicationMsgpack, api.MIMEApplicationMsgpackCharsetUTF8:
			var reqBytes bytes.Buffer
			enc := msgpack.NewEncoder(&reqBytes)
			// Using json tags on structs
			enc.UseJSONTag(true)
			err := enc.Encode(reqBytes)

			if err != nil {
				log.Printf(
					"Failed to MsgPack marshal for %v (%v): %v - Error: %v\n",
					uri,
					method,
					reqBody,
					err,
				)
				return nil, err
			}
			req.SetBody(reqBytes.Bytes())
		// case api.MIMEApplicationJSON, api.MIMEApplicationJSONCharsetUTF8:
		default:
			reqBytes, err := json.Marshal(reqBody)
			if err != nil {
				log.Printf(
					"Failed to JSON marshal for %v (%v): %v - Error: %v\n",
					uri,
					method,
					reqBody,
					err,
				)
				return nil, err
			}
			req.SetBody(reqBytes)
		}
	}
	resp = fasthttp.AcquireResponse()
	c.client.Do(req, resp)
	resBytes, err := getResponseBody(resp)
	if resp.StatusCode() >= fasthttp.StatusMultipleChoices {
		apiErr := ApiError{}

		switch strings.ToLower(string(resp.Header.ContentType())) {
		case api.MIMEApplicationMsgpack, api.MIMEApplicationMsgpackCharsetUTF8:
			dec := msgpack.NewDecoder(bytes.NewReader(resBytes))
			// Using json tags on structs
			dec.UseJSONTag(true)
			err = dec.Decode(&apiErr)
		// case api.MIMEApplicationJSON, api.MIMEApplicationJSONCharsetUTF8:
		default:
			err = json.Unmarshal(resBytes, &apiErr)
		}

		if err == nil {
			err = &apiErr
			return
		} else {
			err = fmt.Errorf(
				"failed to %v %v - Error: %v - Status: %d, Response: %v",
				method,
				uri,
				err,
				resp.StatusCode(),
				string(resBytes),
			)
			log.Printf("%v\n", err.Error())
		}
		return
	}

	if resBody != nil {
		switch strings.ToLower(string(resp.Header.ContentType())) {
		case api.MIMEApplicationMsgpack, api.MIMEApplicationMsgpackCharsetUTF8:
			dec := msgpack.NewDecoder(bytes.NewReader(resBytes))
			// Using json tags on structs
			dec.UseJSONTag(true)
			err = dec.Decode(&resBody)
		// case api.MIMEApplicationJSON, api.MIMEApplicationJSONCharsetUTF8:
		default:
			err = json.Unmarshal(resBytes, &resBody)
		}

		if err != nil {
			log.Printf(
				"Failed to unmarshal response from %v - Error: %v - Response: %v\n",
				uri,
				err,
				string(resBytes),
			)
			return
		}
	}

	return resp, err
}

func getResponseBody(resp *fasthttp.Response) ([]byte, error) {
	switch string(resp.Header.Peek("Content-encoding")) {
	case "gzip":
		body, err := resp.BodyGunzip()
		if err != nil {
			return nil, err
		}
		return body, nil
	case "inflate":
		body, err := resp.BodyInflate()
		if err != nil {
			return nil, err
		}
		return body, nil
	default:
		return resp.Body(), nil
	}
}
