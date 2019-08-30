package price

import (
	"github.com/shopspring/decimal"
)

var oneDollar = decimal.New(1, 0)

// NOTE: SEC regulations being referred to are located at:
// https://www.sec.gov/divisions/marketreg/subpenny612faq.htm

// IsValid returns whether or not the supplied price is
// valid as per SEC regulations
func IsValid(px decimal.Decimal) bool {
	if px.LessThan(oneDollar) {
		if len(px.String()) > len(px.StringFixed(4)) {
			return false
		}
	} else {
		if len(px.String()) > len(px.StringFixed(2)) {
			return false
		}
	}

	return true
}

// FormatForOrder formats the price as per SEC regulations,
// and truncates the value for order purposes
func FormatForOrder(px decimal.Decimal) (decimal.Decimal, int32) {
	if px.LessThan(oneDollar) {
		return px.Truncate(4), 4
	}

	return px.Truncate(2), 2
}

// FormatForCalc formats the price as per SEC regulations,
// and rounds the value for calculation purposes
func FormatForCalc(px decimal.Decimal) (decimal.Decimal, int32) {
	if px.LessThan(oneDollar) {
		return px.Round(4), 4
	}

	return px.Round(2), 2
}

// FormatFloat32ForOrder returns the float32 price as decimal,
// formatted as per SEC regulations and truncated
func FormatFloat32ForOrder(px float32) decimal.Decimal {
	p, _ := FormatForOrder(decimal.NewFromFloat32(px))
	return p
}

// FormatFloat64ForOrder returns the float64 price as decimal,
// formatted as per SEC regulations and truncated
func FormatFloat64ForOrder(px float64) decimal.Decimal {
	p, _ := FormatForOrder(decimal.NewFromFloat(px))
	return p
}

// FormatFloat32ForCalc returns the float32 price as decimal,
// formatted as per SEC regulations and rounded
func FormatFloat32ForCalc(px float32) decimal.Decimal {
	p, _ := FormatForCalc(decimal.NewFromFloat32(px))
	return p
}

// FormatFloat64ForCalc returns the float64 price as decimal,
// formatted as per SEC regulations and rounded
func FormatFloat64ForCalc(px float64) decimal.Decimal {
	p, _ := FormatForCalc(decimal.NewFromFloat(px))
	return p
}
