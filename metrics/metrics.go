package metrics

import (
	"fmt"
	"runtime"
	"time"

	"github.com/alpacahq/gobroker/gbreg"
	"github.com/alpacahq/gobroker/service/assetcache"
	"github.com/alpacahq/gobroker/service/bar"
	"github.com/alpacahq/gobroker/service/quote"
	"github.com/alpacahq/gopaca/db"
	"github.com/gofrs/uuid"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

var (
	// arbitrary symbol list to grab bars/quotes for. a mix
	// of small, medium, large cap & stock + etf
	testSymbols = []string{"AAPL", "SPY", "AMD", "RUN", "EWZ"}
	limit       = 1
)

// MarketData is a generic wrapper struct for
// bar and quote market data
type MarketData struct {
	Timestamp time.Time   `json:"timestamp"`
	Symbol    string      `json:"symbol"`
	Data      interface{} `json:"data"`
}

// MarketDataMetrics includes the required data
// to analyze the health of the market data
// related systems.
type MarketDataMetrics struct {
	// stale data metrics
	OldestQuote MetricQuote `json:"oldest_quote"`
	OldestBar   MetricBar   `json:"oldest_bar"`
	// latency metrics
	QuoteLatency time.Duration `json:"quote_latency"`
	BarLatency   time.Duration `json:"bar_latency"`
	// error metrics
	QuoteError error `json:"quote_error"`
	BarError   error `json:"bar_error"`
}

// MetricQuote contains the metric info needed for quotes
type MetricQuote struct {
	Symbol    string        `json:"symbol"`
	Field     string        `json:"field"`
	Timestamp time.Time     `json:"timestamp"`
	Age       time.Duration `json:"age"` // milliseconds
}

func findOldestQuote(quotes []*quote.QuoteAsAsset) MetricQuote {
	var (
		field     string
		symbol    string
		timestamp = time.Now()
	)

	for _, q := range quotes {
		if q.AskTimestamp.Before(timestamp) {
			timestamp = q.AskTimestamp
			field = "ask"
			symbol = q.Symbol
		}

		if q.BidTimestamp.Before(timestamp) {
			timestamp = q.BidTimestamp
			field = "bid"
			symbol = q.Symbol
		}

		if q.LastTimestamp.Before(timestamp) {
			timestamp = q.LastTimestamp
			field = "last"
			symbol = q.Symbol
		}
	}

	return MetricQuote{
		Symbol:    symbol,
		Field:     field,
		Timestamp: timestamp,
		Age:       time.Now().Sub(timestamp),
	}
}

// MetricBar contains the metric info needed for bars
type MetricBar struct {
	Symbol    string        `json:"symbol"`
	Timestamp time.Time     `json:"timestamp"`
	Age       time.Duration `json:"age"` // milliseconds
}

func findOldestBar(barList []*bar.AssetBars) MetricBar {
	var symbol string
	timestamp := time.Now()

	for _, b := range barList {
		bar := b.Bars[0]
		if bar.Time.Before(timestamp) {
			timestamp = bar.Time
			symbol = b.Symbol
		}
	}

	return MetricBar{
		Symbol:    symbol,
		Timestamp: timestamp,
		Age:       time.Now().Sub(timestamp),
	}
}

// GetMarketDataMetrics returns market data related
// metrics for alerts and analysis.
func GetMarketDataMetrics() (*MarketDataMetrics, error) {
	assetIDs := []uuid.UUID{}
	for _, symbol := range testSymbols {
		asset := assetcache.Get(symbol)
		if asset == nil {
			return nil, fmt.Errorf("assetcache does not contain %v", symbol)
		}
		assetIDs = append(assetIDs, asset.IDAsUUID())
	}

	// quotes
	qSrv := gbreg.Services.Quote().WithTx(db.DB())

	start := time.Now()
	quotes, qErr := qSrv.GetByIDs(assetIDs)

	qLatency := time.Now().Sub(start)

	// bars
	bSrv := gbreg.Services.Bar().WithTx(db.DB())

	start = time.Now()
	bars, bErr := bSrv.GetByIDs(assetIDs, "1Min", nil, nil, &limit)

	bLatency := time.Now().Sub(start)

	return &MarketDataMetrics{
		OldestQuote:  findOldestQuote(quotes),
		OldestBar:    findOldestBar(bars),
		QuoteLatency: qLatency,
		BarLatency:   bLatency,
		QuoteError:   qErr,
		BarError:     bErr,
	}, nil
}

// PerformanceMetrics includes all data relevant to
// the performance of gobroker.
type PerformanceMetrics struct {
	DatabaseLatency    time.Duration `json:"db_latency"`
	MemoryUsageTotal   uint64        `json:"mem_usage_total"`
	MemoryUsagePercent float64       `json:"mem_usage_pct"`
	GoRoutines         int64         `json:"goroutines"`
	CPUUsagePercent    float64       `json:"cpu_usage_pct"`
}

// GetPerformanceMetrics returns performance related
// metrics for alerts and analysis.
func GetPerformanceMetrics() (*PerformanceMetrics, error) {
	// memory stats
	v, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	// cpu stats
	pct, err := cpu.Percent(time.Second, false)
	if err != nil {
		return nil, err
	}

	if len(pct) == 0 {
		return nil, fmt.Errorf("failed to retrieve cpu usage stats")
	}

	// database latency
	start := time.Now()
	if err := db.DB().DB().Ping(); err != nil {
		return nil, err
	}

	dbLatency := time.Now().Sub(start)

	return &PerformanceMetrics{
		MemoryUsageTotal:   v.Used,
		MemoryUsagePercent: v.UsedPercent,
		CPUUsagePercent:    pct[0],
		DatabaseLatency:    dbLatency,
		GoRoutines:         int64(runtime.NumGoroutine()),
	}, nil
}
