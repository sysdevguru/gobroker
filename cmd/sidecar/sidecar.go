package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/alpacahq/gobroker/metrics"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/ddworker"
	"github.com/alpacahq/gopaca/log"
)

const (
	performanceTag = "performance"
	marketDataTag  = "marketData"
)

var (
	port = func() (p string) {
		p = os.Getenv("BROKER_METRICS_PORT")
		if p == "" {
			p = "7777"
		}
		return
	}()
)

func metricsHandler(dd *statsd.Client) error {

	// market data metrics
	{
		if calendar.IsMarketOpen(time.Now()) {
			mktMetrics, err := getMarketDataMetrics()
			if err != nil {
				return err
			}

			// staleness and latency
			dd.Timing("quote_latency", mktMetrics.QuoteLatency, []string{marketDataTag}, 1)
			dd.Timing("bar_latency", mktMetrics.BarLatency, []string{marketDataTag}, 1)
			dd.Timing("oldest_quote", mktMetrics.OldestQuote.Age, []string{marketDataTag}, 1)
			dd.Timing("oldest_bar", mktMetrics.OldestBar.Age, []string{marketDataTag}, 1)

			// errors
			if mktMetrics.QuoteError != nil {
				dd.SimpleEvent("quote error", mktMetrics.QuoteError.Error())
			}
			if mktMetrics.BarError != nil {
				dd.SimpleEvent("bar error", mktMetrics.BarError.Error())
			}
		}
	}

	// performance metrics
	{
		perfMetrics, err := getPerformanceMetrics()
		if err != nil {
			return err
		}

		dd.Gauge("cpu_usage", perfMetrics.CPUUsagePercent, nil, 1)
		dd.Gauge("mem_usage", perfMetrics.MemoryUsagePercent, nil, 1)
		dd.Count("goroutines", perfMetrics.GoRoutines, nil, 1)
		dd.Timing("db_latency", perfMetrics.DatabaseLatency, nil, 1)
	}

	return nil
}

func getMarketDataMetrics() (*metrics.MarketDataMetrics, error) {
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%v/metrics/marketdata", port))
	if err != nil {
		return nil, err
	}

	m := &metrics.MarketDataMetrics{}

	if err := json.NewDecoder(resp.Body).Decode(m); err != nil {
		return nil, fmt.Errorf("failed to parse market data metrics %v", err)
	}

	return m, nil
}

func getPerformanceMetrics() (*metrics.PerformanceMetrics, error) {
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%v/metrics/performance", port))
	if err != nil {
		return nil, err
	}

	m := &metrics.PerformanceMetrics{}

	if err := json.NewDecoder(resp.Body).Decode(m); err != nil {
		return nil, fmt.Errorf("failed to parse market data metrics %v", err)
	}

	return m, nil
}

func init() {
	ddworker.RegisterHandler(metricsHandler, "metrics_handler", time.Second*10)
	ddworker.SetNamespace("gobroker.")
}

func main() {
	log.Info("running gobroker sidecar container")
	ddworker.RunWorker()
}
