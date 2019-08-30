package files

import (
	"reflect"
	"time"

	"github.com/shopspring/decimal"
)

type SoDSecurityOverride struct {
	Symbol                   string           `sql:"type:text"`
	MaintenanceLong          *decimal.Decimal `gorm:"type:decimal"`
	MaintenanceShort         *decimal.Decimal `gorm:"type:decimal"`
	InitialLong              *decimal.Decimal `gorm:"type:decimal"`
	InitialShort             *decimal.Decimal `gorm:"type:decimal"`
	DayTradeRequirementLong  *decimal.Decimal `gorm:"type:decimal"`
	DayTradeRequirementShort *decimal.Decimal `gorm:"type:decimal"`
	UnderlierLessOOMPercent  *decimal.Decimal `gorm:"type:decimal"`
	UnderlierPercent         *decimal.Decimal `gorm:"type:decimal"`
	UncoveredOptionMin       *decimal.Decimal `gorm:"type:decimal"`
	Qualifer                 int
	Value                    string  `sql:"type:text"`
	DateModified             *string `sql:"type:date"`
}

type SecurityOverrideReport struct {
	overrides []SoDSecurityOverride
}

func (sor *SecurityOverrideReport) ExtCode() string {
	return "EXT902"
}

func (sor *SecurityOverrideReport) Delimiter() string {
	return ","
}

func (sor *SecurityOverrideReport) Header() bool {
	return false
}

func (sor *SecurityOverrideReport) Extension() string {
	return "csv"
}

func (sor *SecurityOverrideReport) Value() reflect.Value {
	return reflect.ValueOf(sor.overrides)
}

func (sor *SecurityOverrideReport) Append(v interface{}) {
	sor.overrides = append(sor.overrides, v.(SoDSecurityOverride))
}

func (sor *SecurityOverrideReport) Sync(asOf time.Time) (uint, uint) {
	// ovrs := make([]interface{}, len(sor.overrides))
	// for i := 0; i < len(ovrs); i++ {
	// 	ovrs[i] = &models.SoDSecurityOverrideModel{
	// 		Model:               gorm.Model{},
	// 		SoDSecurityOverride: sor.overrides[i],
	// 	}
	// }
	return 0, 0
}
