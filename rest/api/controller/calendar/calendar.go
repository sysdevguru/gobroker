package calendar

import (
	"time"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gopaca/calendar"
)

var history calendar.MarketHistory

func init() {
	history = calendar.History()
}

func Get(ctx api.Context) {
	startParam := ctx.URLParam("start")
	endParam := ctx.URLParam("end")

	var start, end *time.Time

	if startParam != "" {
		s, err := time.ParseInLocation("2006-01-02", startParam, calendar.NY)
		if err != nil {
			ctx.RespondError(gberrors.InvalidRequestParam.WithMsg(
				"invalid start date"))
			return
		}
		start = &s
	}

	if endParam != "" {
		e, err := time.ParseInLocation("2006-01-02", endParam, calendar.NY)
		if err != nil {
			ctx.RespondError(gberrors.InvalidRequestParam.WithMsg(
				"invalid end date"))
			return
		}
		end = &e
	}

	slc, err := history.Slice(start, end)
	if err != nil {
		ctx.RespondError(gberrors.InvalidRequestParam.WithMsg(
			"start must be before end"))
		return
	}

	ctx.Respond(slc)
}
