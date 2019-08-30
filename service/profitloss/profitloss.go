package profitloss

import (
	"fmt"
	"time"

	"github.com/alpacahq/polycache/rest/client"
	"github.com/alpacahq/polycache/structures"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/price"
	"github.com/alpacahq/gobroker/service/assetcache"
	"github.com/alpacahq/gobroker/service/pfhistory"
	"github.com/alpacahq/gobroker/utils/tradingdate"
	"github.com/alpacahq/marketstore/utils/io"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

// for test mock
var priceProxy = struct {
	Get            func(symbols []string, timeframe string, since, until time.Time) (io.ColumnSeriesMap, error)
	LastDayClosing func(symbols []string, on *tradingdate.TradingDate) (map[string]structures.Trade, error)
	LivePrices     func(symbols []string) (map[string]structures.Trade, error)
}{
	Get:            price.Get,
	LastDayClosing: price.LastDayClosing,
	LivePrices:     client.GetTrades,
}

type ProfitLoss struct {
	TotalPLPC decimal.Decimal `json:"total_plpc"`
	DayPLPC   decimal.Decimal `json:"day_plpc"`
}

type ProfitLossService interface {
	Get(accountID uuid.UUID, on tradingdate.TradingDate, now time.Time) (*ProfitLoss, error)
	WithTx(tx *gorm.DB) ProfitLossService
}

type profitLossService struct {
	tx         *gorm.DB
	assetcache assetcache.AssetCache
}

func Service(assetcache assetcache.AssetCache) ProfitLossService {
	return &profitLossService{assetcache: assetcache}
}

var one = decimal.New(1, 0)

func (s *profitLossService) WithTx(tx *gorm.DB) ProfitLossService {
	s.tx = tx
	return s
}

// Get returns realtime profit loss.
func (s *profitLossService) Get(accountID uuid.UUID, on tradingdate.TradingDate, now time.Time) (*ProfitLoss, error) {
	snaps := []models.DayPLSnapshot{}

	q := s.tx.
		Where("account_id = ?", accountID).
		Where("date < ?", on.String()).
		Order("date ASC").
		Find(&snaps)

	if q.Error != nil {
		return nil, errors.Wrap(q.Error, "failed to query snapshots")
	}

	totalPLPC := getTimeWeightedTotalPLPct(snaps)

	var lastPFValue decimal.Decimal
	if len(snaps) == 0 {
		lastPFValue = decimal.Zero
	} else {
		lastPFValue = snaps[len(snaps)-1].Basis.Add(snaps[len(snaps)-1].ProfitLoss)
	}

	hist, err := s.getIntradayPLHist(accountID, on, now)

	if err != nil {
		return nil, errors.Wrap(err, "failed to calc intraday PL history")
	}

	intraPL := decimal.Zero
	for _, v := range hist.Close {
		if v == nil {
			break
		}
		intraPL = *v
	}

	intraPLPC := decimal.Zero
	if lastPFValue.GreaterThan(decimal.Zero) {
		intraPLPC = intraPL.Div(lastPFValue)
		totalPLPC = totalPLPC.Add(one).Mul(intraPLPC.Add(one)).Sub(one)
	}

	profitloss := ProfitLoss{
		TotalPLPC: totalPLPC,
		DayPLPC:   intraPLPC,
	}

	return &profitloss, nil
}

func getTimeWeightedTotalPLPct(snaps []models.DayPLSnapshot) decimal.Decimal {
	cumprod := one
	for i := range snaps {
		if snaps[i].Basis.GreaterThan(decimal.Zero) {
			coef := snaps[i].ProfitLoss.Div(snaps[i].Basis).Add(one)
			cumprod = cumprod.Mul(coef)
		}
	}
	cumprod = cumprod.Sub(one)
	return cumprod
}

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

func (s *profitLossService) getIntradayPLHist(id uuid.UUID, on tradingdate.TradingDate, now time.Time) (*pfhistory.PFHistoryResponse, error) {
	tx := s.tx

	since := on.MarketOpen()
	until := on.Next().MarketOpen()

	var err error
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
  `, id, since, since).Scan(&sodPositions).Error
	if err != nil {
		return nil, err
	}

	tfpositions := make([]pfhistory.Position, len(sodPositions))
	for i, p := range sodPositions {
		if asset := s.assetcache.Get(p.AssetID); asset != nil {
			sodPositions[i].Symbol = asset.Symbol
			tfpositions[i] = sodPositions[i]
		} else {
			return nil, fmt.Errorf("asset not found for %v", p.AssetID)
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
		  AND (entry_timestamp < ?)
		  AND status != 'split'
		UNION ALL
		SELECT
		  asset_id, qty, exit_timestamp as filled_at, 'sell' as side, exit_price as price
		FROM positions
		WHERE
		  account_id = ?
		  AND (exit_timestamp >= ?)
		  AND (exit_timestamp < ?)
	) t
ORDER BY filled_at asc
  `, id, since, until, id, since, until).Scan(&orders).Error
	if err != nil {
		return nil, err
	}

	tforders := make([]pfhistory.Order, len(orders))
	for i, v := range orders {
		if asset := s.assetcache.Get(v.AssetID); asset != nil {
			orders[i].Symbol = asset.Symbol
			tforders[i] = orders[i]
		} else {
			return nil, fmt.Errorf("asset not found for %v", v.AssetID)
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
	beginningPrices, err := priceProxy.LastDayClosing(symbols, &on)
	if err != nil {
		return nil, err
	}

	csm, err := priceProxy.Get(symbols, "5Min", since, until)
	if err != nil {
		return nil, err
	}

	// Last Price on the ComputePL with be overritten by live quotes.
	livePrices, err := priceProxy.LivePrices(symbols)
	if err != nil {
		return nil, err
	}

	hist := pfhistory.ComputePLWithQuoteOverride(
		tfpositions, tforders, csm, beginningPrices, livePrices,
		since, until, now)

	return &hist, nil
}
