package files

import (
	"reflect"
	"time"

	"github.com/shopspring/decimal"
)

type SoDBuyingPowerDetail struct {
	AccountNumber                   string `gorm:"type:varchar(13);index"`
	Firm                            string `sql:"type:text"`
	OfficeCode                      string `sql:"type:text"`
	StrategyID                      int
	StrategySequence                int
	AccType                         SoDAccountType   `sql:"type:text"`
	CUSIP                           string           `sql:"type:text"`
	TradeQuantity                   *decimal.Decimal `gorm:"type:decimal"`
	Symbol                          string           `sql:"type:text"`
	Description                     string           `sql:"type:text"`
	ClosingPrice                    *decimal.Decimal `gorm:"type:decimal"`
	MarketValue                     *decimal.Decimal `gorm:"type:decimal"`
	SecurityTypeCode                string           `sql:"type:text"`
	UnderlyingSymbol                string           `sql:"type:text"`
	StrikePrice                     *decimal.Decimal `gorm:"type:decimal"`
	ISIN                            string           `csv:"skip" sql:"-"`
	Change                          decimal.Decimal  `csv:"skip" sql:"-"`
	PositionType                    string           `sql:"type:text"`
	ProcessDate                     *string          `sql:"type:date"`
	IsSelling                       string           `sql:"type:text"`
	MaintenanceRequirement          *decimal.Decimal `gorm:"type:decimal"`
	ConcentrationRequirement        *decimal.Decimal `gorm:"type:decimal"`
	OptionStrategy                  string           `sql:"type:text"`
	OptionLeg                       string           `sql:"type:text"`
	StrategyMaintenanceRequirement  *decimal.Decimal `gorm:"type:decimal"`
	StrategyConcetrationRequirement *decimal.Decimal `gorm:"type:decimal"`
}

type BuyingPowerDetailReport struct {
	details []SoDBuyingPowerDetail
}

func (bpd *BuyingPowerDetailReport) ExtCode() string {
	return "EXT982"
}

func (bpd *BuyingPowerDetailReport) Delimiter() string {
	return ","
}

func (bpd *BuyingPowerDetailReport) Header() bool {
	return true
}

func (bpd *BuyingPowerDetailReport) Extension() string {
	return "csv"
}

func (bpd *BuyingPowerDetailReport) Value() reflect.Value {
	return reflect.ValueOf(bpd.details)
}

func (bpd *BuyingPowerDetailReport) Append(v interface{}) {
	bpd.details = append(bpd.details, v.(SoDBuyingPowerDetail))
}

func (bpd *BuyingPowerDetailReport) Sync(asOf time.Time) (uint, uint) {
	// details := make([]interface{}, len(bpd.details))
	// for i := 0; i < len(details); i++ {
	// 	details[i] = &models.SoDBuyingPowerDetailModel{
	// 		Model:                gorm.Model{},
	// 		SoDBuyingPowerDetail: bpd.details[i],
	// 	}
	// }
	// return db.BatchCreate(details)
	return 0, 0
}
