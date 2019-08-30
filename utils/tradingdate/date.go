package tradingdate

import (
	"fmt"
	"time"

	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/clock"
)

// TradingDate represents date
type TradingDate struct {
	timestamp time.Time
}

func (t TradingDate) String() string {
	return t.timestamp.Format("2006-01-02")
}

// MarketOpen returns market opening time at that trading day
func (t TradingDate) MarketOpen() time.Time {
	mo := calendar.MarketOpen(t.timestamp)
	return *mo
}

// MarketClose returns market closing time at that trading day
func (t TradingDate) MarketClose() time.Time {
	mc := calendar.MarketClose(t.timestamp)
	return *mc
}

// SessionEnd returns time.Time at the beginnging of session hours
func (t TradingDate) SessionOpen() time.Time {
	y, m, d := t.timestamp.Date()
	mo := time.Date(y, m, d, 7, 0, 0, 0, calendar.NY)
	return mo
}

// SessionClose returns time.Time at the end of session hours
func (t TradingDate) SessionClose() time.Time {
	y, m, d := t.timestamp.Date()
	mo := time.Date(y, m, d, 19, 0, 0, 0, calendar.NY)
	return mo
}

// Prev returns previous trading date
func (t TradingDate) Prev() TradingDate {
	return TradingDate{timestamp: calendar.PrevClose(t.timestamp)}
}

// Next returns next trading date
func (t TradingDate) Next() TradingDate {
	return TradingDate{timestamp: calendar.NextClose(t.timestamp)}
}

func (t TradingDate) After(a TradingDate) bool {
	return t.timestamp.After(a.timestamp)
}

func (t TradingDate) Before(a TradingDate) bool {
	return t.timestamp.Before(a.timestamp)
}

func (t TradingDate) Equals(a TradingDate) bool {
	return t.timestamp.Equal(a.timestamp)
}

// DaysAgo returns N days ago trading day
func (t TradingDate) DaysAgo(days int) TradingDate {
	i := 0
	current := t
	for i < days {
		current = current.Prev()
		i++
	}
	return current
}

// New generate TradingDate struct for that day
func New(t time.Time) (*TradingDate, error) {
	if calendar.IsMarketDay(t) {
		mc := calendar.MarketClose(t)
		date := TradingDate{timestamp: *mc}
		return &date, nil
	}
	return nil, fmt.Errorf("no trading day")
}

// Last returns last trading date for that timestamp
func Last(t time.Time) TradingDate {
	if calendar.IsMarketDay(t) {
		mo := calendar.MarketOpen(t)
		if t.After(*mo) || t.Equal(*mo) {
			mc := calendar.MarketClose(t)
			return TradingDate{timestamp: *mc}
		}
	}
	prev := calendar.PrevClose(t)
	return TradingDate{timestamp: prev}
}

func NewFromDate(y int, m time.Month, d int) (*TradingDate, error) {
	t := time.Date(y, m, d, 0, 0, 0, 0, calendar.NY)
	if calendar.IsMarketDay(t) {
		mc := calendar.MarketClose(t)
		date := TradingDate{timestamp: *mc}
		return &date, nil
	}
	return nil, fmt.Errorf("no trading day")
}

// Current returns current trading date. Trading date changes on the time of market open.
func Current() TradingDate {
	t := clock.Now().In(calendar.NY)
	return Last(t)
}
