package httplogger

import (
	"io/ioutil"
	"os"

	"github.com/buger/jsonparser"

	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/env"
	"github.com/alpacahq/gopaca/fluentlogger"
	"github.com/alpacahq/gopaca/log"
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

type HTTPLogger struct {
	logger *fluentlogger.FluentLogger
}

func New() iris.Handler {
	m := HTTPLogger{
		logger: fluentlogger.Logger(),
	}
	return m.ServeHTTP
}

var masks = []string{
	"password",
	"ssn",
}

func (h *HTTPLogger) ServeHTTP(ctx context.Context) {
	start := clock.Now()
	ctx.Next()
	end := clock.Now()

	var (
		err     error
		service string
		body    []byte
	)

	if podName := env.GetVar("KUBERNETES_POD_NAME"); podName == "" {
		service = podName
	} else {
		service = os.Args[0]
	}

	// mask the sensitive fields
	if body, _ = ioutil.ReadAll(ctx.Request().Body); len(body) > 0 {
		for _, mask := range masks {
			if _, _, _, err = jsonparser.Get(body, mask); err == nil {
				body, _ = jsonparser.Set(body, []byte("xxx"), mask)
			}
		}
	}

	msg := map[string]interface{}{
		"service":     service,
		"node":        env.GetVar("KUBERNETES_NODE_NAME"),
		"deployment":  env.GetVar("BROKER_MODE"),
		"elapsed":     end.Sub(start).Seconds(),
		"status_code": ctx.GetStatusCode(),
		"ip":          ctx.RemoteAddr(),
		"method":      ctx.Method(),
		"path":        ctx.Path(),
		"query":       ctx.Request().URL.RawQuery,
		"key_id":      ctx.Request().Header.Get("APCA-API-KEY-ID"),
		"acc_id":      ctx.Values().GetString("account_id"),
		"body":        string(body),
	}

	h.logger.Post("alpaca.httplog", msg)

	log.Debug("httplog",
		"method", msg["method"],
		"path", msg["path"],
		"query", msg["query"],
		"status_code", msg["status_code"],
		"elapsed", msg["elapsed"],
		"ip", msg["ip"],
		"key_id", msg["key_id"],
		"acc_id", msg["acc_id"],
		"body", msg["body"],
	)
}
