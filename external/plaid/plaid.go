package plaid

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/alpacahq/gopaca/env"
	"github.com/pkg/errors"
	try "gopkg.in/matryer/try.v1"

	"github.com/alpacahq/gopaca/log"
	"github.com/valyala/fasthttp"
)

var (
	once sync.Once
	pc   *PlaidClient
)

const (
	CodeItemLoginRequired = "ITEM_LOGIN_REQUIRED"
)

const timeout = time.Minute

type PlaidClient struct {
	request func(req *fasthttp.Request, resp *fasthttp.Response) error
}

type APIError struct {
	DisplayMessage *string `json:"display_message"`
	ErrorCode      string  `json:"error_code"`
	ErrorMessage   string  `json:"error_message"`
	ErrorType      string  `json:"error_type"`
	RequestID      string  `json:"request_id"`
}

func (e *APIError) CanDisplay() bool {
	return e.DisplayMessage != nil
}

func (e APIError) Error() string {
	return fmt.Sprintf("%v (type: %v code: %v request: %v)",
		e.ErrorMessage, e.ErrorType, e.ErrorCode, e.RequestID)
}

func newClient() *PlaidClient {
	p := &PlaidClient{}
	request := func(req *fasthttp.Request, resp *fasthttp.Response) (err error) {
		req.Header.SetContentType("application/json")
		if err = try.Do(func(attempt int) (bool, error) {
			err = fasthttp.DoTimeout(req, resp, timeout)
			return err != nil, err
		}); err != nil {
			return
		}
		return
	}
	p.request = request
	return p
}

func Client() *PlaidClient {
	once.Do(func() {
		pc = newClient()
		if pc == nil {
			log.Fatal("failed to start plaid client")
		}
	})
	return pc
}

func (pc *PlaidClient) Request(method, endpoint string, payload map[string]interface{}, output interface{}) error {
	req := fasthttp.AcquireRequest()
	req.Header.SetMethod(method)
	req.SetRequestURI(env.GetVar("PLAID_URL") + endpoint)

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req.SetBody(body)
	resp := fasthttp.AcquireResponse()

	if err = pc.request(req, resp); err != nil {
		return err
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		apiError := APIError{}

		if err := json.Unmarshal(resp.Body(), &apiError); err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to parse error (status_code = %v)", resp.StatusCode()))
		}

		return apiError
	}

	return json.Unmarshal(resp.Body(), output)
}
