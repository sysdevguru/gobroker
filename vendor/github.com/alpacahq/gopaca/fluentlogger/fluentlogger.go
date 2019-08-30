package fluentlogger

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"

	"github.com/alpacahq/gopaca/env"
	"github.com/alpacahq/gopaca/log"
	"github.com/fluent/fluent-logger-golang/fluent"
)

type FluentLogger struct {
	fluent *fluent.Fluent
}

var once sync.Once
var logger *FluentLogger

func Logger() *FluentLogger {
	once.Do(func() {
		var err error
		logger, err = NewLogger()
		if err != nil {
			panic(err)
		}
	})
	return logger
}

func NewLogger() (*FluentLogger, error) {

	fluentHost := env.GetVar("FLUENTD_HOST")
	fluentPort := env.GetVar("FLUENTD_PORT")

	if fluentHost != "" && fluentPort != "" {
		port, err := strconv.Atoi(fluentPort)
		if err != nil {
			return nil, fmt.Errorf("failed to init logger : invalid fluentd port config %v", err)
		}

		fl, err := fluent.New(fluent.Config{
			FluentHost: fluentHost,
			FluentPort: port,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to init logger : failed to init fluent logger %v", err)
		}

		return &FluentLogger{fluent: fl}, nil
	}
	return nil, fmt.Errorf("FLUENTD_HOST or FLUENTD_PORT env var is missing")
}

func (fl *FluentLogger) Post(tag string, message interface{}) {
	if err := fl.fluent.Post(tag, message); err != nil {
		msg, _ := json.Marshal(message)
		log.Error("failed to post fluentd", "tag", tag, "message", string(msg), "error", err)
	}
}
