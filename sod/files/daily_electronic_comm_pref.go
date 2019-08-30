package files

import (
	"reflect"
	"time"
)

type SoDElectronicCommPrefExtract struct {
	AccoutNumber      string  `gorm:"type:varchar(13);index"`
	RegisteredRepCode string  `sql:"type:text"`
	StatementStatus   string  `sql:"type:text"`
	ConfirmStatus     string  `sql:"type:text"`
	ProspectusStatus  string  `sql:"type:text"`
	ProxyStatus       string  `sql:"type:text"`
	TaxStatus         string  `sql:"type:text"`
	ClosedIndicator   string  `sql:"type:text"`
	ClosedReason      string  `sql:"type:text"`
	EmailAddress      string  `sql:"type:text"`
	LastUpdated       *string `sql:"type:date"`
}

type ElectronicCommPrefReport struct {
	extracts []SoDElectronicCommPrefExtract
}

func (e *ElectronicCommPrefReport) ExtCode() string {
	return "EXT989"
}

func (e *ElectronicCommPrefReport) Delimiter() string {
	return ","
}

func (e *ElectronicCommPrefReport) Header() bool {
	return false
}

func (e *ElectronicCommPrefReport) Extension() string {
	return "csv"
}

func (e *ElectronicCommPrefReport) Value() reflect.Value {
	return reflect.ValueOf(e.extracts)
}

func (e *ElectronicCommPrefReport) Append(v interface{}) {
	e.extracts = append(e.extracts, v.(SoDElectronicCommPrefExtract))
}

func (e *ElectronicCommPrefReport) Sync(asOf time.Time) (uint, uint) {
	// extracts := make([]interface{}, len(e.extracts))
	// for i := 0; i < len(extracts); i++ {
	// 	extracts[i] = &models.SoDElectronicCommPrefExtractModel{
	// 		Model: gorm.Model{},
	// 		SoDElectronicCommPrefExtract: e.extracts[i],
	// 	}
	// }
	return 0, 0
}
