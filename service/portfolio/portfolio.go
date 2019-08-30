package portfolio

import (
	"fmt"
	"time"

	"github.com/alpacahq/gobroker/service/position"
	"github.com/alpacahq/gobroker/service/tradeaccount"
	"github.com/pkg/errors"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/price"
	"github.com/alpacahq/gobroker/service/assetcache"
	"github.com/alpacahq/gobroker/service/pfhistory"
	"github.com/alpacahq/gobroker/utils/tradingdate"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/polycache/rest/client"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/shopspring/decimal"
)

type SoDPosition struct {
	AssetID string
	Symbol  string
	Qty     decimal.Decimal
}

func (p SoDPosition) GetQty() decimal.Decimal {
	return p.Qty
}

func (p SoDPosition) GetSymbol() string {
	return p.Symbol
}

type POrder struct {
	AssetID  string
	Symbol   string
	Qty      decimal.Decimal
	Price    decimal.Decimal
	Side     enum.Side
	FilledAt *time.Time
}

func (p POrder) GetQty() decimal.Decimal {
	return p.Qty
}

func (p POrder) GetPrice() decimal.Decimal {
	return p.Price
}

func (p POrder) GetSymbol() string {
	return p.Symbol
}

func (p POrder) GetFilledAt() *time.Time {
	return p.FilledAt
}

func (p POrder) GetSide() enum.Side {
	return p.Side
}

type ChartData struct {
	Arrays     []interface{}       `json:"arrays"`
	Attributes []map[string]string `json:"attributes"`
	Timeframe  calendar.RangeFreq  `json:"timeframe"`
	BaseValue  float64             `json:"base_value"`
}

// NewChartData convert portfolio history response to basic chart data format.
func NewChartData(
	timestamps []time.Time,
	profitLosses []*decimal.Decimal,
	lastPFValue decimal.Decimal,
	timeframe calendar.RangeFreq) *ChartData {

	basis, _ := lastPFValue.Float64()

	arrays := make([]interface{}, 4)

	t := make([]int64, len(timestamps))
	c := make([]*float64, len(timestamps))
	pl := make([]*float64, len(timestamps))
	pf := make([]*float64, len(timestamps))

	for i := range timestamps {
		t[i] = timestamps[i].Unix() * 1000

		if profitLosses[i] != nil {
			fpl, _ := profitLosses[i].Float64()
			pl[i] = &fpl
			if basis > 0 {
				pct := fpl / basis
				c[i] = &pct
			} else {
				pct := float64(0.0)
				c[i] = &pct
			}

			pfVal, _ := profitLosses[i].Add(lastPFValue).Float64()
			pf[i] = &pfVal
		}
	}

	arrays[0] = t
	arrays[1] = pl
	arrays[2] = c
	arrays[3] = pf

	attrs := make([]map[string]string, 4)
	attrs[0] = map[string]string{"name": "Date", "type": "date"}
	attrs[1] = map[string]string{"name": "ProfitLoss", "type": "float64"}
	attrs[2] = map[string]string{"name": "ProfitLossPctChange", "type": "float64"}
	attrs[3] = map[string]string{"name": "PortfolioValue", "type": "float64"}
	data := ChartData{Arrays: arrays, Attributes: attrs, Timeframe: timeframe, BaseValue: basis}

	return &data
}

func NewDailyChartData(
	timestamps []time.Time,
	cumProdPctChange,
	profitLosses,
	portfolioValues []*decimal.Decimal,
	lastPFValue decimal.Decimal,
	timeframe calendar.RangeFreq) *ChartData {

	floatLastPFValue, _ := lastPFValue.Float64()
	arrays := make([]interface{}, 4)

	t := make([]int64, len(timestamps))
	c := make([]*float64, len(timestamps))
	pl := make([]*float64, len(timestamps))
	pf := make([]*float64, len(timestamps))

	for i := range timestamps {
		t[i] = timestamps[i].Unix() * 1000

		if cumProdPctChange[i] != nil {
			x, _ := cumProdPctChange[i].Float64()
			c[i] = &x
		} else {
			x := float64(0.0)
			c[i] = &x
		}

		if profitLosses[i] != nil {
			x, _ := profitLosses[i].Float64()
			pl[i] = &x
		} else {
			x := float64(0.0)
			pl[i] = &x
		}

		if portfolioValues[i] != nil {
			x, _ := portfolioValues[i].Float64()
			pf[i] = &x
		} else {
			x := float64(0.0)
			pf[i] = &x
		}
	}

	arrays[0] = t
	arrays[1] = pl
	arrays[2] = c
	arrays[3] = pf

	attrs := make([]map[string]string, 4)
	attrs[0] = map[string]string{"name": "Date", "type": "date"}
	attrs[1] = map[string]string{"name": "ProfitLoss", "type": "float64"}
	attrs[2] = map[string]string{"name": "ProfitLossPctChange", "type": "float64"}
	attrs[3] = map[string]string{"name": "PortfolioValue", "type": "float64"}
	data := ChartData{Arrays: arrays, Attributes: attrs, Timeframe: timeframe, BaseValue: floatLastPFValue}

	return &data
}

type PortfolioService interface {
	GetHistory(
		id uuid.UUID,
		on tradingdate.TradingDate,
		timeframe calendar.RangeFreq,
		period string, now *time.Time) (*ChartData, error)
	WithTx(tx *gorm.DB) PortfolioService
}

type portfolioService struct {
	tx         *gorm.DB
	assetcache assetcache.AssetCache
	accService tradeaccount.TradeAccountService
	posService position.PositionService
}

func Service(assetcache assetcache.AssetCache, accService tradeaccount.TradeAccountService, posService position.PositionService) PortfolioService {
	return &portfolioService{
		assetcache: assetcache,
		accService: accService,
		posService: posService,
	}
}

func (s *portfolioService) WithTx(tx *gorm.DB) PortfolioService {
	s.tx = tx
	return s
}

func (s *portfolioService) GetHistory(
	id uuid.UUID,
	on tradingdate.TradingDate,
	timeframe calendar.RangeFreq,
	period string, now *time.Time) (*ChartData, error) {

	if timeframe == calendar.Min5 {
		return s.getIntradayPortfolioHistories(id, on, now)
	}

	return s.getDailyPortfolioHistories(id, on, period, now)
}

func (s *portfolioService) getDailyPortfolioHistories(
	id uuid.UUID,
	on tradingdate.TradingDate,
	period string, now *time.Time) (*ChartData, error) {

	tx := s.tx

	var since, until tradingdate.TradingDate
	if period != "all" {
		pmap := map[string]int{
			"1M": 20,
			"3M": 60,
			"6M": 20 * 6,
			"1A": 252,
		}
		days, ok := pmap[period]
		if !ok {
			return nil, fmt.Errorf("selected period does not exist")
		}
		since = on.DaysAgo(days)
	} else {
		acc, err := s.accService.WithTx(tx).GetByID(id)
		if err != nil {
			return nil, err
		}
		accCreatedAt := acc.CreatedAt.In(calendar.NY)
		if calendar.IsMarketDay(accCreatedAt) {
			// Account which created on the trading day is expected to have
			// balance snapshot on that day.
			if td, err := tradingdate.New(accCreatedAt); err == nil {
				since = *td
			}
		} else {
			// Otherwise the account should have the next trading day's balance snapshot
			since = tradingdate.Last(accCreatedAt).Next()
		}
	}
	// historical portfolio chart should be until current trading day - 1 trading day,
	// because last trading day PL is not determined by apex side.
	if now != nil && !calendar.IsMarketDay(*now) {
		d := now.In(calendar.NY)
		if (d.Hour() >= 9 && d.Minute() >= 30) || (d.Hour() >= 10) {
			until = on
		} else {
			until = on.Prev()
		}
	} else {
		until = on.Prev()
	}

	snaps := []models.DayPLSnapshot{}

	err := tx.
		Where("account_id = ?", id).
		Where("date >= ?", since.String()).
		Where("date <= ?", until.String()).
		Order("date ASC").
		Find(&snaps).Error

	if err != nil {
		return nil, err
	}

	if len(snaps) == 0 {
		return NewDailyChartData(
			[]time.Time{},
			[]*decimal.Decimal{},
			[]*decimal.Decimal{},
			[]*decimal.Decimal{},
			decimal.Zero,
			calendar.D1), nil
	}

	timestamps, err := calendar.NewRange(
		since.MarketOpen(), until.MarketClose(), calendar.D1)
	if err != nil {
		return nil, err
	}

	pls := make([]*decimal.Decimal, len(timestamps))
	portfolioValues := make([]*decimal.Decimal, len(timestamps))
	cumProdPctChanges := make([]*decimal.Decimal, len(timestamps))
	one := decimal.New(1, 0)

	for i, t := range timestamps {
		ok := false

		for _, snap := range snaps {

			if snap.DateString() == t.Format("2006-01-02") {
				pls[i] = &snap.ProfitLoss
				pfVal := snap.Basis.Add(snap.ProfitLoss)
				portfolioValues[i] = &pfVal
				var prev decimal.Decimal
				if i == 0 {
					prev = one
				} else {
					prev = cumProdPctChanges[i-1].Add(one)
				}

				var cumProdPctChange decimal.Decimal
				if snap.Basis.GreaterThan(decimal.Zero) {
					pctChange := snap.ProfitLoss.Div(snap.Basis)
					cumProdPctChange = pctChange.Add(one).Mul(prev).Sub(one)
				} else {
					cumProdPctChange = prev.Sub(one)
				}

				cumProdPctChanges[i] = &cumProdPctChange

				ok = true
				break
			}

			if !ok {
				cumProdPctChanges[i] = &decimal.Zero
				pls[i] = &decimal.Zero
			}
		}
	}

	// Represents zero for culmative percent change.
	baseValue := decimal.Zero
	if len(snaps) > 0 {
		baseValue = snaps[0].Basis
	}

	chartdata := NewDailyChartData(
		timestamps,
		cumProdPctChanges,
		pls, portfolioValues,
		baseValue, calendar.D1)

	return chartdata, nil
}

// GetTotalEquity calculate realtime total equity value.
// Need to pass in date = current trading date to be in sync with other service calls.
func (s *portfolioService) GetTotalEquity(accID uuid.UUID, date tradingdate.TradingDate) (decimal.Decimal, error) {
	balances, err := s.accService.WithTx(s.tx).GetBalancesByID(accID, date.MarketOpen())
	if err != nil {
		return decimal.Zero, errors.Wrap(err, "failed to get balances")
	}

	positions, err := s.posService.WithTx(s.tx).List(accID)
	if err != nil {
		return decimal.Zero, errors.Wrap(err, "failed to get positions")
	}

	totalEquity := balances.Cash
	for _, p := range positions {
		totalEquity = totalEquity.Add(p.MarketValue)
	}

	return totalEquity, nil
}

func (s *portfolioService) getIntradayPortfolioHistories(
	accountID uuid.UUID,
	on tradingdate.TradingDate,
	now *time.Time) (*ChartData, error) {

	portfolioValue, err := s.GetTotalEquity(accountID, on)
	if err != nil {
		return nil, err
	}

	tx := s.tx

	since := on.MarketOpen()
	until := on.MarketClose()

	sodPositions := []SoDPosition{}
	orders := []POrder{}

	err = tx.Raw(`
WITH S AS (
  SELECT p.asset_id,
         p.qty,
         p.side
  FROM positions p
  WHERE
    p.account_id = ?
    AND p.entry_timestamp < ?
    AND ( p.exit_timestamp IS NULL OR p.exit_timestamp >= ? )
    AND p.status != 'split'
)
SELECT SUM(s.qty) qty, s.asset_id asset_id
FROM S s
GROUP BY s.side, s.asset_id
  `, accountID, since, since).Scan(&sodPositions).Error
	if err != nil {
		return nil, err
	}

	tfpositions := make([]pfhistory.Position, len(sodPositions))
	for i, p := range sodPositions {
		if asset := s.assetcache.GetByID(uuid.FromStringOrNil(p.AssetID)); asset != nil {
			p.Symbol = asset.Symbol
			tfpositions[i] = p
		} else {
			return nil, fmt.Errorf("failed to load asset %v", p.AssetID)
		}
	}

	// Right now, only take care for longs
	err = tx.Raw(`
SELECT
	*
FROM
	(
		SELECT
		  asset_id, qty, entry_timestamp as filled_at, 'buy' as side, entry_price as price
		FROM positions
		WHERE
		  account_id = ?
		  AND (entry_timestamp >= ?)
		  AND (entry_timestamp <= ?)
		  AND status != 'split'
		UNION ALL
		SELECT
		  asset_id, qty, exit_timestamp as filled_at, 'sell' as side, exit_price as price
		FROM positions
		WHERE
		  account_id = ?
		  AND (exit_timestamp >= ?)
		  AND (exit_timestamp <= ?)
	) t
ORDER BY filled_at asc
  `, accountID, since, until, accountID, since, until).Scan(&orders).Error
	if err != nil {
		return nil, err
	}

	tforders := make([]pfhistory.Order, len(orders))
	for i, v := range orders {
		if asset := s.assetcache.GetByID(uuid.FromStringOrNil(v.AssetID)); asset != nil {
			v.Symbol = asset.Symbol
			tforders[i] = v
		} else {
			return nil, fmt.Errorf("failed to load asset %v", v.AssetID)
		}
	}

	// Get all the related symbols
	smap := map[string]bool{}
	for _, p := range tfpositions {
		smap[p.GetSymbol()] = true
	}
	for _, o := range tforders {
		smap[o.GetSymbol()] = true
	}
	symbols := []string{}
	for k := range smap {
		symbols = append(symbols, k)
	}

	// Use prev trading day's market close price as beginning price.
	beginningPrices, err := price.LastDayClosing(symbols, &on)
	if err != nil {
		return nil, err
	}

	csm, err := price.Get(symbols, "5Min", since, until)
	if err != nil {
		return nil, err
	}

	// Last Price on the ComputePL with be overritten by live quotes.
	livePrices, err := client.GetTrades(symbols)
	if err != nil {
		return nil, err
	}

	hist := pfhistory.ComputePLWithQuoteOverride(
		tfpositions, tforders, csm,
		beginningPrices, livePrices,
		since, until, *now)

	// subtract last close profit/loss from current portfolio value
	// to get total equity at end of last trading day, and pass
	// to chart data function.
	dayChange := decimal.Zero
	for i := range hist.Close {
		if hist.Close[i] == nil {
			continue
		}
		dayChange = *hist.Close[i]
	}

	portfolioValue = portfolioValue.Sub(dayChange)

	return NewChartData(hist.Time, hist.Close, portfolioValue, calendar.Min5), nil
}
