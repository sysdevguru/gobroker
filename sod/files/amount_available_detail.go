package files

import (
	"reflect"
	"time"

	"github.com/shopspring/decimal"
)

type SoDAmountAvailableDetail struct {
	ProcessDate               string           `sql:"type:date"`
	OfficeCode                string           `sql:"type:text"`
	AccountNumber             string           `gorm:"type:varchar(13);index"`
	GrossFundsAvailable       *decimal.Decimal `gorm:"type:decimal"`
	PendingTransfer           *decimal.Decimal `gorm:"type:decimal"`
	RecentDeposit             *decimal.Decimal `gorm:"type:decimal"`
	PendingDebitInterest      *decimal.Decimal `gorm:"type:decimal"`
	PendingDebitDividend      *decimal.Decimal `gorm:"type:decimal"`
	FullyPaidUnsettledBalance *decimal.Decimal `gorm:"type:decimal"`
	NetFundsAvailable         *decimal.Decimal `gorm:"type:decimal"`
	UnknownCol1               *decimal.Decimal `gorm:"type:decimal"`
}

type AmountAvailableDetailReport struct {
	details []SoDAmountAvailableDetail
}

func (a *AmountAvailableDetailReport) ExtCode() string {
	return "EXT997"
}

func (a *AmountAvailableDetailReport) Delimiter() string {
	return ","
}

func (a *AmountAvailableDetailReport) Header() bool {
	return false
}

func (a *AmountAvailableDetailReport) Extension() string {
	return "csv"
}

func (a *AmountAvailableDetailReport) Value() reflect.Value {
	return reflect.ValueOf(a.details)
}

func (a *AmountAvailableDetailReport) Append(v interface{}) {
	a.details = append(a.details, v.(SoDAmountAvailableDetail))
}

func (a *AmountAvailableDetailReport) Sync(asOf time.Time) (uint, uint) {
	// details := make([]interface{}, len(a.details))
	// for i := 0; i < len(details); i++ {
	// 	details[i] = &models.SoDAmountAvailableDetailModel{
	// 		Model: gorm.Model{},
	// 		SoDAmountAvailableDetail: a.details[i],
	// 	}
	// }
	// return db.BatchCreate(details)
	return 0, 0
}
