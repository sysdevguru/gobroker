package files

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/alpacahq/gobroker/mailer"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gobroker/utils"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/log"
	"github.com/shopspring/decimal"
)

type SoDMarginCallType string

const (
	ConcentratedMaintenance SoDMarginCallType = "CM"
	DayTrading              SoDMarginCallType = "DT"
	EquityMaintenance       SoDMarginCallType = "EM"
	GoodFaithViolations     SoDMarginCallType = "GF"
	GoodFaithWarnings       SoDMarginCallType = "GW"
	JBOEquity               SoDMarginCallType = "JE"
	LeverageMaintenance     SoDMarginCallType = "LM"
	MoneyDue                SoDMarginCallType = "MD"
	RegulationMT            SoDMarginCallType = "MT"
	PortfolioEquity         SoDMarginCallType = "PE"
	PortfolioMargin         SoDMarginCallType = "PM"
	RequiredMaintenanceCall SoDMarginCallType = "RM"
	RegulationT             SoDMarginCallType = "RT"
	Type1Shorts             SoDMarginCallType = "S1"
)

var callTypeM = map[SoDMarginCallType]enum.MarginCallType{
	ConcentratedMaintenance: enum.ConcentratedMaintenance,
	DayTrading:              enum.DayTrading,
	EquityMaintenance:       enum.EquityMaintenance,
	GoodFaithViolations:     enum.GoodFaithViolations,
	GoodFaithWarnings:       enum.GoodFaithWarnings,
	JBOEquity:               enum.JBOEquity,
	LeverageMaintenance:     enum.LeverageMaintenance,
	MoneyDue:                enum.MoneyDue,
	RegulationMT:            enum.RegulationMT,
	PortfolioEquity:         enum.PortfolioEquity,
	PortfolioMargin:         enum.PortfolioMargin,
	RequiredMaintenanceCall: enum.RequiredMaintenanceCall,
	RegulationT:             enum.RegulationT,
	Type1Shorts:             enum.Type1Shorts,
}

func (mct SoDMarginCallType) ToModel() enum.MarginCallType {
	return callTypeM[mct]
}

type SoDMarginCall struct {
	CallID        int               // unused
	AccountNumber string            `gorm:"type:varchar(13);index"`
	AccountName   string            `sql:"type:text"`
	CallAmount    decimal.Decimal   `gorm:"type:decimal"`
	CallType      SoDMarginCallType `sql:"type:text"`
	TradeDate     string            `sql:"type:date"`
	DueDate       string            `sql:"type:date"`
	RegTDate      *string           `sql:"type:date"`
}

type MarginCallReport struct {
	calls []SoDMarginCall
}

func (mcr *MarginCallReport) ExtCode() string {
	return "EXT250"
}

func (mcr *MarginCallReport) Delimiter() string {
	return ","
}

func (mcr *MarginCallReport) Header() bool {
	return false
}

func (mcr *MarginCallReport) Extension() string {
	return "csv"
}

func (mcr *MarginCallReport) Value() reflect.Value {
	return reflect.ValueOf(mcr.calls)
}

func (mcr *MarginCallReport) Append(v interface{}) {
	mcr.calls = append(mcr.calls, v.(SoDMarginCall))
}

func (mcr *MarginCallReport) Sync(asOf time.Time) (uint, uint) {

	errors := []models.BatchError{}

	for _, call := range mcr.calls {
		acct := &models.Account{}

		if IsFirmAccount(call.AccountNumber) {
			continue
		}

		tx := db.Begin()

		srv := account.Service().WithTx(tx)

		acct, err := srv.GetByApexAccount(call.AccountNumber)

		if err != nil {
			tx.Rollback()
			if strings.Contains(err.Error(), "account not found") {
				if utils.Prod() {
					errors = append(errors, mcr.genError(asOf, call, fmt.Errorf("account not found")))
				}
				continue
			} else {
				log.Panic("start of day database error", "file", mcr.ExtCode(), "error", err)
			}
		}

		dueDate, err := time.Parse("01/02/2006", call.DueDate)
		if err != nil {
			tx.Rollback()
			errors = append(errors, mcr.genError(asOf, call, fmt.Errorf("failed to parse due_date (%v)", err)))
			continue
		}

		// store the margin call in the DB
		mc := models.MarginCall{
			AccountID:  acct.ID,
			CallAmount: call.CallAmount,
			CallType:   call.CallType.ToModel(),
			TradeDate:  call.TradeDate,
			DueDate:    call.DueDate,
		}

		if err := tx.FirstOrCreate(&mc).Error; err != nil {
			tx.Rollback()
			log.Panic("start of day database error", "file", mcr.ExtCode(), "error", err)
		}

		if mc.ShouldNotify(nil) {
			now := clock.Now()

			// notify the account owner
			switch call.CallType {
			// in the case of an EM call and the account has been marked PDT,
			case EquityMaintenance:
				// here we decided whether this is really a PDT margin call
				// or just a standard one.
				if acct.PatternDayTrader && acct.MarkedPatternDayTraderAt != nil {
					markedAt := acct.MarkedPatternDayTraderAt.In(calendar.NY)
					// if they were marked PDT in the past week, then this equity
					// maintenance call is PDT related, so we should send the PDT
					// email, and not the generic margin call email.
					if markedAt.After(now.Add(-7 * 24 * time.Hour)) {
						go mailer.SendPDTCall(
							*acct.ApexAccount,
							*acct.Owners[0].Details.GivenName,
							acct.Owners[0].Email,
							dueDate,
							call.CallAmount,
							nil,
						)
						mc.LastNotifiedAt = &now

					}
				}
				fallthrough
			default:
				if err := mailer.SendMarginCall(
					*acct.ApexAccount,
					*acct.Owners[0].Details.GivenName,
					acct.Owners[0].Email,
					dueDate,
					call.CallAmount,
					nil,
				); err != nil {
					errors = append(errors, mcr.genError(asOf, call, fmt.Errorf("mail delivery failure (%v)", err)))
				} else {
					mc.LastNotifiedAt = &now
				}
			}
		}

		// update last_notified_at timestamp
		if err := tx.Save(&mc).Error; err != nil {
			tx.Rollback()
			log.Panic("start of day database error", "file", mcr.ExtCode(), "error", err)
		}

		tx.Commit()
	}

	StoreErrors(errors)

	count := 0

	db.DB().Model(&models.MarginCall{}).Count(&count)

	return uint(count), uint(len(errors))
}

func (mcr *MarginCallReport) genError(asOf time.Time, sodCall SoDMarginCall, err error) models.BatchError {
	log.Error("start of day error", "file", mcr.ExtCode(), "error", err)
	buf, _ := json.Marshal(map[string]interface{}{
		"error":           err,
		"sod_margin_call": sodCall,
	})
	return models.BatchError{
		ProcessDate:               asOf.Format("2006-01-02"),
		FileCode:                  mcr.ExtCode(),
		PrimaryRecordIdentifier:   sodCall.AccountNumber,
		SecondaryRecordIdentifier: string(sodCall.CallType.ToModel()),
		Error:                     buf,
	}
}
