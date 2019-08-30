package files

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gobroker/service/op"
	"github.com/alpacahq/gobroker/utils"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/log"
	"github.com/shopspring/decimal"
)

type SoDBuySell string

const (
	Buy  SoDBuySell = "B"
	Sell SoDBuySell = "S"
)

func (bs *SoDBuySell) ToEnum() enum.Side {
	switch *bs {
	case Buy:
		return enum.Buy
	default:
		return enum.Sell
	}
}

type SoDTradeActivity struct {
	AccountNumber               string           `gorm:"type:varchar(13);index"`
	ProcessDate                 *string          `sql:"type:date"`
	Firm                        string           `sql:"type:text"`
	CorrespondentID             string           `sql:"type:text"`
	CorrespondentOfficeID       string           `sql:"type:text"`
	CorrespondentCode           string           `sql:"type:text"`
	OfficeCode                  string           `sql:"type:text"`
	RegisteredRepCode           string           `sql:"type:text"`
	AccType                     SoDAccountType   `sql:"type:text"`
	BuySellCode                 SoDBuySell       `sql:"type:text"`
	TradeDate                   *string          `sql:"type:date"`
	TradeNumber                 string           `sql:"type:text"`
	ExecutionTime               string           `sql:"type:text"`
	CUSIP                       string           `sql:"type:text"`
	Symbol                      string           `sql:"type:text"`
	Quantity                    *decimal.Decimal `gorm:"type:decimal"`
	Price                       *decimal.Decimal `gorm:"type:decimal"`
	MarketCode                  string           `sql:"type:text"`
	CapacityCode                string           `sql:"type:text"`
	CommissionGrossCalculated   *decimal.Decimal `gorm:"type:decimal"`
	CommissionGrossEntered      *decimal.Decimal `gorm:"type:decimal"`
	SettlementDate              *string          `sql:"type:date"`
	CurrencyCode                string           `sql:"type:text"`
	PrincipalAmount             *decimal.Decimal `gorm:"type:decimal"`
	NetAmount                   *decimal.Decimal `gorm:"type:decimal"`
	FeeSec                      *decimal.Decimal `gorm:"type:decimal"`
	FeeMisc                     *decimal.Decimal `gorm:"type:decimal"`
	Fee1                        *decimal.Decimal `gorm:"type:decimal"`
	Fee2                        *decimal.Decimal `gorm:"type:decimal"`
	Fee3                        *decimal.Decimal `gorm:"type:decimal"`
	Fee4                        *decimal.Decimal `gorm:"type:decimal"`
	Fee5                        *decimal.Decimal `gorm:"type:decimal"`
	EntryDate                   *string          `sql:"type:date"`
	ShortDescription            string           `csv:"skip" sql:"-"`
	TrailerCode                 string           `csv:"skip" sql:"-"`
	TradeInterest               string           `sql:"type:text"`
	ExecutingBrokerBack         string           `sql:"type:text"`
	SecurityTypeCode            SoDSecurityType  `sql:"type:text"`
	CommissionRRCategory        string           `sql:"type:text"`
	Reallowance                 string           `csv:"skip" sql:"-"`
	CommissionEntered           *decimal.Decimal `gorm:"type:decimal"`
	ShortName                   string           `sql:"type:text"`
	Factor                      *decimal.Decimal `gorm:"type:decimal"`
	CommissionNet               string           `csv:"skip" sql:"-"`
	Trailer                     string           `sql:"type:text"`
	ExecutingBrokerFront        string           `sql:"type:text"`
	FeeMF                       string           `csv:"skip" sql:"-"`
	ClearingSymbol              string           `csv:"skip" sql:"-"`
	Repo                        string           `csv:"skip" sql:"-"`
	Description1                string           `sql:"type:text"`
	SecuritySubType             string           `sql:"type:text"`
	InstructionsTradeLegendCode string           `csv:"skip" sql:"-"`
	Country                     string           `csv:"skip" sql:"-"`
	ISIN                        string           `csv:"skip" sql:"-"`
	LanguageID                  string           `csv:"skip" sql:"-"`
	InstructionsSpecial1        string           `csv:"skip" sql:"-"`
	InstructionsSpecial2        string           `sql:"type:text"`
	OriginalTradeNumber         string           `csv:"skip" sql:"-"`
	TradeLegendCode             string           `csv:"skip" sql:"-"`
	OptionSymbolRoot            string           `sql:"type:text"`
	DisplaySymbol               string           `sql:"type:text"`
	StrikePrice                 *decimal.Decimal `gorm:"type:decimal"`
	CallPut                     string           `sql:"type:text"`
	ExpirationDeliveryDate      *string          `sql:"type:date"`
	OptionContractDate          *string          `sql:"type:date"`
}

type TradeActivityReport struct {
	activities []SoDTradeActivity
}

func (tar *TradeActivityReport) ExtCode() string {
	return "EXT872"
}

func (tar *TradeActivityReport) Delimiter() string {
	return ","
}

func (tar *TradeActivityReport) Header() bool {
	// Apex give us trade activity report with header in UAT.
	if utils.Stg() {
		return true
	}
	return false
}

func (tar *TradeActivityReport) Extension() string {
	return "csv"
}

func (tar *TradeActivityReport) Value() reflect.Value {
	return reflect.ValueOf(tar.activities)
}

func (tar *TradeActivityReport) Append(v interface{}) {
	tar.activities = append(tar.activities, v.(SoDTradeActivity))
}

// Sync goes through the trades for the day and compares them to
// the executions stored in the DB. Any inconsistencies are stored
// in the batch_errors table.
func (tar *TradeActivityReport) Sync(asOf time.Time) (uint, uint) {
	errors := []models.BatchError{}
	tradeMap := make(map[string][]SoDTradeActivity)

	for _, activity := range tar.activities {
		if IsFirmAccount(activity.AccountNumber) || !activity.SecurityTypeCode.Supported() {
			continue
		}

		if trades, ok := tradeMap[activity.AccountNumber]; ok {
			trades = append(trades, activity)
			tradeMap[activity.AccountNumber] = trades
		} else {
			tradeMap[activity.AccountNumber] = []SoDTradeActivity{activity}
		}
	}

	for apexAcct, trades := range tradeMap {

		tx := db.DB().Begin()

		svc := account.Service().WithTx(tx)
		svc.SetForUpdate()

		_, err := svc.GetByApexAccount(apexAcct)
		if err != nil {
			errors = append(errors, tar.genError(asOf, trades, fmt.Errorf("account not found")))
			tx.Rollback()
			continue
		}

		// find the account
		// Warn : this might break the logic if braggart reporting delayed.
		executions, err := op.GetDayBraggartExecutions(tx, apexAcct, asOf)
		if err != nil {
			tx.Rollback()
			log.Panic("start of day database error", "file", tar.ExtCode(), "error", err)
		}

		if len(executions) == 0 {
			if utils.Prod() {
				errors = append(errors, tar.genError(asOf, trades, fmt.Errorf("executions not found")))
			}
			tx.Rollback()
			continue
		}

		// validate the trades
		pairs, err := tar.validate(trades, executions)
		if err != nil {
			errors = append(errors, tar.genError(asOf, trades, err))
			tx.Rollback()
			continue
		}

		for _, pair := range pairs {
			patch := models.Execution{
				FeeSec:  pair.report.FeeSec,
				FeeMisc: pair.report.FeeMisc,
				Fee1:    pair.report.Fee1,
				Fee2:    pair.report.Fee2,
				Fee3:    pair.report.Fee3,
				Fee4:    pair.report.Fee4,
				Fee5:    pair.report.Fee5,
			}
			if err := tx.Model(&pair.execution).Updates(patch).Error; err != nil {
				tx.Rollback()
				log.Panic("start of day database error", "file", tar.ExtCode(), "error", err)
			}
		}

		oIDs := getOrderIDs(pairs)
		var orders []models.Order
		if err := tx.Where("id IN (?)", oIDs).Preload("Executions").Find(&orders).Error; err != nil {
			log.Panic("start of day database error", "file", tar.ExtCode(), "error", err)
		}

		if len(oIDs) != len(orders) {
			errors = append(errors, tar.genError(asOf, trades, fmt.Errorf("execution order_ids and actual orders mismatch")))
			tx.Rollback()
			continue
		}

		for _, o := range orders {
			fee := decimal.Zero
			for _, e := range o.Executions {
				if e.HasFee() {
					fee = fee.Add(e.TotalFee())
				}
			}
			// select fee not to update executions (by defualt automatically UPDATE sql is called by gorm)
			if err := tx.Model(&o).Select("fee").Updates(models.Order{Fee: &fee}).Error; err != nil {
				tx.Rollback()
				log.Panic("start of day database error", "file", tar.ExtCode(), "error", err)
			}
		}

		if err := tx.Commit().Error; err != nil {
			log.Panic("start of day database error", "file", tar.ExtCode(), "error", err)
		}
	}

	StoreErrors(errors)

	return uint(len(tar.activities) - len(errors)), uint(len(errors))
}

func (tar *TradeActivityReport) genError(asOf time.Time, trades []SoDTradeActivity, err error) models.BatchError {
	log.Error("start of day error", "file", tar.ExtCode(), "error", err)
	buf, _ := json.Marshal(map[string]interface{}{
		"error":      err.Error(),
		"sod_trades": trades,
	})
	return models.BatchError{
		ProcessDate:             asOf.Format("2006-01-02"),
		FileCode:                tar.ExtCode(),
		PrimaryRecordIdentifier: trades[0].AccountNumber,
		Error:                   buf,
	}
}

type Pair struct {
	report    SoDTradeActivity
	execution models.Execution
}

func (tar *TradeActivityReport) validate(trades []SoDTradeActivity, executions []models.Execution) ([]Pair, error) {
	// trade count doesn't match, report an error
	if len(trades) != len(executions) {
		return nil, fmt.Errorf(
			"trade count mismatch [sod: %v|db: %v]",
			len(trades), len(executions))
	}

	pairs := []Pair{}

	for ti, trade := range trades {
		for i, exec := range executions {
			if strings.EqualFold(models.ApexFormat(exec.Symbol), trade.Symbol) &&
				exec.Side == trade.BuySellCode.ToEnum() &&
				exec.Qty.Mul(exec.Side.Coeff()).Equal(*trade.Quantity) &&
				exec.Price.Equal(*trade.Price) &&
				(exec.TransactionTime.In(calendar.NY).Format("1504") == trade.ExecutionTime) {
				pairs = append(pairs, Pair{report: trades[ti], execution: executions[i]})
				// matched the trade, let's remove it so we don't double match
				executions = append(executions[:i], executions[i+1:]...)
				break
			}
			// we couldn't match the trade, report an error
			if i == len(executions)-1 {
				return nil, fmt.Errorf("invalid trade: %v", trade)
			}
		}
	}

	return pairs, nil
}

func getOrderIDs(executions []Pair) []string {
	oIDs := map[string]struct{}{}
	for _, e := range executions {
		oIDs[e.execution.OrderID] = struct{}{}
	}
	keys := []string{}
	for key := range oIDs {
		keys = append(keys, key)
	}
	return keys
}
