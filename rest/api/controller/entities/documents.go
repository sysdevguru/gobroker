package entities

import (
	"net/http"
	"time"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gopaca/calendar"
)

type DocumentQuery struct {
	Start time.Time
	End   time.Time
	Type  string
}

func (q *DocumentQuery) Parse(req *http.Request) (err error) {
	params := req.URL.Query()

	s := params.Get("start")
	if s == "" {
		return gberrors.InvalidRequestParam.WithMsg("start date is required")
	}

	q.Start, err = time.ParseInLocation("2006-01-02", s, calendar.NY)
	if err != nil {
		return gberrors.InvalidRequestParam.
			WithMsg("start date is incorrectly formatted").
			WithError(err)
	}

	e := params.Get("end")
	if e == "" {
		return gberrors.InvalidRequestParam.WithMsg("end date is required")
	}

	q.End, err = time.ParseInLocation("2006-01-02", e, calendar.NY)
	if err != nil {
		return gberrors.InvalidRequestParam.
			WithMsg("end date is incorrectly formatted").
			WithError(err)
	}

	if q.Start.IsZero() || q.End.IsZero() || q.Start.After(q.End) {
		return gberrors.InvalidRequestParam.WithMsg("invalid start and end dates")
	}

	q.Type = params.Get("type")
	if q.Type == "" {
		q.Type = "all"
	}

	return nil
}
