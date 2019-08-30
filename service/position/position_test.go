package position

import (
	"math/big"
	"testing"

	"github.com/alpacahq/gobroker/dbtest"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/service/assetcache"
	"github.com/alpacahq/gobroker/utils/tradingdate"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/polycache/structures"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type PositionTestSuite struct {
	dbtest.Suite
	date      tradingdate.TradingDate
	asset     *models.Asset
	position  *models.Position
	accountID uuid.UUID
}

func TestPositionTestSuite(t *testing.T) {
	suite.Run(t, new(PositionTestSuite))
}

func (s *PositionTestSuite) SetupSuite() {
	s.SetupDB()
	s.date = tradingdate.Current()
	s.asset = &models.Asset{
		Class:    enum.AssetClassUSEquity,
		Exchange: "NASDAQ",
		Symbol:   "AAPL",
		Status:   enum.AssetActive,
		Tradable: true,
	}
	if err := db.DB().Create(s.asset).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	s.accountID = uuid.Must(uuid.NewV4())
	s.position = &models.Position{
		AssetID:        s.asset.IDAsUUID(),
		AccountID:      s.accountID.String(),
		Status:         models.Open,
		Side:           models.Long,
		Qty:            decimal.NewFromBigInt(big.NewInt(int64(100)), 0),
		EntryPrice:     decimal.NewFromFloat(100.00),
		EntryTimestamp: clock.Now(),
		EntryOrderID:   uuid.Must(uuid.NewV4()).String(),
	}
	if err := db.DB().Create(s.position).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}

	priceProxy.LivePrices = func(symbols []string) (map[string]structures.Trade, error) {
		ret := map[string]structures.Trade{}
		for _, symbol := range symbols {
			if symbol == "AAPL" {
				ret[symbol] = structures.Trade{
					Price:     110.00,
					Size:      10,
					Timestamp: clock.Now(),
				}
			}
		}
		return ret, nil
	}

	priceProxy.LastDayClosing = func(symbols []string, on *tradingdate.TradingDate) (map[string]structures.Trade, error) {
		ret := map[string]structures.Trade{}
		for _, symbol := range symbols {
			if symbol == "AAPL" {
				ret[symbol] = structures.Trade{
					Price:     109.00,
					Size:      10,
					Timestamp: clock.Now(),
				}
			}
		}
		return ret, nil
	}
}

func (s *PositionTestSuite) TearDownSuite() {
	s.TeardownDB()
}

func (s *PositionTestSuite) TestGetByAssetID() {
	srv := Service(assetcache.GetAssetCache()).WithTx(db.DB())

	pos, err := srv.GetByAssetID(s.accountID, s.asset.IDAsUUID())
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), pos)
	assert.Equal(s.T(), pos.Symbol, s.asset.Symbol)

	pos, err = srv.GetByAssetID(s.accountID, uuid.Must(uuid.NewV4()))
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), pos)
}

func (s *PositionTestSuite) TestList() {
	srv := Service(assetcache.GetAssetCache()).WithTx(db.DB())

	pos, err := srv.List(s.accountID)
	assert.NotNil(s.T(), pos)
	assert.Len(s.T(), pos, 1)
	assert.Equal(s.T(), pos[0].Symbol, s.asset.Symbol)
	assert.Equal(s.T(), decimal.NewFromFloat(11000).String(), pos[0].MarketValue.String())
	assert.Equal(s.T(), "0.1", pos[0].UnrealizedPLPC.String())
	assert.Equal(s.T(), "100", pos[0].AvgEntryPrice.String())
	assert.Equal(s.T(), "100", pos[0].Qty.String())
	assert.Nil(s.T(), err)

	pos, err = srv.List(uuid.Must(uuid.NewV4()))
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), pos)
	assert.Len(s.T(), pos, 0)
}
