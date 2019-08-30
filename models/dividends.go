package models

import (
	"time"

	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/utils/date"
	"github.com/shopspring/decimal"
)

type Dividend struct {
	AccountID                   string                                   `json:"account_id" gorm:"primary_key" sql:"type:uuid references accounts(id);"`
	AssetID                     string                                   `json:"asset_id" gorm:"primary_key" sql:"type:uuid references assets(id);"`
	Symbol                      string                                   `json:"symbol" sql:"type:varchar(12)"`                    // recording purpose
	CUSIP                       string                                   `json:"cusip" gorm:"column:cusip" sql:"type:varchar(12)"` // recording purpose
	ExchangeDate                date.Date                                `json:"exchange_date" gorm:"primary_key" sql:"type:date"`
	RecordDate                  date.Date                                `json:"record_date" sql:"type:date"`
	PayDate                     date.Date                                `json:"pay_date" sql:"type:date"`
	DividendRate                decimal.Decimal                          `json:"dividend_rate" gorm:"type:decimal"`
	MaturityDate                *date.Date                               `json:"maturity_date" sql:"type:date"`
	CouponDate                  *string                                  `json:"coupon_date" sql:"type:date"`
	FirstCouponDate             *date.Date                               `json:"first_coupon_date" sql:"type:date"`
	CouponRate                  *decimal.Decimal                         `json:"coupon_rate" sql:"type:text"`
	IssueDate                   *date.Date                               `json:"issue_date" sql:"type:date"`
	Position                    decimal.Decimal                          `json:"position" sql:"type:decimal"`
	PositionQuantityLongOrShort enum.DividendPositionQuantityLongOrShort `json:"position_quantity_long_or_short" sql:"type:text"`
	DividendInterest            *decimal.Decimal                         `json:"dividend_interest" gorm:"type:decimal"`
	WithHoldAmount              *decimal.Decimal                         `json:"withhold_amount" gorm:"type:decimal"`
	PayedAt                     *time.Time                               `json:"payed_at"` // will be updated from cash activity report.
	LastReportDate              date.Date                                `json:"last_report_date" sql:"type:date"`
	CreatedAt                   time.Time                                `json:"-"`
	UpdatedAt                   time.Time                                `json:"-"`
}
