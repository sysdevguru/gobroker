package apex

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/alpacahq/apex/http"
	"github.com/valyala/fasthttp"
	"gopkg.in/matryer/try.v1"
)

var (
	once           sync.Once
	ac             *Apex
	defaultTimeout = 10 * time.Second
)

type Apex struct {
	request func(req *fasthttp.Request, resp *fasthttp.Response, timeout ...time.Duration) error
	JWT     string
	Dev     bool
}

func newClient() (*Apex, error) {
	a := &Apex{Dev: os.Getenv("BROKER_MODE") == "DEV"}

	a.request = func(req *fasthttp.Request, resp *fasthttp.Response, timeout ...time.Duration) (err error) {

		if len(a.JWT) > 0 {
			req.Header.Set("Authorization", a.JWT)
		}

		if len(req.Header.ContentType()) == 0 {
			req.Header.SetContentType("application/json")
		}

		if err = try.Do(func(attempt int) (bool, error) {
			if len(timeout) > 0 {
				err = fasthttp.DoTimeout(req, resp, timeout[0])
			} else {
				err = fasthttp.DoTimeout(req, resp, defaultTimeout)
			}

			// apex's API's are not the most reliable, and sometimes
			// either the call just fails, or their nginx times out
			// w/ 504 on GET requests - this is to eliminate some of
			// the noise.
			retry := err != nil || (resp.StatusCode() == fasthttp.StatusGatewayTimeout &&
				string(req.Header.Method()) == "GET")

			return attempt < 3 && retry, err
		}); err != nil {
			return
		}

		if resp.StatusCode() == fasthttp.StatusUnauthorized {
			if err = a.Authenticate(); err == nil {
				return a.request(req, resp)
			}
		}

		return
	}

	return a, a.Authenticate()
}

// Anywhere env. variables are used if any of those are not set when once.Do() is called,
// panic(Apex environment not set) with
func checkEnv() error {
	if os.Getenv("BROKER_MODE") != "DEV" {
		switch "" {
		case os.Getenv("APEX_URL"):
			return fmt.Errorf("APEX_URL not set")
		case os.Getenv("APEX_CORRESPONDENT_CODE"):
			return fmt.Errorf("APEX_CORRESPONDENT_CODE not set")
		case os.Getenv("APEX_CUSTOMER_ID"):
			return fmt.Errorf("APEX_CUSTOMER_ID not set")
		case os.Getenv("APEX_FIRM_CODE"):
			return fmt.Errorf("APEX_FIRM_CODE not set")
		case os.Getenv("APEX_WS_URL"):
			return fmt.Errorf("APEX_WS_URL not set")
		case os.Getenv("APEX_SECRET"):
			return fmt.Errorf("APEX_SECRET not set")
		case os.Getenv("APEX_USER"):
			return fmt.Errorf("APEX_USER not set")
		case os.Getenv("APEX_ENTITY"):
			return fmt.Errorf("APEX_ENTITY not set")
		case os.Getenv("APEX_ENCRYPTION_KEY"):
			return fmt.Errorf("APEX_ENCRYPTION_KEY not set")
		}
	}

	return nil
}

func Client() *Apex {
	once.Do(func() {
		var err error

		if err = checkEnv(); err != nil {
			panic(fmt.Errorf("apex environment is invalid (%v)", err))
		}

		ac, err = newClient()
		if err != nil {
			panic(fmt.Errorf("failed to start apex client (%v)", err))
		}
	})
	return ac
}

// getJSON is a shortcut to call() for GET JSON body.
func (a *Apex) getJSON(uri string, resBody interface{}) (resp *fasthttp.Response, err error) {
	return a.call(uri, "GET", nil, resBody)
}

func (a *Apex) Call(uri, method string, reqBody, resBody interface{}) error {
	_, err := a.call(uri, method, reqBody, resBody)
	return err
}

// call calls Apex API at the endpoint uri with method. Optional reqBody (nil-able)
// sets request body in JSON if supplied and method is not GET.  resBody (nil-able)
// is set to unmarshalled struct from JSON response, unless it is a type of *[]byte
// in which case the raw body bytes are returned.
func (a *Apex) call(uri, method string, reqBody, resBody interface{}, timeout ...time.Duration) (resp *fasthttp.Response, err error) {
	req := fasthttp.AcquireRequest()
	req.SetRequestURI(uri)
	req.Header.SetMethod(method)

	if method != "GET" && reqBody != nil {
		reqBytes, err := json.Marshal(reqBody)
		if err != nil {
			return resp, err
		}
		req.SetBody(reqBytes)
	}

	resp = fasthttp.AcquireResponse()

	if err = a.request(req, resp, timeout...); err != nil {
		return
	}

	resBytes, err := http.GetResponseBody(resp)
	if err != nil {
		return
	}

	if resp.StatusCode() >= fasthttp.StatusMultipleChoices {
		err = fmt.Errorf(
			"request: (%s %s) response: (%v %s)",
			method, uri, resp.StatusCode(), string(resBytes))
		return
	}

	if resBody != nil {
		if resBodyBytes, ok := resBody.(*[]byte); ok {
			*resBodyBytes = resBytes
		} else {
			if err = json.Unmarshal(resBytes, &resBody); err != nil {
				return
			}
		}
	}

	return
}
