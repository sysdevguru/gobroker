package backfiller

import (
	"time"

	"github.com/alpacahq/gobroker/utils/tradingdate"
	"github.com/alpacahq/gopaca/calendar"
)

type Backfiller struct {
	Start tradingdate.TradingDate
	End   tradingdate.TradingDate

	interm *tradingdate.TradingDate
}

func New(frm string, to string) (*Backfiller, error) {
	frmTime, err := time.ParseInLocation("2006/01/02", frm, calendar.NY)
	if err != nil {
		return nil, err
	}
	frmDate, err := tradingdate.New(frmTime)
	if err != nil {
		return nil, err
	}
	toTime, err := time.ParseInLocation("2006/01/02", to, calendar.NY)
	if err != nil {
		return nil, err
	}
	toDate, err := tradingdate.New(toTime)
	if err != nil {
		return nil, err
	}
	return &Backfiller{
		Start: *frmDate,
		End:   *toDate,
	}, nil
}

func NewWithTradingDate(frm, to tradingdate.TradingDate) *Backfiller {
	return &Backfiller{
		Start: frm,
		End:   to,
	}
}

func (b *Backfiller) Next() bool {
	if b.interm == nil {
		b.interm = &b.Start
		return true
	}
	interm := b.interm.Next()
	if interm.After(b.End) {
		return false
	}
	b.interm = &interm
	return true
}

func (b *Backfiller) Value() tradingdate.TradingDate {
	return *b.interm
}
