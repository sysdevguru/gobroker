package models

import (
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
)

type CorporateAction struct {
	AssetID uuid.UUID                `json:"asset_id" gorm:"primary_key" sql:"type:uuid;"`
	Type    enum.CorporateActionType `json:"type" gorm:"primary_key" sql:"type:text"`
	Date    string                   `json:"date" gorm:"primary_key" sql:"type:date"`
	// ratio by which shares were split
	StockFactor *decimal.Decimal `json:"stock_factor" gorm:"type:decimal;"`
	// $ amount per share to pay out to to account holding position
	CashFactor *decimal.Decimal `json:"cash_factor" gorm:"type:decimal;"`
}
