package ddworker

import (
	"os"
	"sync"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/alpacahq/gopaca/log"
)

type Handler func(*statsd.Client) error

type item struct {
	handler  Handler
	interval time.Duration
	name     string
}

var handlers []item
var once sync.Once
var dd *statsd.Client

func RegisterHandler(handler Handler, name string, interval time.Duration) {
	handlers = append(handlers, item{handler: handler, name: name, interval: interval})
}

func client() *statsd.Client {
	once.Do(func() {
		ip := os.Getenv("DOGSTATSD_HOST_IP")
		if ip == "" {
			log.Warn("won't send log to statsd (DOGSTATSD_HOST_IP envvar not found)")
			return
		}

		var err error
		dd, err = statsd.New(ip + ":8125")
		if err != nil {
			log.Warn("won't send log to statsd (failed to init statsd)")
			return
		}
	})

	return dd
}

func SetNamespace(ns string) {
	if cli := client(); cli != nil {
		cli.Namespace = ns
	}
}

func RunWorker() {
	tickers := map[time.Duration]*time.Ticker{}
	wg := sync.WaitGroup{}

	for i := range handlers {
		handler := handlers[i]
		if _, ok := tickers[handler.interval]; !ok {
			ticker := time.NewTicker(handler.interval)
			tickers[handler.interval] = ticker
			wg.Add(1)
			go func(ticker *time.Ticker, interval time.Duration) {
				handleTicker(ticker, interval)
				wg.Done()
			}(ticker, handler.interval)
		}
	}

	wg.Wait()
}

func handleTicker(ticker *time.Ticker, interval time.Duration) {
	for _ = range ticker.C {
		for _, handler := range handlers {
			if handler.interval == interval {
				cli := client()

				if cli == nil {
					continue
				}

				if err := handler.handler(cli); err != nil {
					log.Error("failed to call handler", "error", err)
				}
			}
		}
	}
}
