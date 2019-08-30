package models

import (
	"encoding/json"
)

// BatchError is used to store errors encountered during SoD
// batch processing. Each error is defined as unique by its
// composite key. The composite key is comprised of the processing
// date, the file code (EXT...), as well as the primary and
// secondary record identifiers which are different depending on
// the file code. For example, for EXT871 (positions), the key
// would take the form: 2018-01-03|EXT871|3AP00001|AAPL. Whereas
// for EXT235 (mandatory actions), the key would take the form:
// 2018-02-01|EXT235|UVXY|StockSplit. This will make the records
// relatively human readable, and idempotent between batches.
type BatchError struct {
	ProcessDate               string          `gorm:"primary_key" sql:"type:date NOT NULL"`
	FileCode                  string          `gorm:"primary_key" sql:"type:text NOT NULL"`
	PrimaryRecordIdentifier   string          `gorm:"primary_key" sql:"type:text NOT NULL"`
	SecondaryRecordIdentifier string          `gorm:"primary_key" sql:"type:text;default:''"`
	Error                     json.RawMessage `sql:"type:json"`
}

// BatchMetric is used to store metrics related to processing of
// the start of day files. It includes the duration that it took
// to parse each file, as well as the number of records, and
// number of errors, keyed on date and file extension code.
type BatchMetric struct {
	ProcessDate     string `json:"date" gorm:"primary_key" sql:"type:date NOT NULL"`
	FileCode        string `json:"code" gorm:"primary_key" sql:"type:text NOT NULL"`
	ProcessDuration int    `json:"duration" sql:"type:integer NOT NULL"`
	RecordCount     uint   `json:"successes" gorm:"not null"`
	ErrorCount      uint   `json:"failures" gorm:"not null"`
}
