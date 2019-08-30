package files

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gobroker/utils"
	"github.com/alpacahq/gobroker/utils/date"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/log"
	"github.com/shopspring/decimal"
)

type SoDBuyingPowerSummary struct {
	OvernightBuyingPowerID               int
	AccountNumber                        string           `gorm:"type:varchar(13);index"`
	Firm                                 string           `sql:"type:text"`
	OfficeCode                           string           `sql:"type:text"`
	CorrespondentCode                    string           `sql:"type:text"`
	ProcessDate                          *string          `sql:"type:date"`
	CurrencyCode                         string           `sql:"type:text"`
	TotalEquity                          *decimal.Decimal `gorm:"type:decimal"`
	MarginEquity                         *decimal.Decimal `gorm:"type:decimal"`
	MarginRequirement                    *decimal.Decimal `gorm:"type:decimal"`
	MarginExcessEquity                   *decimal.Decimal `gorm:"type:decimal"`
	CashEquity                           *decimal.Decimal `gorm:"type:decimal"`
	CashRequirement                      *decimal.Decimal `gorm:"type:decimal"`
	CashExcessEquity                     *decimal.Decimal `gorm:"type:decimal"`
	MarginRequirementWithConcentration   *decimal.Decimal `gorm:"type:decimal"`
	MarginExcessEquityWithConcentration  *decimal.Decimal `gorm:"type:decimal"`
	OvernightBuyingPowerCalculated       *decimal.Decimal `gorm:"type:decimal"`
	OvernightBuyingPowerIssued           *decimal.Decimal `gorm:"type:decimal"`
	DayTradeBuyingPowerIssued            *decimal.Decimal `gorm:"type:decimal"`
	RegTBuyingPowerCalculated            *decimal.Decimal `gorm:"type:decimal"`
	RegTBuyingPowerIssued                *decimal.Decimal `gorm:"type:decimal"`
	OvernightFactorCalculated            *decimal.Decimal `gorm:"type:decimal"`
	OvernightFactorIssued                *decimal.Decimal `gorm:"type:decimal"`
	DayTradeFactorCalculated             *decimal.Decimal `gorm:"type:decimal"`
	DayTradeFactorIssued                 *decimal.Decimal `gorm:"type:decimal"`
	MarginEquityPercent                  *decimal.Decimal `gorm:"type:decimal"`
	PositionMarketValue                  *decimal.Decimal `gorm:"type:decimal"`
	LongEquityMarketValue                *decimal.Decimal `gorm:"type:decimal"`
	ShortEquityMarketValue               *decimal.Decimal `gorm:"type:decimal"`
	LongOptionMarketValue                *decimal.Decimal `gorm:"type:decimal"`
	ShortOptionMarketValue               *decimal.Decimal `gorm:"type:decimal"`
	TotalTradeBalance                    *decimal.Decimal `gorm:"type:decimal"`
	TotalSettleBalance                   *decimal.Decimal `gorm:"type:decimal"`
	CashTradeBalance                     *decimal.Decimal `gorm:"type:decimal"`
	MarginTradeBalance                   *decimal.Decimal `gorm:"type:decimal"`
	ShortTradeBalance                    *decimal.Decimal `gorm:"type:decimal"`
	MoneyMarketTradeBalance              *decimal.Decimal `gorm:"type:decimal"`
	CashSettleBalance                    *decimal.Decimal `gorm:"type:decimal"`
	MarginSettleBalance                  *decimal.Decimal `gorm:"type:decimal"`
	ShortSettleBalance                   *decimal.Decimal `gorm:"type:decimal"`
	MoneyMarketSettleBalance             *decimal.Decimal `gorm:"type:decimal"`
	FreeCash                             *decimal.Decimal `gorm:"type:decimal"`
	SMA                                  *decimal.Decimal `gorm:"type:decimal"`
	AvailableToWithdraw                  *decimal.Decimal `gorm:"type:decimal"`
	FutureBalance                        *decimal.Decimal `gorm:"type:decimal"`
	FutureEquity                         *decimal.Decimal `gorm:"type:decimal"`
	FutureRequirement                    *decimal.Decimal `gorm:"type:decimal"`
	OptionsRequirement                   *decimal.Decimal `gorm:"type:decimal"`
	NonOptionsRequirement                *decimal.Decimal `gorm:"type:decimal"`
	LastUpdate                           string           `gorm:"type:date"`
	NonOptionsRequirementNotConcentrated *decimal.Decimal `gorm:"type:decimal"`
	TypeIUnavailableCashProceeds         *decimal.Decimal `gorm:"type:decimal"`
	TypeIIUnavailableCashProceeds        *decimal.Decimal `gorm:"type:decimal"`
	NetBalance                           *decimal.Decimal `gorm:"type:decimal"`
	SMACommitted                         *decimal.Decimal `gorm:"type:decimal"`
	HighWaterMark                        *decimal.Decimal `gorm:"type:decimal"`
}

type BuyingPowerSummaryReport struct {
	summaries []SoDBuyingPowerSummary
}

func (bps *BuyingPowerSummaryReport) ExtCode() string {
	return "EXT981"
}

func (bps *BuyingPowerSummaryReport) Delimiter() string {
	return ","
}

func (bps *BuyingPowerSummaryReport) Header() bool {
	return true
}

func (bps *BuyingPowerSummaryReport) Extension() string {
	return "csv"
}

func (bps *BuyingPowerSummaryReport) Value() reflect.Value {
	return reflect.ValueOf(bps.summaries)
}

func (bps *BuyingPowerSummaryReport) Append(v interface{}) {
	bps.summaries = append(bps.summaries, v.(SoDBuyingPowerSummary))
}

// Sync goes through the buying power summaries, and updates the
// account table's cash and cash_withdrawable accordingly. Any
// errors are stored to the batch_errors table.
func (bps *BuyingPowerSummaryReport) Sync(asOf time.Time) (uint, uint) {
	errors := []models.BatchError{}

	for _, summary := range bps.summaries {

		if IsFirmAccount(summary.AccountNumber) {
			continue
		}

		acct := &models.Account{}

		// find the account
		q := db.DB().Where("apex_account = ?", summary.AccountNumber).Find(&acct)

		if q.RecordNotFound() {
			if utils.Prod() {
				errors = append(errors, bps.genError(asOf, summary, nil, fmt.Errorf("account not found")))
			}
			continue
		}

		if q.Error != nil {
			log.Panic("start of day database error", "file", bps.ExtCode(), "error", q.Error)
		}

		// update the buying power information
		patches := map[string]interface{}{
			"cash": *summary.NetBalance,
			// since we are only allowing 1x margin for now,
			// cash withdrawable should not exceed the net
			// account balance
			"cash_withdrawable": decimal.Min(
				*summary.NetBalance,
				*summary.AvailableToWithdraw),
		}

		tx := db.RepeatableRead()
		srv := account.Service().WithTx(tx)

		_, err := srv.PatchInternal(acct.IDAsUUID(), patches)
		if err != nil {
			tx.Rollback()
			errors = append(errors, bps.genError(asOf, summary, acct, err))
			continue
		}

		var accCash models.Cash
		patch := models.Cash{
			AccountID: acct.ID,
			Date:      date.DateOf(asOf),
		}

		if err := tx.FirstOrCreate(&accCash, patch).Error; err != nil {
			tx.Rollback()
			log.Panic("start of day database error", "file", bps.ExtCode(), "error", err)
		}

		accCash.Value = *summary.NetBalance

		if err := tx.Save(&accCash).Error; err != nil {
			tx.Rollback()
			log.Panic("start of day database error", "file", bps.ExtCode(), "error", err)
		}

		if err := tx.Commit().Error; err != nil {
			tx.Rollback()
			log.Panic("start of day database error", "file", bps.ExtCode(), "error", err)
		}
	}

	StoreErrors(errors)

	return uint(len(bps.summaries) - len(errors)), uint(len(errors))
}

func (bps *BuyingPowerSummaryReport) genError(asOf time.Time, summary SoDBuyingPowerSummary, acct *models.Account, err error) models.BatchError {
	log.Error("start of day error", "file", bps.ExtCode(), "error", err)
	buf, _ := json.Marshal(map[string]interface{}{
		"error":            err.Error(),
		"sod_buying_power": summary,
		"account":          *acct,
	})
	return models.BatchError{
		ProcessDate:             asOf.Format("2006-01-02"),
		FileCode:                bps.ExtCode(),
		PrimaryRecordIdentifier: summary.AccountNumber,
		Error:                   buf,
	}
}
