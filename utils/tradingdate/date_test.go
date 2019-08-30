package tradingdate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/alpacahq/gopaca/calendar"
	"github.com/stretchr/testify/suite"
)

type TradingDateTestSuite struct {
	suite.Suite
}

func TestTradingDateTestSuite(t *testing.T) {
	suite.Run(t, new(TradingDateTestSuite))
}

func (s *TradingDateTestSuite) TestNew() {
	d := time.Date(2018, 1, 25, 9, 30, 0, 0, calendar.NY)
	t, _ := New(d)
	assert.Equal(s.T(), t.String(), "2018-01-25")

	d = time.Date(2018, 1, 25, 9, 29, 0, 0, calendar.NY)
	t, _ = New(d)
	assert.Equal(s.T(), t.String(), "2018-01-25")

	d = time.Date(2018, 1, 21, 9, 29, 0, 0, calendar.NY)
	_, err := New(d)
	assert.Equal(s.T(), err.Error(), "no trading day")
}

func (s *TradingDateTestSuite) TestLast() {

	// Before market open
	d := time.Date(2018, 1, 25, 9, 29, 59, 999999, calendar.NY)
	t := Last(d)
	assert.Equal(s.T(), t.String(), "2018-01-24")

	// After market open
	d = time.Date(2018, 1, 25, 9, 30, 0, 0, calendar.NY)
	t = Last(d)
	assert.Equal(s.T(), t.String(), "2018-01-25")

	// After market close
	d = time.Date(2018, 1, 25, 16, 00, 0, 0, calendar.NY)
	t = Last(d)
	assert.Equal(s.T(), t.String(), "2018-01-25")

	// End of the weekend
	d = time.Date(2018, 1, 22, 9, 29, 59, 999999, calendar.NY)
	assert.Equal(s.T(), Last(d).String(), "2018-01-19")

	// On Sunday
	d = time.Date(2018, 1, 21, 9, 29, 59, 999999, calendar.NY)
	assert.Equal(s.T(), Last(d).String(), "2018-01-19")
}

func (s *TradingDateTestSuite) TestDaysAgo() {
	d := time.Date(2018, 1, 25, 9, 30, 0, 0, calendar.NY)
	t, _ := New(d)
	assert.Equal(s.T(), t.String(), "2018-01-25")
	assert.Equal(s.T(), t.DaysAgo(1).String(), "2018-01-24")
	assert.Equal(s.T(), t.DaysAgo(5).String(), "2018-01-18")
}

func (s *TradingDateTestSuite) TestNext() {
	d := time.Date(2018, 1, 25, 9, 30, 0, 0, calendar.NY)
	t, _ := New(d)
	assert.Equal(s.T(), t.String(), "2018-01-25")
	assert.Equal(s.T(), t.Next().String(), "2018-01-26")

	d = time.Date(2018, 1, 26, 9, 30, 0, 0, calendar.NY)
	t, _ = New(d)
	assert.Equal(s.T(), t.String(), "2018-01-26")
	assert.Equal(s.T(), t.Next().String(), "2018-01-29")
}

func (s *TradingDateTestSuite) TestPrev() {
	d := time.Date(2018, 1, 25, 9, 30, 0, 0, calendar.NY)
	t, _ := New(d)
	assert.Equal(s.T(), t.String(), "2018-01-25")
	assert.Equal(s.T(), t.Prev().String(), "2018-01-24")

	d = time.Date(2018, 1, 22, 9, 30, 0, 0, calendar.NY)
	t, _ = New(d)
	assert.Equal(s.T(), t.String(), "2018-01-22")
	assert.Equal(s.T(), t.Prev().String(), "2018-01-19")
}
