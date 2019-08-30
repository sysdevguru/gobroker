package position

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/price"
	"github.com/alpacahq/gobroker/service/assetcache"
	"github.com/alpacahq/gobroker/utils/tradingdate"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/log"
	pricefmt "github.com/alpacahq/gopaca/price"
	"github.com/alpacahq/polycache/rest/client"
	"github.com/alpacahq/polycache/structures"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

// for test mock
var priceProxy = struct {
	LivePrices     func(symbols []string) (map[string]structures.Trade, error)
	LastDayClosing func(symbols []string, on *tradingdate.TradingDate) (map[string]structures.Trade, error)
}{
	LivePrices:     client.GetTrades,
	LastDayClosing: price.LastDayClosing,
}

type PositionService interface {
	GetByAssetID(accountID uuid.UUID, assetID uuid.UUID) (*ConsolidatedPosition, error)
	List(accountID uuid.UUID) ([]*ConsolidatedPosition, error)
	MarketValueAt(accountID uuid.UUID, at tradingdate.TradingDate) (decimal.Decimal, error)
	RawPositionsActAt(accountID uuid.UUID, at tradingdate.TradingDate) ([]*models.Position, error)
	WithTx(tx *gorm.DB) PositionService
}

type positionService struct {
	PositionService
	tx         *gorm.DB
	assetcache assetcache.AssetCache
}

func Service(assetcache assetcache.AssetCache) PositionService {
	return &positionService{assetcache: assetcache}
}

func (s *positionService) WithTx(tx *gorm.DB) PositionService {
	s.tx = tx
	return s
}

func (s *positionService) List(accountID uuid.UUID) ([]*ConsolidatedPosition, error) {
	var rawPositions []*models.Position

	q := s.tx.Where(
		"account_id = ? AND status = ?",
		accountID, models.Open).Find(&rawPositions)

	if q.Error != nil && q.Error != gorm.ErrRecordNotFound {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}

	if len(rawPositions) == 0 {
		return []*ConsolidatedPosition{}, nil
	}

	symbols := make([]string, len(rawPositions))
	posByAsset := map[uuid.UUID][]*models.Position{}

	for i, pos := range rawPositions {
		asset := s.assetcache.GetByID(pos.AssetID)
		if asset == nil {
			return nil, gberrors.InternalServerError.WithError(fmt.Errorf("asset \"%v\" not found", pos.AssetID))
		}
		symbols[i] = asset.Symbol

		// group by asset
		if plist, ok := posByAsset[pos.AssetID]; ok {
			posByAsset[pos.AssetID] = append(plist, pos)
		} else {
			posByAsset[pos.AssetID] = []*models.Position{pos}
		}
	}

	pcache, err := newPriceCache(symbols)
	if err != nil {
		return nil, gberrors.InternalServerError.WithError(errors.Wrap(err, "failed to build price cache"))
	}

	// aggregate by asset
	outplist := []*ConsolidatedPosition{}
	for assetID, plist := range posByAsset {
		// we checked nil above
		asset := s.assetcache.GetByID(assetID)
		consolidated, err := consolidate(asset, plist, pcache)
		if err != nil {
			return nil, gberrors.InternalServerError.WithError(errors.Wrap(err, "consolidation error"))
		}
		outplist = append(outplist, consolidated)
	}

	sort.Slice(outplist, func(i int, j int) bool {
		return strings.Compare(outplist[i].Symbol, outplist[j].Symbol) > 0
	})

	return outplist, nil
}

func (s *positionService) GetByAssetID(accountID uuid.UUID, assetID uuid.UUID) (*ConsolidatedPosition, error) {
	var rawPositions []*models.Position

	q := s.tx.Where(
		"account_id = ? AND status = ? AND asset_id = ?",
		accountID, models.Open, assetID).Find(&rawPositions)

	if len(rawPositions) == 0 {
		return nil, gberrors.NotFound.WithMsg("position does not exist")
	}

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}

	asset := s.assetcache.GetByID(assetID)
	if asset == nil {
		// Its internal server error, because its unexpected to have asset_id in db,
		// but not have it on asset cache.
		return nil, gberrors.InternalServerError.WithError(fmt.Errorf("asset \"%v\" not found", assetID))
	}
	pcache, err := newPriceCache([]string{asset.Symbol})
	if err != nil {
		return nil, gberrors.InternalServerError.WithError(errors.Wrap(err, "failed to build price cache"))
	}

	pos, err := consolidate(asset, rawPositions, pcache)
	if err != nil {
		return nil, gberrors.InternalServerError.WithError(errors.Wrap(err, "consolidation error"))
	}
	return pos, nil
}

type evaledPosition struct {
	models.Position
	MarketValue    decimal.Decimal
	CostBasis      decimal.Decimal
	UnrealizedPL   decimal.Decimal
	UnrealizedPLPC decimal.Decimal
	CurrentPrice   decimal.Decimal
	LastDayPrice   decimal.Decimal
	ChangeToday    decimal.Decimal
}

func evaluatePosition(
	position *models.Position,
	currentPrice, lastDayPrice decimal.Decimal) *evaledPosition {

	marketValue := position.Qty.Mul(currentPrice)
	costBasis := position.Qty.Mul(position.EntryPrice)
	coeff := decimal.New(1, 0)
	if position.Side != models.Long {
		coeff = decimal.New(-1, 0)
	}
	unrealizedProfitLoss := coeff.Mul(
		currentPrice.Sub(position.EntryPrice)).Mul(position.Qty)
	unrealizedProfitLossPct := coeff.Mul(
		currentPrice.Sub(position.EntryPrice)).Div(position.EntryPrice)

	epos := &evaledPosition{
		Position:       *position,
		MarketValue:    marketValue,
		CostBasis:      costBasis,
		UnrealizedPL:   unrealizedProfitLoss,
		UnrealizedPLPC: unrealizedProfitLossPct,
		CurrentPrice:   currentPrice,
		LastDayPrice:   lastDayPrice,
		ChangeToday:    currentPrice.Sub(lastDayPrice).Div(lastDayPrice),
	}
	return epos
}

type ConsolidatedPosition struct {
	AssetID                uuid.UUID           `json:"asset_id"`
	Symbol                 string              `json:"symbol"`
	Exchange               string              `json:"exchange"`
	AssetClass             enum.AssetClass     `json:"asset_class"`
	Qty                    decimal.Decimal     `json:"qty"`
	AvgEntryPrice          decimal.Decimal     `json:"avg_entry_price"`
	Side                   models.PositionSide `json:"side"`
	MarketValue            decimal.Decimal     `json:"market_value"`
	CostBasis              decimal.Decimal     `json:"cost_basis"`
	UnrealizedPL           decimal.Decimal     `json:"unrealized_pl"`
	UnrealizedPLPC         decimal.Decimal     `json:"unrealized_plpc"`
	UnrealizedIntradayPL   decimal.Decimal     `json:"unrealized_intraday_pl"`
	UnrealizedIntradayPLPC decimal.Decimal     `json:"unrealized_intraday_plpc"`
	CurrentPrice           decimal.Decimal     `json:"current_price"`
	LastDayPrice           decimal.Decimal     `json:"lastday_price"`
	ChangeToday            decimal.Decimal     `json:"change_today"`
	RawPositionIDs         []uint              `json:"-"`
}

func consolidate(
	asset *models.Asset,
	positions []*models.Position,
	pcache *priceCache) (*ConsolidatedPosition, error) {

	symbol := asset.Symbol
	currentPrice, lastDayPrice, err := pcache.get(symbol)
	if err != nil {
		// returns error in case something is missing.
		// argueably it is still useful to see intact positions,
		// but the total view is misleading anyway.
		log.Error("position service error", "action", "price retrieval", "error", err)
		return nil, gberrors.InternalServerError
	}
	totalMarketValue := decimal.Zero
	totalCostBasis := decimal.Zero
	totalQty := decimal.Zero
	pIDs := make([]uint, len(positions))
	var side *models.PositionSide
	coeff := decimal.New(1, 0)

	marketOpen := calendar.MarketOpen(clock.Now().In(calendar.NY))
	// For position BMO use lastprice as costbasis, else use cost_basis to calculate
	// unrealized intraday PL
	intradayCostBasis := decimal.Zero

	for i, pos := range positions {
		epos := evaluatePosition(pos, currentPrice, lastDayPrice)
		totalMarketValue = totalMarketValue.Add(epos.MarketValue)
		totalCostBasis = totalCostBasis.Add(epos.CostBasis)
		if side == nil {
			side = &pos.Side
		}
		if *side != pos.Side {
			// should not happen;
			// and US equity will probably not support it unlike FX
			return nil, fmt.Errorf("conflicting sides in account = %v, sybmol = %v", pos.AccountID, symbol)
		}

		if marketOpen != nil && pos.EntryTimestamp.Before(*marketOpen) {
			intradayCostBasis = intradayCostBasis.Add(lastDayPrice.Mul(pos.Qty))
		} else {
			intradayCostBasis = intradayCostBasis.Add(epos.CostBasis)
		}

		totalQty = totalQty.Add(pos.Qty)
		pIDs[i] = uint(pos.ID)
	}
	if *side != models.Long {
		// though short is not supported yet
		coeff = decimal.New(-1, 0)
	}
	unrealizedProfitLoss := coeff.Mul(
		totalMarketValue.Sub(totalCostBasis))
	unrealizedProfitLossPct := coeff.Mul(
		totalMarketValue.Sub(
			totalCostBasis).Div(totalCostBasis))

	unrealizedIntradayProfitLoss := coeff.Mul(
		totalMarketValue.Sub(intradayCostBasis))
	unrealizedIntradayProfitLossPct := coeff.Mul(
		totalMarketValue.Sub(
			intradayCostBasis).Div(intradayCostBasis))

	return &ConsolidatedPosition{
		AssetID:                asset.IDAsUUID(),
		Symbol:                 asset.Symbol,
		Exchange:               asset.Exchange,
		AssetClass:             asset.Class,
		Qty:                    totalQty,
		AvgEntryPrice:          totalCostBasis.Div(totalQty),
		Side:                   *side,
		MarketValue:            totalMarketValue,
		CostBasis:              totalCostBasis,
		UnrealizedPL:           unrealizedProfitLoss,
		UnrealizedPLPC:         unrealizedProfitLossPct,
		UnrealizedIntradayPL:   unrealizedIntradayProfitLoss,
		UnrealizedIntradayPLPC: unrealizedIntradayProfitLossPct,
		CurrentPrice:           currentPrice,
		LastDayPrice:           lastDayPrice,
		ChangeToday:            currentPrice.Sub(lastDayPrice).Div(lastDayPrice),
		RawPositionIDs:         pIDs,
	}, nil
}

// priceCache is to query and store the latest & lastDay prices
// for short period of time
type priceCache struct {
	sync.WaitGroup
	symbols       []string
	latestPrices  map[string]structures.Trade
	lastDayPrices map[string]structures.Trade
}

// query the quotes and bars in parallel
func newPriceCache(symbols []string) (*priceCache, error) {
	var (
		latestErr     error
		lastDayErr    error
		lastDayPrices map[string]structures.Trade
		latestPrices  map[string]structures.Trade
		pc            = &priceCache{symbols: symbols}
	)

	pc.Add(2)

	go func() {
		latestPrices, latestErr = priceProxy.LivePrices(symbols)
		pc.Done()
	}()

	go func() {
		lastDayPrices, lastDayErr = priceProxy.LastDayClosing(symbols, nil)
		pc.Done()
	}()

	pc.Wait()

	if latestErr != nil {
		return nil, latestErr
	}

	if lastDayErr != nil {
		return nil, lastDayErr
	}

	pc.lastDayPrices = lastDayPrices
	pc.latestPrices = latestPrices

	return pc, nil
}

func (c *priceCache) get(symbol string) (latestPrice, lastDayPrice decimal.Decimal, err error) {
	lastDayOffer, haveLastDay := c.lastDayPrices[symbol]
	lastTrade, haveLatest := c.latestPrices[symbol]

	lastDayPrice = pricefmt.FormatFloat64ForCalc(lastDayOffer.Price)
	latestPrice = pricefmt.FormatFloat64ForCalc(lastTrade.Price)

	if !haveLatest && haveLastDay {
		latestPrice = lastDayPrice
	}
	if !haveLastDay && haveLatest {
		// TODO: get open price
		lastDayPrice = latestPrice
	}
	if (!haveLastDay && !haveLatest) ||
		lastDayPrice.Equals(decimal.Zero) ||
		latestPrice.Equals(decimal.Zero) {
		return decimal.Zero, decimal.Zero, fmt.Errorf("price is not available for %v", symbol)
	}
	return latestPrice, lastDayPrice, nil
}
