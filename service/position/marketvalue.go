package position

import (
	"fmt"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/utils/tradingdate"
	"github.com/alpacahq/gopaca/price"
	"github.com/alpacahq/polycache/structures"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
)

// MarketValueAt return Market Value of the positions at EOD on that trading day.
func (s *positionService) MarketValueAt(accountID uuid.UUID, at tradingdate.TradingDate) (decimal.Decimal, error) {
	var rawPositions []*models.Position
	value := decimal.Zero

	q := s.tx.
		Where("account_id = ? AND status != ?", accountID, models.Split).
		Where("(entry_timestamp < ?) AND (exit_timestamp >= ? OR exit_timestamp IS NULL)", at.Next().MarketOpen(), at.Next().MarketOpen()). // might better to use session start / end
		Find(&rawPositions)

	if len(rawPositions) == 0 {
		return value, nil
	}

	if q.Error != nil {
		return value, q.Error
	}

	symbols := make([]string, len(rawPositions))
	qtyByAsset := map[uuid.UUID]decimal.Decimal{}

	for i, pos := range rawPositions {
		asset := s.assetcache.GetByID(pos.AssetID)
		if asset == nil {
			return value, fmt.Errorf("asset \"%v\" not found", pos.AssetID)
		}
		symbols[i] = asset.Symbol

		// group by asset
		if qty, ok := qtyByAsset[pos.AssetID]; ok {
			qtyByAsset[pos.AssetID] = qty.Add(pos.Qty)
		} else {
			qtyByAsset[pos.AssetID] = pos.Qty
		}
	}

	cache, err := newClosePriceCache(symbols, at)
	if err != nil {
		return value, err
	}

	for assetID, qty := range qtyByAsset {
		asset := s.assetcache.GetByID(assetID)
		if asset == nil {
			return value, fmt.Errorf("asset \"%v\" not found", assetID)
		}

		price, err := cache.get(asset.Symbol)
		if err != nil {
			return value, err
		}

		value = value.Add(price.Mul(qty))
	}
	return value, nil
}

type closePriceCache struct {
	symbols []string
	prices  map[string]structures.Trade
}

func newClosePriceCache(symbols []string, at tradingdate.TradingDate) (*closePriceCache, error) {
	// I wanna closing price on that tradingdate.
	nextDay := at.Next()
	lastDayPrices, err := priceProxy.LastDayClosing(symbols, &nextDay)
	if err != nil {
		return nil, err
	}

	return &closePriceCache{
		symbols: symbols,
		prices:  lastDayPrices,
	}, nil
}

func (c *closePriceCache) get(symbol string) (lastDayPrice decimal.Decimal, err error) {
	trade, ok := c.prices[symbol]
	if ok {
		return price.FormatFloat64ForCalc(trade.Price), nil
	}
	return decimal.Zero, fmt.Errorf("price is not available for %v", symbol)
}
