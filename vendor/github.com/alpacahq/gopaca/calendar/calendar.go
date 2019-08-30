package calendar

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

var NY, _ = time.LoadLocation("America/New_York")

const Day = 24 * time.Hour
const Week = 7 * Day
const Year = 365 * Day

type calendar struct {
	days map[string]string
}

type marketTime struct {
	calendar calendar
}

var MarketTime *marketTime

func init() {
	build()
}

func build() {
	cal := calendar{days: make(map[string]string)}
	market := marketTime{calendar: cal}
	cFile := make(map[string][]string)
	json.Unmarshal([]byte(calendarJSON), &cFile)
	for _, date := range cFile["non_trading_days"] {
		market.calendar.days[date] = "closed"
	}
	for _, date := range cFile["early_closes"] {
		market.calendar.days[date] = "early_close"
	}
	MarketTime = &market
}

func MarketOpen(t time.Time) *time.Time {
	t = t.In(NY)

	if !IsMarketDay(t) {
		return nil
	}

	mo := time.Date(t.Year(), t.Month(), t.Day(), 9, 30, 0, 0, NY)

	return &mo
}

func MarketClose(t time.Time) *time.Time {
	if !IsMarketDay(t) {
		return nil
	}

	o := MarketOpen(t)
	if o != nil {
		c := NextClose(*o)
		return &c
	}
	panic("Never reaches here")
}

// IsMarketDay check if today is a trading day or not.
func IsMarketDay(t time.Time) bool {
	t = t.In(NY)

	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return false
	}
	if state, ok := MarketTime.calendar.days[t.Format("2006-01-02")]; ok {
		switch state {
		case "early_close":
			return true
		case "closed":
			return false
		}
	}
	return true
}

func IsMarketOpen(t time.Time) bool {
	t = t.In(NY)

	open := time.Date(t.Year(), t.Month(), t.Day(), 9, 30, 0, 0, NY)
	normalClose := time.Date(t.Year(), t.Month(), t.Day(), 16, 0, 0, 0, NY)
	earlyClose := time.Date(t.Year(), t.Month(), t.Day(), 13, 0, 0, 0, NY)
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return false
	}
	if state, ok := MarketTime.calendar.days[t.Format("2006-01-02")]; ok {
		switch state {
		case "early_close":
			if t.Before(open) || t.After(earlyClose) || t.Equal(earlyClose) {
				return false
			} else {
				return true
			}
		case "closed":
			return false
		}
	} else {
		if t.Before(open) || t.After(normalClose) || t.Equal(normalClose) {
			return false
		} else {
			return true
		}
	}
	return true
}

func NextOpen(t time.Time) time.Time {
	t = t.In(NY)
	open := time.Date(t.Year(), t.Month(), t.Day(), 9, 30, 0, 0, NY)

	state, _ := MarketTime.calendar.days[t.Format("2006-01-02")]

	switch {
	case (t.Weekday() == time.Saturday || t.Weekday() == time.Sunday):
		return findNextOpen(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, NY))
	case state == "closed":
		return findNextOpen(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, NY))
	default:
		if t.Before(open) {
			return open
		}
		return findNextOpen(time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, NY))
	}
}

func findNextOpen(t time.Time) time.Time {
	o := t
	for {
		o = o.AddDate(0, 0, 1)
		if o.Weekday() == time.Saturday || o.Weekday() == time.Sunday {
			continue
		}
		state, ok := MarketTime.calendar.days[o.Format("2006-01-02")]
		if !ok || state == "early_close" {
			return time.Date(o.Year(), o.Month(), o.Day(), 9, 30, 0, 0, NY)
		}
	}
}

func findNextClose(t time.Time) time.Time {
	c := t
	for {
		c = c.AddDate(0, 0, 1)
		if c.Weekday() == time.Saturday || c.Weekday() == time.Sunday {
			continue
		}
		if state, ok := MarketTime.calendar.days[c.Format("2006-01-02")]; ok {
			if state == "early_close" {
				return time.Date(c.Year(), c.Month(), c.Day(), 13, 0, 0, 0, NY)
			}
		} else {
			return time.Date(c.Year(), c.Month(), c.Day(), 16, 0, 0, 0, NY)
		}
	}
}

func NextClose(t time.Time) time.Time {
	t = t.In(NY)
	switch {
	case IsMarketDay(t):
		state := MarketTime.calendar.days[t.Format("2006-01-02")]
		var cls time.Time
		if state == "early_close" {
			cls = time.Date(t.Year(), t.Month(), t.Day(), 13, 0, 0, 0, NY)
		} else {
			cls = time.Date(t.Year(), t.Month(), t.Day(), 16, 0, 0, 0, NY)
		}
		if t.Before(cls) {
			return cls
		}
	}
	return findNextClose(t)
}

func findPrevClose(t time.Time) time.Time {
	c := t
	for {
		c = c.AddDate(0, 0, -1)
		if c.Weekday() == time.Saturday || c.Weekday() == time.Sunday {
			continue
		}
		if state, ok := MarketTime.calendar.days[c.Format("2006-01-02")]; ok {
			if state == "early_close" {
				return time.Date(c.Year(), c.Month(), c.Day(), 13, 0, 0, 0, NY)
			}
		} else {
			return time.Date(c.Year(), c.Month(), c.Day(), 16, 0, 0, 0, NY)
		}
	}
}

func PrevClose(t time.Time) time.Time {
	t = t.In(NY)

	normalClose := time.Date(t.Year(), t.Month(), t.Day(), 16, 0, 0, 0, NY)
	state, nonTradingDay := MarketTime.calendar.days[t.Format("2006-01-02")]
	nonTradingDay = nonTradingDay || (t.Weekday() == time.Saturday || t.Weekday() == time.Sunday)
	switch {
	case (!nonTradingDay) && t.After(normalClose):
		return normalClose
	case nonTradingDay:
		earlyClose := time.Date(t.Year(), t.Month(), t.Day(), 13, 0, 0, 0, NY)
		if state == "early_close" && t.After(earlyClose) {
			return earlyClose
		}
		return findPrevClose(t)
	default:
		return findPrevClose(t)
	}
}

type TradingDay struct {
	Epoch int64  `json:"-"`
	Date  string `json:"date"`
	Open  string `json:"open"`
	Close string `json:"close"`
}

type MarketHistory []TradingDay

func (m MarketHistory) Slice(start, end *time.Time) (MarketHistory, error) {
	if start != nil && end != nil && start.After(*end) {
		return nil, fmt.Errorf("start must be before end")
	}

	i, j := 0, len(m)-1

	if start != nil {
		epoch := time.Date(
			start.Year(), start.Month(), start.Day(),
			0, 0, 0, 0, start.Location()).Unix()

		for i < len(m) {
			if m[i].Epoch >= epoch {
				break
			}
			i++
		}
	}

	if end != nil {
		epoch := time.Date(
			end.Year(), end.Month(), end.Day(),
			0, 0, 0, 0, end.Location()).Unix()

		for j > i {
			if m[j].Epoch <= epoch {
				break
			}
			j--
		}
	}

	return m[i : j+1], nil
}

const marketOpen = "09:30"
const normalClose = "16:00"
const earlyClose = "13:00"

// History returns a list of TradingDay representing all of the
// US market trading hours from 1970 to 2029.
func History() (history MarketHistory) {
	start := time.Date(1970, 1, 1, 0, 0, 0, 0, NY)
	end := time.Date(2029, 12, 24, 0, 0, 0, 0, NY)

	iter := NewIterator(start, end, Day)

	for iter.Next() {
		t := iter.Current()

		if IsMarketDay(t) {
			date := t.Format("2006-01-02")

			td := TradingDay{
				Epoch: t.Unix(),
				Date:  date,
				Open:  marketOpen,
				Close: normalClose,
			}

			if state, ok := MarketTime.calendar.days[date]; ok {
				if strings.EqualFold(state, "early_close") {
					td.Close = earlyClose
				}
			}

			history = append(history, td)
		}
	}
	return
}
