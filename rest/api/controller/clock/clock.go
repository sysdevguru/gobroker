package clock

import (
	"time"

	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/clock"
)

type ClockReading struct {
	Timestamp time.Time `json:"timestamp"`
	IsOpen    bool      `json:"is_open"`
	NextOpen  time.Time `json:"next_open"`
	NextClose time.Time `json:"next_close"`
}

func Get(ctx api.Context) {

	now := clock.Now().In(calendar.NY)

	ctx.Respond(
		ClockReading{
			Timestamp: now,
			IsOpen:    calendar.IsMarketOpen(now),
			NextOpen:  calendar.NextOpen(now),
			NextClose: calendar.NextClose(now),
		},
	)
}
