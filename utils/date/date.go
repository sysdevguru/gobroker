package date

import (
	"database/sql/driver"
	"time"

	"cloud.google.com/go/civil"
)

type Date struct {
	civil.Date
}

type DateLayout string

func DateOf(t time.Time) Date {
	var d Date
	d.Date.Year, d.Date.Month, d.Date.Day = t.Date()
	return d
}

func Parse(layout string, value string) (Date, error) {
	t, err := time.Parse(layout, value)
	if err != nil {
		return Date{}, err
	}
	return DateOf(t), nil
}

func ParseDate(s string) (Date, error) {
	if d, err := civil.ParseDate(s); err != nil {
		return Date{}, err
	} else {
		return Date{Date: d}, nil
	}
}

func (d Date) Value() (driver.Value, error) {
	return d.Date.String(), nil
}

func (d *Date) Scan(value interface{}) error {
	if value == nil {
		d = nil
		return nil
	}

	t := value.(time.Time)
	if t.IsZero() {
		d = nil
		return nil
	}

	d.Date = civil.DateOf(t)
	return nil
}
