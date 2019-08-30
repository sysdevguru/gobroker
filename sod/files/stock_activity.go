package files

import (
	"reflect"
	"time"

	"github.com/shopspring/decimal"
)

// NOTE: only applies for ACATS stock transfers
type SoDStockActivity struct {
	AccountNumber         string         `gorm:"type:varchar(13);index"`
	CurrencyCode          string         `sql:"type:text"`
	AccType               SoDAccountType `sql:"type:text"`
	EntryDate             *string        `sql:"type:date"`
	Cusip                 string         `sql:"type:text"`
	SequenceNumber        int
	LocLocation           string           `csv:"skip" sql:"-"`
	LocMemo               string           `csv:"skip" sql:"-"`
	TradeDate             *string          `sql:"type:date"`
	TradeNumber           string           `sql:"type:text"`
	SettleDate            *string          `sql:"type:date"`
	TradeSettleBasis      string           `sql:"type:text"`
	Trailer               string           `sql:"type:text"`
	Quantity              *decimal.Decimal `gorm:"type:decimal"`
	SecurityTypeCode      string           `sql:"type:text"`
	CertificateNumber     string           `csv:"skip" sql:"-"`
	LocationFrom1         string           `csv:"skip" sql:"-"`
	LocationFrom2         string           `csv:"skip" sql:"-"`
	LocationTo1           string           `csv:"skip" sql:"-"`
	LocationTo2           string           `csv:"skip" sql:"-"`
	CertificateHeld       string           `csv:"skip" sql:"-"`
	EnteredDate           *string          `sql:"type:date"`
	SourceProgram         string           `sql:"type:text"`
	StatementIndicator    string           `csv:"skip" sql:"-"`
	ActivityIndicator     string           `csv:"skip" sql:"-"`
	CorrSeg               string           `csv:"skip" sql:"-"`
	Override              string           `csv:"skip" sql:"-"`
	PDOverride            string           `csv:"skip" sql:"-"`
	MLPAdjustment         string           `csv:"skip" sql:"-"`
	IRAIgnore             string           `csv:"skip" sql:"-"`
	MergeEntryCode        string           `csv:"skip" sql:"-"`
	HistoryEntryCode      string           `csv:"skip" sql:"-"`
	EntryType             string           `sql:"type:text"`
	TerminalID            string           `sql:"type:text"`
	UserID                string           `sql:"type:text"`
	SegIndicator          string           `sql:"type:text"`
	IssueDate             *string          `sql:"type:date"`
	CertificateShortDesc  string           `sql:"type:text"`
	SMAChangeAmount       *decimal.Decimal `gorm:"type:decimal"`
	SMAChangePrice        *decimal.Decimal `gorm:"type:decimal"`
	SMAChangeRate         decimal.Decimal  `csv:"skip" sql:"-"`
	ContraAccountNumber   string           `csv:"skip" sql:"-"`
	ContraCurrencyCode    string           `csv:"skip" sql:"-"`
	ContraAccountTypeCode string           `csv:"skip" sql:"-"`
	ReInvestmentAmount    *decimal.Decimal `gorm:"type:decimal"`
	ReInvestmentPrice     decimal.Decimal  `csv:"skip" sql:"-"`
	DTCNumberExp          string           `sql:"type:text"`
	DTCNumber             string           `csv:"skip" sql:"-"`
	SequenceCusipNumber   string           `sql:"type:text"`
	SequenceEntryDate     *string          `sql:"type:date"`
	ProcessDate           *string          `sql:"type:date"`
}

type StockActivityReport struct {
	activities []SoDStockActivity
}

func (sar *StockActivityReport) ExtCode() string {
	return "EXT870"
}

func (sar *StockActivityReport) Delimiter() string {
	return ","
}

func (sar *StockActivityReport) Header() bool {
	return false
}

func (sar *StockActivityReport) Extension() string {
	return "csv"
}

func (sar *StockActivityReport) Value() reflect.Value {
	return reflect.ValueOf(sar.activities)
}

func (sar *StockActivityReport) Append(v interface{}) {
	sar.activities = append(sar.activities, v.(SoDStockActivity))
}

func (sar *StockActivityReport) Sync(asOf time.Time) (uint, uint) {
	// acts := make([]interface{}, len(sar.activities))
	// for i := 0; i < len(acts); i++ {
	// 	acts[i] = &models.SoDStockActivityModel{
	// 		Model:            gorm.Model{},
	// 		SoDStockActivity: sar.activities[i],
	// 	}
	// }
	// return db.BatchCreate(acts)
	return 0, 0
}
