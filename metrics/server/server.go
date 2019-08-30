package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/alpacahq/gobroker/metrics"
	"github.com/alpacahq/gopaca/env"
	"github.com/alpacahq/gopaca/log"
)

func marketDataMetricsHandler(w http.ResponseWriter, r *http.Request) {
	mktMetrics, err := metrics.GetMarketDataMetrics()
	if err != nil {
		log.Error("failed to retrieve market data metrics", "error", err)
		return
	}

	json.NewEncoder(w).Encode(mktMetrics)
}

func performanceMetricsHandler(w http.ResponseWriter, r *http.Request) {
	perfMetrics, err := metrics.GetPerformanceMetrics()
	if err != nil {
		log.Error("failed to retrieve performance metrics", "error", err)
		return
	}

	json.NewEncoder(w).Encode(perfMetrics)
}

// Serve the broker metrics endpoint
func Serve() error {
	port := env.GetVar("BROKER_METRICS_PORT")
	addr := ":" + port

	log.Info("start serving metrics endpoint")

	router := http.NewServeMux()
	router.HandleFunc("/metrics/marketdata", marketDataMetricsHandler)
	router.HandleFunc("/metrics/performance", performanceMetricsHandler)

	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	return server.ListenAndServe()
}
