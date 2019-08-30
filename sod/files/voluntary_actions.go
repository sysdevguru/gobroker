package files

import (
	"reflect"
	"time"

	"github.com/alpacahq/gobroker/models/enum"
	"github.com/shopspring/decimal"
)

type SoDVoluntaryAction struct {
	AccountNumber      string                  `gorm:"type:varchar(13);index"`
	Firm               string                  `sql:"type:text"`
	AccountName        string                  `sql:"type:text"`
	TotalShareQuantity decimal.Decimal         `gorm:"type:decimal"`
	OfficeName         string                  `sql:"type:text"`
	OfficeCode         string                  `sql:"type:text"`
	Cusip              string                  `sql:"type:text"`
	Symbol             string                  `sql:"type:text"`
	ShortDescription   string                  `sql:"type:text"`
	ISIN               string                  `sql:"type:text"`
	Action             enum.SoDCorporateAction `sql:"type:text"`
	CountryCode        string                  `sql:"type:text"`
	RecordDate         *string                 `sql:"type:date"`
	ExpirationDate     *string                 `sql:"type:date"`
	ReorgCutOffDate    *string                 `sql:"type:date"`
	RedemptionDate     *string                 `sql:"type:date"`
	ProcessDate        *string                 `sql:"type:date"`
	LastChangeDate     *string                 `sql:"type:date"`
	ActionMessage      string                  `sql:"type:text"`
}

type VoluntaryActionReport struct {
	actions []SoDVoluntaryAction
}

func (va *VoluntaryActionReport) ExtCode() string {
	return "EXT236"
}

func (va *VoluntaryActionReport) Delimiter() string {
	return "|"
}

func (va *VoluntaryActionReport) Header() bool {
	return false
}

func (va *VoluntaryActionReport) Extension() string {
	return "txt"
}

func (va *VoluntaryActionReport) Value() reflect.Value {
	return reflect.ValueOf(va.actions)
}

func (va *VoluntaryActionReport) Append(v interface{}) {
	va.actions = append(va.actions, v.(SoDVoluntaryAction))
}

func (va *VoluntaryActionReport) Sync(asOf time.Time) (uint, uint) {
	// acts := make([]interface{}, len(va.actions))
	// for i := 0; i < len(acts); i++ {
	// 	acts[i] = &models.SoDVoluntaryActionModel{
	// 		Model:              gorm.Model{},
	// 		SoDVoluntaryAction: va.actions[i],
	// 	}
	// }
	// return db.BatchCreate(acts)
	return 0, 0
}
