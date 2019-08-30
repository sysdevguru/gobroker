package bar

import (
	"testing"
	"time"

	"github.com/alpacahq/gobroker/dbtest"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/service/assetcache"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/marketstore/utils"
	"github.com/alpacahq/marketstore/utils/io"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type BarTestSuite struct {
	dbtest.Suite
	asset *models.Asset
}

func TestBarTestSuite(t *testing.T) {
	suite.Run(t, new(BarTestSuite))
}

func (s *BarTestSuite) SetupSuite() {
	s.SetupDB()

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
}

func (s *BarTestSuite) TearDownSuite() {
	s.TeardownDB()
}

func (s *BarTestSuite) TestGetByID() {
	srv := barService{
		tx:  db.DB(),
		now: func() time.Time { return clock.Now() },
		mktsdb: func(name string, args interface{}) (io.ColumnSeriesMap, error) {
			return nil, nil
		},
		assetcache: assetcache.GetAssetCache(),
	}

	start := clock.Now()
	end := clock.Now()
	limit := 10

	assetBars, err := srv.GetByID(s.asset.IDAsUUID(), "1Min", &start, &end, &limit)
	assert.Len(s.T(), assetBars.Bars, 0)
	assert.Nil(s.T(), err)

	assetBars, err = srv.GetByID(uuid.Must(uuid.NewV4()), "1Min", &start, &end, &limit)
	assert.Nil(s.T(), assetBars)
	assert.NotNil(s.T(), err)

	srv.mktsdb = func(name string, args interface{}) (io.ColumnSeriesMap, error) {
		m := io.NewColumnSeriesMap()
		cs := io.NewColumnSeries()
		cs.AddColumn("Epoch", []int64{start.Unix() + 60, start.Unix() + 120, start.Unix() + 180, start.Unix() + 240})
		cs.AddColumn("Open", []float32{1.0, 2.0, 3.0, 4.0})
		cs.AddColumn("High", []float32{1.0, 2.0, 3.0, 4.0})
		cs.AddColumn("Low", []float32{1.0, 2.0, 3.0, 4.0})
		cs.AddColumn("Close", []float32{1.0, 2.0, 3.0, 4.0})
		cs.AddColumn("Volume", []int32{1, 2, 3, 4})
		m.AddColumnSeries(*io.NewTimeBucketKey("AAPL/1Min/OHLCV"), cs)
		return m, nil
	}

	assetBars, err = srv.GetByID(s.asset.IDAsUUID(), "1Min", &start, &end, &limit)
	assert.Len(s.T(), assetBars.Bars, 4)
	assert.Nil(s.T(), err)
}

func (s *BarTestSuite) TestGetByIDs() {
	srv := barService{
		tx:  db.DB(),
		now: func() time.Time { return clock.Now() },
		mktsdb: func(name string, args interface{}) (io.ColumnSeriesMap, error) {
			return nil, nil
		},
		assetcache: assetcache.GetAssetCache(),
	}

	start := clock.Now()
	end := clock.Now()
	limit := 10

	bars, err := srv.GetByIDs([]uuid.UUID{s.asset.IDAsUUID()}, "1Min", &start, &end, &limit)
	assert.Len(s.T(), bars, 0)
	assert.Nil(s.T(), err)

	bars, err = srv.GetByIDs([]uuid.UUID{uuid.Must(uuid.NewV4())}, "1Min", &start, &end, &limit)
	assert.NotNil(s.T(), bars)
	assert.Len(s.T(), bars, 0)
	assert.Nil(s.T(), err)

	srv.mktsdb = func(name string, args interface{}) (io.ColumnSeriesMap, error) {
		m := io.NewColumnSeriesMap()
		cs := io.NewColumnSeries()
		cs.AddColumn("Epoch", []int64{start.Unix() + 60, start.Unix() + 120, start.Unix() + 180, start.Unix() + 240})
		cs.AddColumn("Open", []float32{1.0, 2.0, 3.0, 4.0})
		cs.AddColumn("High", []float32{1.0, 2.0, 3.0, 4.0})
		cs.AddColumn("Low", []float32{1.0, 2.0, 3.0, 4.0})
		cs.AddColumn("Close", []float32{1.0, 2.0, 3.0, 4.0})
		cs.AddColumn("Volume", []int32{1, 2, 3, 4})
		m.AddColumnSeries(*io.NewTimeBucketKey("AAPL/1Min/OHLCV"), cs)
		return m, nil
	}

	bars, err = srv.GetByIDs([]uuid.UUID{s.asset.IDAsUUID()}, "1Min", &start, &end, &limit)
	assert.NotNil(s.T(), bars)
	assert.Len(s.T(), bars, 1)
	assert.Len(s.T(), bars[0].Bars, 4)
	assert.Nil(s.T(), err)
}

func (s *BarTestSuite) TestAggregateTrim1DExclude() {
	start := time.Date(2018, 4, 2, 0, 0, 0, 0, calendar.NY)
	end := time.Date(2018, 4, 5, 0, 0, 0, 0, calendar.NY)
	limit := 10

	srv := barService{
		tx:  db.DB(),
		now: func() time.Time { return end.Add(14 * time.Hour) },
		mktsdb: func(name string, args interface{}) (io.ColumnSeriesMap, error) {
			return nil, nil
		},
		assetcache: assetcache.GetAssetCache(),
	}

	bars, err := srv.GetByIDs([]uuid.UUID{s.asset.IDAsUUID()}, "1D", &start, &end, &limit)
	assert.Len(s.T(), bars, 0)
	assert.Nil(s.T(), err)
	srv.mktsdb = func(name string, args interface{}) (io.ColumnSeriesMap, error) {
		m := io.NewColumnSeriesMap()
		cs := io.NewColumnSeries()
		cs.AddColumn("Epoch", []int64{
			start.Truncate(utils.Day).Unix(),
			start.Truncate(utils.Day).Add(utils.Day).Unix(),
			start.Truncate(utils.Day).Add(2 * utils.Day).Unix(),
			start.Truncate(utils.Day).Add(3 * utils.Day).Unix()})
		cs.AddColumn("Open", []float32{1.0, 2.0, 3.0, 4.0})
		cs.AddColumn("High", []float32{1.0, 2.0, 3.0, 4.0})
		cs.AddColumn("Low", []float32{1.0, 2.0, 3.0, 4.0})
		cs.AddColumn("Close", []float32{1.0, 2.0, 3.0, 4.0})
		cs.AddColumn("Volume", []int32{1, 2, 3, 4})
		m.AddColumnSeries(*io.NewTimeBucketKey("AAPL/1D/OHLCV"), cs)

		minCs := io.NewColumnSeries()
		minCs.AddColumn("Epoch", []int64{end.Add(7 * time.Hour).Unix()})
		minCs.AddColumn("Open", []float32{1.0})
		minCs.AddColumn("High", []float32{1.0})
		minCs.AddColumn("Low", []float32{1.0})
		minCs.AddColumn("Close", []float32{1.0})
		minCs.AddColumn("Volume", []int32{1})
		m.AddColumnSeries(*io.NewTimeBucketKey("AAPL/1Min/OHLCV"), cs)
		return m, nil
	}

	bars, err = srv.GetByIDs([]uuid.UUID{s.asset.IDAsUUID()}, "1D", &start, &end, &limit)
	assert.NotNil(s.T(), bars)
	require.Len(s.T(), bars, 1)
	assert.Len(s.T(), bars[0].Bars, 3)
	assert.Nil(s.T(), err)
}

func (s *BarTestSuite) TestAggregateTrim1DInclude() {
	start := time.Date(2018, 4, 2, 4, 0, 0, 0, time.UTC)
	end := time.Date(2018, 4, 5, 4, 0, 0, 0, time.UTC)
	limit := 10

	srv := barService{
		tx:  db.DB(),
		now: func() time.Time { return end.Add(18 * time.Hour) },
		mktsdb: func(name string, args interface{}) (io.ColumnSeriesMap, error) {
			return nil, nil
		},
		assetcache: assetcache.GetAssetCache(),
	}

	bars, err := srv.GetByIDs([]uuid.UUID{s.asset.IDAsUUID()}, "1D", &start, &end, &limit)
	assert.Len(s.T(), bars, 0)
	assert.Nil(s.T(), err)
	srv.mktsdb = func(name string, args interface{}) (io.ColumnSeriesMap, error) {
		m := io.NewColumnSeriesMap()
		cs := io.NewColumnSeries()
		cs.AddColumn("Epoch", []int64{
			start.Truncate(utils.Day).Unix(),
			start.Truncate(utils.Day).Add(utils.Day).Unix(),
			start.Truncate(utils.Day).Add(2 * utils.Day).Unix(),
			start.Truncate(utils.Day).Add(3 * utils.Day).Unix()})
		cs.AddColumn("Open", []float32{1.0, 2.0, 3.0, 4.0})
		cs.AddColumn("High", []float32{1.0, 2.0, 3.0, 4.0})
		cs.AddColumn("Low", []float32{1.0, 2.0, 3.0, 4.0})
		cs.AddColumn("Close", []float32{1.0, 2.0, 3.0, 4.0})
		cs.AddColumn("Volume", []int32{1, 2, 3, 4})
		m.AddColumnSeries(*io.NewTimeBucketKey("AAPL/1D/OHLCV"), cs)

		minCs := io.NewColumnSeries()
		minCs.AddColumn("Epoch", []int64{end.Add(7 * time.Hour).Unix()})
		minCs.AddColumn("Open", []float32{1.0})
		minCs.AddColumn("High", []float32{1.0})
		minCs.AddColumn("Low", []float32{1.0})
		minCs.AddColumn("Close", []float32{1.0})
		minCs.AddColumn("Volume", []int32{1})
		m.AddColumnSeries(*io.NewTimeBucketKey("AAPL/1Min/OHLCV"), cs)
		return m, nil
	}

	bars, err = srv.GetByIDs([]uuid.UUID{s.asset.IDAsUUID()}, "1D", &start, &end, &limit)
	assert.NotNil(s.T(), bars)
	require.Len(s.T(), bars, 1)
	assert.Len(s.T(), bars[0].Bars, 4)
	assert.Nil(s.T(), err)
}

func (s *BarTestSuite) TestAggregateTrim1HExclude() {
	start := time.Date(2018, 4, 5, 11, 0, 0, 0, calendar.NY)
	end := time.Date(2018, 4, 5, 15, 0, 0, 0, calendar.NY)
	limit := 10

	srv := barService{
		tx:  db.DB(),
		now: func() time.Time { return end.Add(-2 * time.Minute) },
		mktsdb: func(name string, args interface{}) (io.ColumnSeriesMap, error) {
			return nil, nil
		},
		assetcache: assetcache.GetAssetCache(),
	}

	bars, err := srv.GetByIDs([]uuid.UUID{s.asset.IDAsUUID()}, "1H", &start, &end, &limit)
	assert.Len(s.T(), bars, 0)
	assert.Nil(s.T(), err)
	srv.mktsdb = func(name string, args interface{}) (io.ColumnSeriesMap, error) {
		m := io.NewColumnSeriesMap()
		cs := io.NewColumnSeries()
		cs.AddColumn("Epoch", []int64{
			start.Unix(),
			start.Add(time.Hour).Unix(),
			start.Add(2 * time.Hour).Unix(),
			start.Add(3 * time.Hour).Unix()})
		cs.AddColumn("Open", []float32{1.0, 2.0, 3.0, 4.0})
		cs.AddColumn("High", []float32{1.0, 2.0, 3.0, 4.0})
		cs.AddColumn("Low", []float32{1.0, 2.0, 3.0, 4.0})
		cs.AddColumn("Close", []float32{1.0, 2.0, 3.0, 4.0})
		cs.AddColumn("Volume", []int32{1, 2, 3, 4})
		m.AddColumnSeries(*io.NewTimeBucketKey("AAPL/1H/OHLCV"), cs)

		minCs := io.NewColumnSeries()
		minCs.AddColumn("Epoch", []int64{end.Add(-2 * time.Minute).Unix()})
		minCs.AddColumn("Open", []float32{1.0})
		minCs.AddColumn("High", []float32{1.0})
		minCs.AddColumn("Low", []float32{1.0})
		minCs.AddColumn("Close", []float32{1.0})
		minCs.AddColumn("Volume", []int32{1})
		m.AddColumnSeries(*io.NewTimeBucketKey("AAPL/1Min/OHLCV"), cs)
		return m, nil
	}

	bars, err = srv.GetByIDs([]uuid.UUID{s.asset.IDAsUUID()}, "1H", &start, &end, &limit)
	assert.NotNil(s.T(), bars)
	require.Len(s.T(), bars, 1)
	assert.Len(s.T(), bars[0].Bars, 3)
	assert.Nil(s.T(), err)
}
