package pfhistory

import (
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/marketstore/utils/io"
	"github.com/alpacahq/polycache/structures"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type PfhistoryTestSuite struct {
	suite.Suite
}

func TestPfhistoryTestSuite(t *testing.T) {
	suite.Run(t, new(PfhistoryTestSuite))
}

// Fake position
type FPosition struct {
	symbol   string
	quantity decimal.Decimal
}

func (p FPosition) GetSymbol() string {
	return p.symbol
}

func (p FPosition) GetQty() decimal.Decimal {
	return p.quantity
}

// Fake Order
type FOrder struct {
	models.Order
}

func (o FOrder) GetSymbol() string {
	return o.Symbol
}

func (o FOrder) GetQty() decimal.Decimal {
	return *o.FilledQty
}

func (o FOrder) GetSide() enum.Side {
	return o.Side
}

func (o FOrder) GetFilledAt() *time.Time {
	return o.FilledAt
}

func (o FOrder) GetPrice() decimal.Decimal {
	return *o.FilledAvgPrice
}

func NewFakePosition(symbol string, quantity float64) Position {
	dquant := decimal.NewFromFloat(quantity)
	return FPosition{
		symbol:   symbol,
		quantity: dquant,
	}
}

func NewFakeOrder(symbol string, qty int, price float32, timestamp string, side enum.Side) Order {
	parsed, _ := time.ParseInLocation("2006-01-02 15:04", timestamp, calendar.NY)
	np := decimal.NewFromFloat(float64(price))
	filledQty := decimal.New(int64(qty), 0)
	o := models.Order{
		Symbol:         symbol,
		FilledQty:      &filledQty,
		FilledAt:       &parsed,
		FilledAvgPrice: &np,
		Side:           side,
	}
	return FOrder{o}
}

func NewFakeCS(epochs calendar.DateRange, values []float32) *io.ColumnSeries {
	cs := io.NewColumnSeries()
	cs.AddColumn("Epoch", epochs.Unix())
	cs.AddColumn("Close", values)
	return cs
}

func (s *PfhistoryTestSuite) TestComputePL() {

	positions := make([]Position, 2)
	positions[0] = NewFakePosition("AAPL", 1)
	positions[1] = NewFakePosition("NVDA", 1)

	orders := make([]Order, 3)
	orders[0] = NewFakeOrder("AAPL", 1, 15, "2017-11-20 9:36", "buy")
	orders[1] = NewFakeOrder("NVDA", 1, 2050, "2017-11-20 9:45", "sell")
	orders[2] = NewFakeOrder("AAPL", 1, 20, "2017-11-20 9:54", "sell")

	begin := time.Date(2017, 11, 20, 9, 30, 0, 0, calendar.NY)
	end := time.Date(2017, 11, 20, 9, 55, 0, 0, calendar.NY)
	timestamps, _ := calendar.NewRange(begin, end, calendar.Min5)

	assert.Len(s.T(), timestamps, 5)

	cs1 := NewFakeCS(timestamps, []float32{11, 12, 10, 12, 10})
	cs2 := NewFakeCS(timestamps, []float32{2000, 2100, 2000, 2100, 2000})
	csm := io.ColumnSeriesMap{}
	aaplTbk := io.NewTimeBucketKey("AAPL/5Min/OHLCV", io.DefaultTimeBucketSchema)
	nvdaTbk := io.NewTimeBucketKey("NVDA/5Min/OHLCV", io.DefaultTimeBucketSchema)
	csm.AddColumnSeries(*aaplTbk, cs1)
	csm.AddColumnSeries(*nvdaTbk, cs2)

	beginningPrices := map[string]structures.Trade{}
	beginningPrices["AAPL"] = structures.Trade{Price: 10}
	beginningPrices["NVDA"] = structures.Trade{Price: 2000}

	hist := ComputePL(positions, orders, csm, beginningPrices, begin, end, end)

	fmt.Println(hist)
	// :30
	assert.True(s.T(), hist.Close[0].Equal(decimal.NewFromFloat(0+1)))
	// :35
	assert.True(s.T(), hist.Close[1].Equal(decimal.NewFromFloat(100+(2-3))))
	// :40
	assert.True(s.T(), hist.Close[2].Equal(decimal.NewFromFloat(0+(0-5))))
	// :45
	assert.True(s.T(), hist.Close[3].Equal(decimal.NewFromFloat(50+(2-3))))
	// :50
	assert.True(s.T(), hist.Close[4].Equal(decimal.NewFromFloat(50+(5+0))))

	// With missing candles
	timestamps = []time.Time{
		time.Date(2017, 11, 20, 9, 40, 0, 0, calendar.NY),
		time.Date(2017, 11, 20, 9, 45, 0, 0, calendar.NY),
	}
	cs2 = NewFakeCS(timestamps, []float32{2000, 2100})
	csm = io.ColumnSeriesMap{}
	csm.AddColumnSeries(*aaplTbk, cs1)
	csm.AddColumnSeries(*nvdaTbk, cs2)

	beginningPrices["NVDA"] = structures.Trade{Price: 2000}

	hist = ComputePL(positions, orders, csm, beginningPrices, begin, end, end)

	assert.True(s.T(), hist.Close[0].Equal(decimal.NewFromFloat(0+1)))
	assert.True(s.T(), hist.Close[1].Equal(decimal.NewFromFloat(0+(2-3))))
	assert.True(s.T(), hist.Close[2].Equal(decimal.NewFromFloat(0+(0-5))))
	assert.True(s.T(), hist.Close[3].Equal(decimal.NewFromFloat(50+(2-3))))
	assert.True(s.T(), hist.Close[4].Equal(decimal.NewFromFloat(50+(5+0))))

	// With no candles w/ order (exceptional though)
	timestamps = []time.Time{}
	cs2 = NewFakeCS(timestamps, []float32{})
	csm = io.ColumnSeriesMap{}
	csm.AddColumnSeries(*aaplTbk, cs1)
	csm.AddColumnSeries(*nvdaTbk, cs2)

	beginningPrices["NVDA"] = structures.Trade{Price: 2000}

	hist = ComputePL(positions, orders, csm, beginningPrices, begin, end, end)

	assert.True(s.T(), hist.Close[0].Equal(decimal.NewFromFloat(0+1)))
	assert.True(s.T(), hist.Close[1].Equal(decimal.NewFromFloat(0+(2-3))))
	assert.True(s.T(), hist.Close[2].Equal(decimal.NewFromFloat(0+(0-5))))
	assert.True(s.T(), hist.Close[3].Equal(decimal.NewFromFloat(50+(2-3))))
	assert.True(s.T(), hist.Close[4].Equal(decimal.NewFromFloat(50+(5+0))))

	// w/ during market open
	hist = ComputePL(positions, orders, csm, beginningPrices, begin, end,
		time.Date(2017, 11, 20, 9, 45, 0, 0, calendar.NY))
	assert.NotNil(s.T(), hist.Close[3])
	assert.Nil(s.T(), hist.Close[4])

	// w/ candles w/o orders w/ market open
	orders = []Order{}
	hist = ComputePL(positions, orders, csm, beginningPrices, begin, end,
		time.Date(2017, 11, 20, 9, 45, 0, 0, calendar.NY))
	assert.True(s.T(), hist.Close[0].Equal(decimal.NewFromFloat(0+1)))
	assert.True(s.T(), hist.Close[1].Equal(decimal.NewFromFloat(0+2)))
	assert.True(s.T(), hist.Close[2].Equal(decimal.NewFromFloat(0+(0))))
	assert.True(s.T(), hist.Close[3].Equal(decimal.NewFromFloat(0+(2))))
	assert.Nil(s.T(), hist.Close[4])

	// w/ delayed candles w/ market open
	timestamps = []time.Time{
		time.Date(2017, 11, 20, 9, 30, 0, 0, calendar.NY),
		time.Date(2017, 11, 20, 9, 35, 0, 0, calendar.NY),
	}
	cs2 = NewFakeCS(timestamps, []float32{2000, 2100})
	csm = io.ColumnSeriesMap{}
	csm.AddColumnSeries(*aaplTbk, cs1)
	csm.AddColumnSeries(*nvdaTbk, cs2)

	hist = ComputePL(positions, orders, csm, beginningPrices, begin, end,
		time.Date(2017, 11, 20, 9, 45, 0, 0, calendar.NY))

	assert.True(s.T(), hist.Close[0].Equal(decimal.NewFromFloat(0+1)))
	assert.True(s.T(), hist.Close[1].Equal(decimal.NewFromFloat(100+2)))
	assert.True(s.T(), hist.Close[2].Equal(decimal.NewFromFloat(100+(0))))
	assert.True(s.T(), hist.Close[3].Equal(decimal.NewFromFloat(100+(2))))
	assert.Nil(s.T(), hist.Close[4])

	// w/ no positions
	positions = []Position{}
	hist = ComputePL(positions, orders, csm, beginningPrices, begin, end,
		time.Date(2017, 11, 20, 9, 45, 0, 0, calendar.NY))
	assert.True(s.T(), hist.Close[0].Equal(decimal.NewFromFloat(0)))
	assert.True(s.T(), hist.Close[1].Equal(decimal.NewFromFloat(0)))
	assert.True(s.T(), hist.Close[2].Equal(decimal.NewFromFloat(0)))
	assert.True(s.T(), hist.Close[3].Equal(decimal.NewFromFloat(0)))
	assert.Nil(s.T(), hist.Close[4])

	// w/ orders entried and exit at the same bucket, w/o position at first
	orders = make([]Order, 2)
	orders[0] = NewFakeOrder("AAPL", 1, 13, "2017-11-20 9:36", "buy")
	orders[1] = NewFakeOrder("AAPL", 1, 15, "2017-11-20 9:37", "sell")
	hist = ComputePL(positions, orders, csm, beginningPrices, begin, end,
		time.Date(2017, 11, 20, 9, 45, 0, 0, calendar.NY))

	assert.True(s.T(), hist.Close[0].Equal(decimal.NewFromFloat(0)))
	assert.True(s.T(), hist.Close[1].Equal(decimal.NewFromFloat(2)))
	assert.True(s.T(), hist.Close[2].Equal(decimal.NewFromFloat(2)))
	assert.True(s.T(), hist.Close[3].Equal(decimal.NewFromFloat(2)))
	assert.Nil(s.T(), hist.Close[4])
}
