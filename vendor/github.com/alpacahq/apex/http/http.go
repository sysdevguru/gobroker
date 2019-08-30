package http

import (
	"github.com/valyala/fasthttp"
)

func GetResponseBody(resp *fasthttp.Response) ([]byte, error) {
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
