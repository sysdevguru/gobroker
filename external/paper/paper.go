package paper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gopaca/env"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	try "gopkg.in/matryer/try.v1"
)

type Client struct {
	baseURL     string
	requestFunc func(req *fasthttp.Request, resp *fasthttp.Response) (err error)
}

type APIError struct {
	Message    string
	StatusCode int
	Code       int
	Debug      string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%v (%v)", e.Message, e.Code)
}

func NewClient() *Client {
	host := os.Getenv("PAPERTRADER_SERVICE_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	port := os.Getenv("PAPERTRADER_SERVICE_PORT")
	if port == "" {
		port = "5998"
	}

	request := func(req *fasthttp.Request, resp *fasthttp.Response) (err error) {
		req.Header.SetContentType("application/json")
		req.Header.Set("X-PT-SECRET", env.GetVar("PT_SECRET"))
		if err = try.Do(func(attempt int) (bool, error) {
			err = fasthttp.DoTimeout(req, resp, time.Second*3)
			return err != nil, err
		}); err != nil {
			return
		}
		return
	}

	return &Client{
		baseURL:     fmt.Sprintf("http://%s:%s%s", host, port, "/papertrader/api"),
		requestFunc: request,
	}
}

func (c *Client) request(method, endpoint string, payload map[string]interface{}, output interface{}) error {
	req := fasthttp.AcquireRequest()
	req.Header.SetMethod(method)

	req.SetRequestURI(c.baseURL + endpoint)

	if payload != nil {
		body, err := json.Marshal(payload)
		if err != nil {
			return errors.Wrap(err, "failed to marshal body")
		}
		req.SetBody(body)
	}

	resp := fasthttp.AcquireResponse()

	if err := c.requestFunc(req, resp); err != nil {
		return errors.Wrap(err, "failed request func")
	}

	if resp.StatusCode() > 300 {
		var apiError APIError

		if err := json.Unmarshal(resp.Body(), &apiError); err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to parse error (status_code = %v)", resp.StatusCode()))
		}
		apiError.StatusCode = resp.StatusCode()

		return &apiError
	}

	if resp.StatusCode() == http.StatusNoContent {
		return nil
	}

	return json.Unmarshal(resp.Body(), output)
}

func (c *Client) Proxy(ctx api.Context) error {
	req := fasthttp.AcquireRequest()
	req.Header.SetMethod(ctx.Method())
	req.SetRequestURI(c.paperURL(ctx.Request().URL))

	var body []byte

	if _, err := ctx.Request().Body.Read(body); err != nil && err != io.EOF {
		return errors.Wrap(err, "failed to write body")
	} else {
		req.SetBody(body)
	}

	resp := fasthttp.AcquireResponse()

	if err := c.requestFunc(req, resp); err != nil {
		return errors.Wrap(err, "failed request func")
	}

	if resp.StatusCode() > 300 {
		var apiError APIError

		if err := json.Unmarshal(resp.Body(), &apiError); err != nil {
			return gberrors.InternalServerError.WithError(
				errors.Wrap(err, fmt.Sprintf("failed to parse error (status_code = %v)", resp.StatusCode())))
		}
		apiError.StatusCode = resp.StatusCode()

		return &apiError
	}

	ctx.StatusCode(resp.StatusCode())
	ctx.RespondWithContent(string(resp.Header.ContentType()), resp.Body())

	return nil
}

func (c *Client) paperURL(uri *url.URL) string {
	urlParts := strings.Split(uri.Path, "paper_accounts")
	u, _ := url.Parse(c.baseURL)
	u.Path = path.Join(u.Path, strings.Replace(urlParts[0], "/gobroker/api", "", 1), "accounts", urlParts[1])
	u.RawQuery = uri.RawQuery

	return u.String()
}
