package models

import (
	"time"

	"github.com/shopspring/decimal"
)

type Fundamental struct {
	ID                uint            `json:"-" gorm:"primary_key"`
	CreatedAt         time.Time       `json:"-"`
	UpdatedAt         time.Time       `json:"-"`
	DeletedAt         *time.Time      `json:"-"`
	AssetID           string          `json:"asset_id" gorm:"not null;unique_index" sql:"type:uuid;"`
	Symbol            string          `json:"symbol" sql:"type:text"`
	FullName          string          `json:"full_name" sql:"type:text"`
	IndustryName      string          `json:"industry_name" sql:"type:text"`
	IndustryGroup     string          `json:"industry_group" sql:"type:text"`
	Sector            string          `json:"sector" sql:"type:text"`
	PERatio           float32         `json:"pe_ratio"`
	PEGRatio          float32         `json:"peg_ratio"`
	Beta              float32         `json:"beta"`
	EPS               float32         `json:"eps"`
	MarketCap         int64           `json:"market_cap"`
	SharesOutstanding int64           `json:"shares_outstanding"`
	AvgVol            int64           `json:"avg_vol"`
	DivRate           float32         `json:"div_rate"`
	ROE               float32         `json:"roe"`
	ROA               float32         `json:"roa"`
	PS                float32         `json:"ps"`
	PC                float32         `json:"pc"`
	GrossMargin       float32         `json:"gross_margin"`
	FiftyTwoWeekHigh  decimal.Decimal `json:"fifty_two_week_high" sql:"type:decimal"`
	FiftyTwoWeekLow   decimal.Decimal `json:"fifty_two_week_low" sql:"type:decimal"`
	ShortDescription  string          `json:"short_description" sql:"type:text"`
	LongDescription   string          `json:"long_description" sql:"type:text"`
}
