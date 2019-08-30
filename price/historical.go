package price

import (
	"fmt"
	"strconv"
	"time"

	"github.com/alpacahq/gobroker/external/mkts"
	"github.com/alpacahq/gobroker/utils/tradingdate"
	"github.com/alpacahq/polycache/structures"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"

	"github.com/alpacahq/marketstore/frontend"
	io "github.com/alpacahq/marketstore/utils/io"
)

// LastDayClosing retrieves the last closing price for the given trading date
func LastDayClosing(symbols []string, on *tradingdate.TradingDate) (map[string]structures.Trade, error) {
	if len(symbols) == 0 {
		return nil, nil
	}

	args := &frontend.MultiQueryRequest{}
	var prev tradingdate.TradingDate

	if on == nil {
		prev = tradingdate.Current().Prev()
	} else {
		prev = on.Prev()
	}

	for _, symbol := range symbols {
		args.Requests = append(args.Requests,
			frontend.NewQueryRequestBuilder(fmt.Sprintf("%v/1Min/OHLCV", symbol)).
				EpochEnd(prev.MarketClose().Unix()-1).
				LimitRecordCount(1).
				End())
	}

	csm, err := queryBars(args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query mktsdb")
	}

	trades := make(map[string]structures.Trade, len(symbols))

	for tbk, cs := range csm {
		trade := columnSeriesToTrade(cs)
		if trade == nil {
			continue
		}
		trades[tbk.GetItems()[0]] = *trade
	}
	return trades, nil
}

func Get(symbols []string, timeframe string, since time.Time, until time.Time) (io.ColumnSeriesMap, error) {
	if len(symbols) == 0 {
		return nil, nil
	}

	args := &frontend.MultiQueryRequest{}
	for _, symbol := range symbols {
		args.Requests = append(args.Requests,
			frontend.NewQueryRequestBuilder(fmt.Sprintf("%v/%v/OHLCV", symbol, timeframe)).
				EpochStart(since.Unix()).
				EpochEnd(until.Unix()).
				End())
	}

	return queryBars(args)
}

func GetLatest(symbols []string, until *time.Time) (map[string]structures.Trade, error) {
	if len(symbols) == 0 {
		return nil, nil
	}

	args := &frontend.MultiQueryRequest{}
	for _, symbol := range symbols {
		args.Requests = append(args.Requests,
			frontend.NewQueryRequestBuilder(fmt.Sprintf("%v/1Min/OHLCV", symbol)).
				EpochEnd(until.Unix()).
				LimitRecordCount(1).
				End())
	}

	csm, err := queryBars(args)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query mktsdb")
	}

	trades := make(map[string]structures.Trade, len(symbols))

	for tbk, cs := range csm {
		trade := columnSeriesToTrade(cs)
		if trade == nil {
			continue
		}
		trades[tbk.GetItems()[0]] = *trade
	}
	return trades, nil
}

func columnSeriesToTrade(cs *io.ColumnSeries) *structures.Trade {
	if cs == nil {
		return nil
	}
	c := cs.GetByName("Close")
	if c == nil {
		return nil
	}
	e := cs.GetByName("Epoch")
	if e == nil {
		return nil
	}
	closes := c.([]float32)
	epochs := e.([]int64)
	if len(closes) > 0 && len(epochs) > 0 {
		t := time.Unix(epochs[0], 0)
		return &structures.Trade{
			Timestamp: t,
			Price:     float64(closes[0]),
		}
	}
	return nil
}

type Offer struct {
	Price     decimal.Decimal `json:"price"`
	Change    decimal.Decimal `json:"change"`
	Timestamp time.Time       `json:"timestamp"`
	Symbol    string          `json:"symbol"`
}

func decimalFromFloatString(f float64) decimal.Decimal {
	s := strconv.FormatFloat(f, 'f', -1, 32)
	d, _ := decimal.NewFromString(s)
	return d
}

func queryBars(args *frontend.MultiQueryRequest) (io.ColumnSeriesMap, error) {
	resp, err := mkts.Client().DoRPC("Query", args)
	if err != nil {
		return nil, err
	}

	if resp == nil {
		return io.NewColumnSeriesMap(), nil
	}

	return *resp.(*io.ColumnSeriesMap), nil
}
