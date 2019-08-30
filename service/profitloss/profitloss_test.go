package profitloss

import (
	"testing"
	"time"

	"github.com/alpacahq/gobroker/dbtest"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gobroker/service/assetcache"
	"github.com/alpacahq/gobroker/utils/tradingdate"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/marketstore/utils/io"
	"github.com/alpacahq/polycache/structures"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ProfitlossTestSuite struct {
	dbtest.Suite
	asset *models.Asset
	acct  *models.Account
}

func TestProfitlossTestSuite(t *testing.T) {
	suite.Run(t, new(ProfitlossTestSuite))
}

func (s *ProfitlossTestSuite) SetupSuite() {
	s.SetupDB()
	s.asset = &models.Asset{
		Class:    enum.AssetClassUSEquity,
		Exchange: "NASDAQ",
		Symbol:   "AAPL",
		Status:   enum.AssetActive,
		Tradable: true,
	}

	tx := db.Begin()

	if err := tx.Create(s.asset).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}

	srv := account.Service().WithTx(tx)
	s.acct, _ = srv.Create(
		"profitloss@example.com",
		uuid.Must(uuid.NewV4()),
	)
	assert.NotNil(s.T(), s.acct)

	tx.Commit()

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

	priceProxy.LivePrices = func(symbols []string) (map[string]structures.Trade, error) {
		return map[string]structures.Trade{}, nil
	}

}

func (s *ProfitlossTestSuite) TearDownSuite() {
	s.TeardownDB()
}

func (s *ProfitlossTestSuite) TestGet() {

	sod := time.Date(2017, 05, 02, 9, 30, 0, 0, calendar.NY)

	snap := &models.DayPLSnapshot{
		AccountID:  s.acct.ID,
		ProfitLoss: decimal.Zero,
		Basis:      decimal.New(int64(1000), 0),
		Date:       "2017-05-01",
	}

	if err := db.DB().Create(snap).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}

	exitID := uuid.Must(uuid.NewV4()).String()
	exitPx := decimal.New(int64(110), 0)
	exitTime := sod.Add(time.Minute * 5)
	position := &models.Position{
		AssetID:        s.asset.IDAsUUID(),
		AccountID:      s.acct.ID,
		Status:         models.Closed,
		Side:           models.Long,
		Qty:            decimal.New(3, 0),
		EntryPrice:     decimal.New(100, 0),
		EntryTimestamp: sod.Add(time.Minute),
		EntryOrderID:   uuid.Must(uuid.NewV4()).String(),
		ExitPrice:      &exitPx,
		ExitTimestamp:  &exitTime,
		ExitOrderID:    &exitID,
	}
	if err := db.DB().Create(position).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}

	timestamps, _ := calendar.NewRange(sod, sod.Add(15*time.Minute), calendar.Min5)
	assert.Equal(s.T(), len(timestamps), 3)

	cs1 := NewFakeCS(timestamps, []float32{11, 12, 10})
	csm := io.ColumnSeriesMap{}
	aaplTbk := io.NewTimeBucketKey("AAPL/5Min/OHLCV", io.DefaultTimeBucketSchema)
	csm.AddColumnSeries(*aaplTbk, cs1)

	priceProxy.Get = func(symbols []string, timeframe string, since, until time.Time) (io.ColumnSeriesMap, error) {
		return csm, nil
	}

	svc := Service(assetcache.GetAssetCache()).WithTx(db.DB())

	pl, err := svc.Get(s.acct.IDAsUUID(), tradingdate.Last(sod), sod.Add(time.Minute*15))
	if err != nil {
		assert.FailNow(s.T(), err.Error())
	}

	{
		assert.Equal(s.T(), "0.03", pl.DayPLPC.String())
	}
}

func NewFakeCS(epochs calendar.DateRange, values []float32) *io.ColumnSeries {
	cs := io.NewColumnSeries()
	cs.AddColumn("Epoch", epochs.Unix())
	cs.AddColumn("Close", values)
	return cs
}
