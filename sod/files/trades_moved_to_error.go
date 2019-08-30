package files

import (
	"reflect"
	"time"

	"github.com/shopspring/decimal"
)

type SoDTradesMovedToError struct {
	AccountNumber  string           `gorm:"type:varchar(13);index"`
	ErrorAccount   string           `sql:"type:text"`
	AccType        SoDAccountType   `sql:"type:text"`
	Quantity       *decimal.Decimal `gorm:"type:decimal"`
	Price          *decimal.Decimal `gorm:"type:decimal"`
	SecurityNumber string           `sql:"type:text"`
	BuyOrSell      SoDBuySell       `sql:"type:text"`
	TradeDate      *string          `sql:"type:date"`
	TradeNumber    string           `sql:"type:text"`
	RestrictedDesc string           `sql:"type:text"`
}

type TradesMovedToErrorReport struct {
	trades []SoDTradesMovedToError
}

func (t *TradesMovedToErrorReport) ExtCode() string {
	return "EXT596"
}

func (t *TradesMovedToErrorReport) Delimiter() string {
	return ","
}

func (t *TradesMovedToErrorReport) Header() bool {
	return false
}

func (t *TradesMovedToErrorReport) Extension() string {
	return "txt"
}

func (t *TradesMovedToErrorReport) Value() reflect.Value {
	return reflect.ValueOf(t.trades)
}

func (t *TradesMovedToErrorReport) Append(v interface{}) {
	t.trades = append(t.trades, v.(SoDTradesMovedToError))
}

func (t *TradesMovedToErrorReport) Sync(asOf time.Time) (uint, uint) {
	// trades := make([]interface{}, len(t.trades))
	// for i := 0; i < len(trades); i++ {
	// 	trades[i] = &models.SoDTradesMovedToErrorModel{
	// 		Model: gorm.Model{},
	// 		SoDTradesMovedToError: t.trades[i],
	// 	}
	// }
	// return db.BatchCreate(trades)
	return 0, 0
}
