package files

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/jinzhu/gorm"

	"github.com/alpacahq/apex"
	"github.com/alpacahq/gobroker/mailer"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gobroker/utils"
	"github.com/alpacahq/gobroker/utils/date"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/log"
	"github.com/alpacahq/gopaca/set"
	"github.com/shopspring/decimal"
)

type SoDCashActivity struct {
	AccountNumber           string           `gorm:"type:varchar(13);index"`
	AccType                 SoDAccountType   `sql:"type:text"`
	Amount                  *decimal.Decimal `gorm:"type:decimal"`
	Description             string           `sql:"type:text"`
	CurrencyCode            string           `sql:"type:text"`
	ProcessDate             *string          `sql:"type:date"`
	BatchCode               string           `sql:"type:text"`
	Cusip                   string           `sql:"type:text"`
	EntryDate               *string          `sql:"type:date"`
	SourceProgram           string           `sql:"type:text"`
	UserID                  string           `sql:"type:text"`
	ActivityIndicator       string           `sql:"type:text"`
	OfficeCode              string           `sql:"type:text"`
	ACATSControlNumber      string           `sql:"type:text"`
	ActionCode              string           `csv:"skip" sql:"-"`
	CheckNumber             string           `sql:"type:text"`
	ContraAccountNumber     string           `csv:"skip" sql:"-"`
	ContraAccountTypeCode   string           `csv:"skip" sql:"-"`
	ContraCurrencyCode      string           `csv:"skip" sql:"-"`
	CorrWindowCode          string           `csv:"skip" sql:"-"`
	CurrentPaydownFactor    decimal.Decimal  `csv:"skip" sql:"-"`
	DivTaxTypeCode          string           `sql:"type:text"`
	DTCNumber               string           `csv:"skip" sql:"-"`
	DTCNumberExp            string           `csv:"skip" sql:"-"`
	EffectiveDate           *string          `sql:"type:date"`
	EnteredBy               string           `sql:"type:text"`
	EntryTypeCode           string           `sql:"type:text"`
	FeeIndicator            string           `csv:"skip" sql:"-"`
	Firm                    string           `sql:"type:text"`
	ForeignCode             string           `sql:"type:text"`
	FundsUserCode           string           `sql:"type:text"`
	GLPostStatusCode        string           `sql:"type:text"`
	HistoryEntryCode        string           `sql:"type:text"`
	InterestEffectiveDate   *string          `sql:"type:date"`
	LastPaydownFactor       decimal.Decimal  `csv:"skip" sql:"-"`
	MDHIndicator            string           `csv:"skip" sql:"-"`
	MergeEntryCode          string           `csv:"skip" sql:"-"`
	MAgentIndicator         string           `csv:"skip" sql:"-"`
	MoneyMarketCode         string           `sql:"type:text"`
	MutualFundPostIndicator string           `csv:"skip" sql:"-"`
	OriginalQuantity        *decimal.Decimal `gorm:"type:decimal"`
	OverrideIndicator       string           `sql:"type:text"`
	PasMergeEntryCode       string           `sql:"type:text"`
	PayTypeCode             string           `sql:"type:text"`
	Price                   *decimal.Decimal `gorm:"type:decimal"`
	RecTypeCode             string           `sql:"type:text"`
	RegisteredRepCode1      string           `sql:"type:text"`
	RRCategoryCode          int              `csv:"skip" sql:"-"`
	SequenceNumber          int
	SMAChangeAmount         decimal.Decimal `csv:"skip" sql:"-"`
	StatementIndicator      string          `csv:"skip" sql:"-"`
	StatusCode              string          `csv:"skip" sql:"-"`
	TaxOverrideCode         string          `csv:"skip" sql:"-"`
	TaxYear                 string          `csv:"skip" sql:"-"`
	TerminalID              string          `sql:"type:text"`
	ThirdPartyCode          string          `csv:"skip" sql:"-"`
	TradeDate               *string         `sql:"type:date"`
	TradeNumber             string          `sql:"type:text"`
	UserEntryDate           *string         `sql:"type:date"`
	WireFundsCode           string          `csv:"skip" sql:"-"`
	WitholdTaxIndicator     string          `sql:"type:text"`
	WitholdTaxTypeCode      string          `sql:"type:text"`
	CorrespondentOfficeID   int
	CorrespondentID         int
	RegisteredRepCode2      string `sql:"type:text"`
}

func (s *SoDCashActivity) IsACH() bool {
	switch s.BatchCode {
	case "9C":
		fallthrough
	case "9D":
		return true
	default:
		return false
	}
}

func (s *SoDCashActivity) IsWire() bool {
	return s.BatchCode == "FW"
}

type CashActivityReport struct {
	activities []SoDCashActivity
}

func (car *CashActivityReport) ExtCode() string {
	return "EXT869"
}

func (car *CashActivityReport) Delimiter() string {
	return ","
}

func (car *CashActivityReport) Header() bool {
	return false
}

func (car *CashActivityReport) Extension() string {
	return "csv"
}

func (car *CashActivityReport) Value() reflect.Value {
	return reflect.ValueOf(car.activities)
}

func (car *CashActivityReport) Append(v interface{}) {
	car.activities = append(car.activities, v.(SoDCashActivity))
}

type CashActivityReportProcessor struct {
	asof                          date.Date
	transfers                     map[string][]SoDCashActivity
	dividends                     map[string][]SoDCashActivity
	accounts                      []string
	errors                        []models.BatchError
	iterIndex                     int
	ExtCode                       string
	sendMoneyTransferNotification func(
		acct, givenName, email string,
		transferredOn time.Time,
		direction apex.TransferDirection,
		transferAmount decimal.Decimal,
		deliverAt *time.Time) error
}

func NewCashActivityReportProcessor(asof date.Date, activities []SoDCashActivity) *CashActivityReportProcessor {
	transfers := make(map[string][]SoDCashActivity)
	dividends := make(map[string][]SoDCashActivity)

	accset := set.New()

	for i := range activities {
		activity := activities[i]

		if IsFirmAccount(activity.AccountNumber) {
			continue
		}

		log.Debug("check activity", "apexAccount", activity.AccountNumber)

		switch activity.BatchCode {
		// wire
		case "FW":
			fallthrough
		// ach
		case "9C":
			fallthrough
		case "9D":
			if items, ok := transfers[activity.AccountNumber]; ok {
				transfers[activity.AccountNumber] = append(items, activities[i])
			} else {
				transfers[activity.AccountNumber] = []SoDCashActivity{activities[i]}
			}
			accset.Add(activity.AccountNumber)
			continue
		case "$+":
			switch activity.EnteredBy {
			case "CIL":
				// Payments made to investors who received fractional shares
				// as a consequence of stock splits etc.
			case "DIV":
				// dividents
				if items, ok := dividends[activity.AccountNumber]; ok {
					dividends[activity.AccountNumber] = append(items, activities[i])
				} else {
					dividends[activity.AccountNumber] = []SoDCashActivity{activities[i]}
				}
				accset.Add(activity.AccountNumber)
				continue
			case "INT":
				// interests
			case "RLY":
				// loyarity payments
			default:
				// Only 4 as doc says
				panic("unexpected EntryCode")
			}
		}

		log.Warn("observed unprocessing cash activity", "file", "EXT869")
	}

	return &CashActivityReportProcessor{
		asof:                          asof,
		dividends:                     dividends,
		transfers:                     transfers,
		accounts:                      accset.List(),
		errors:                        []models.BatchError{},
		iterIndex:                     -1,
		ExtCode:                       "EXT869",
		sendMoneyTransferNotification: mailer.SendMoneyTransferNotification,
	}
}

func (p *CashActivityReportProcessor) Errors() []models.BatchError {
	return p.errors
}

func (p *CashActivityReportProcessor) NoEmail() *CashActivityReportProcessor {
	p.sendMoneyTransferNotification = func(
		acct, givenName, email string,
		transferredOn time.Time,
		direction apex.TransferDirection,
		transferAmount decimal.Decimal,
		deliverAt *time.Time) error {
		log.Debug("send money transfer notification", "acct", acct)
		return nil
	}
	return p
}

func (p *CashActivityReportProcessor) Next() bool {
	nextIdx := p.iterIndex + 1
	if nextIdx < len(p.accounts) {
		p.iterIndex = nextIdx
		return true
	}
	return false
}

func (p *CashActivityReportProcessor) Values() (apexAccount string, transfers []SoDCashActivity, dividends []SoDCashActivity) {
	apexAccount = p.accounts[p.iterIndex]

	if items, ok := p.transfers[apexAccount]; ok {
		transfers = items
	} else {
		transfers = []SoDCashActivity{}
	}

	if items, ok := p.dividends[apexAccount]; ok {
		dividends = items
	} else {
		dividends = []SoDCashActivity{}
	}

	return apexAccount, transfers, dividends
}

func (p *CashActivityReportProcessor) AddError(apexAccount string, keysAndValues ...interface{}) {

	log.Error(
		"start of day error",
		append([]interface{}{"file", p.ExtCode}, keysAndValues...)...,
	)

	m := map[string]interface{}{}

	for i := range keysAndValues {
		if i%2 == 0 {
			continue
		}
		key := keysAndValues[i-1].(string)
		value := keysAndValues[i]
		m[key] = fmt.Sprintf("%v", value)
	}
	m["apexAccount"] = apexAccount

	buf, _ := json.Marshal(m)

	err := models.BatchError{
		ProcessDate:             p.asof.String(),
		FileCode:                p.ExtCode,
		PrimaryRecordIdentifier: apexAccount,
		Error:                   buf,
	}

	p.errors = append(p.errors, err)
}

func (p *CashActivityReportProcessor) Run() {
	for p.Next() {
		apexAcct, transfers, dividends := p.Values()

		log.Debug("try cash activity process",
			"acct", apexAcct,
			"transfers", len(transfers),
			"dividends", len(dividends))

		tx := db.Begin()

		// wires
		{
			// aggregate wires for idempotency
			wires := []models.Transfer{}

			q := tx.
				Where("type = ? AND created_at > ?",
					enum.Wire, p.asof.String()).
				Find(&wires)

			if q.Error != nil {
				log.Panic("start of day database error", "file", p.ExtCode, "error", q.Error)
			}

			if len(wires) > 0 {
				// we have wires that have already been processed
				// so let's find the ones that we have already
				// handled and omit them from this op
				for i, transfer := range transfers {
					if transfer.IsWire() {
						for _, wire := range wires {
							if wire.Amount.Equal(transfer.Amount.Abs()) && ((transfer.Amount.Sign() == 0 && wire.Direction == apex.Outgoing) ||
								(transfer.Amount.Sign() > 0 && wire.Direction == apex.Incoming)) {
								// found it, remove it
								transfers = append(transfers[:i], transfers[i+1:]...)
								break
							}
						}
					}
				}
			}

			// create the remaining wires that aren't accounted for
			// and remove them from the list as well so they are not
			// handled in the following ACH logic
			srv := account.Service().WithTx(tx)

			for i, t := range transfers {
				if t.IsWire() {
					acct, err := srv.GetByApexAccount(t.AccountNumber)
					if err != nil {
						if utils.Prod() {
							tx.Rollback()
							log.Panic("account not found", "account", t.AccountNumber)
						}
						continue
					}

					b := true
					date := p.asof.String()

					transfer := &models.Transfer{
						Type:             enum.Wire,
						Status:           enum.TransferComplete,
						BalanceValidated: &b,
						BatchProcessedAt: &date,
						AccountID:        acct.ID,
					}

					if t.Amount.Sign() > 0 {
						transfer.Direction = apex.Incoming
						transfer.Amount = *t.Amount
					} else {
						transfer.Direction = apex.Outgoing
						transfer.Amount = t.Amount.Mul(decimal.New(-1, 0))
					}

					if err := tx.Save(transfer).Error; err != nil {
						log.Panic("start of day database error", "file", p.ExtCode, "error", err)
					}

					// created it, remove it from the list
					transfers = append(transfers[:i], transfers[i+1:]...)
				}
			}
		}

		// achs
		{
			var acct models.Account

			if err := tx.
				Where("apex_account = ?", apexAcct).
				Preload("Owners").
				Preload("Owners.Details", "replaced_by IS NULL").
				Preload("Transfers", func(db *gorm.DB) *gorm.DB {
					// get not batch processed one or processed one on that as of date to be idempotent.
					return db.
						Where("created_at < ? AND status = ?",
							p.asof.AddDays(1).String(), enum.TransferComplete).
						Where("batch_processed_at IS NULL OR batch_processed_at = ?", p.asof.String()).
						Order("created_at asc")
				}).Find(&acct).Error; err != nil {

				switch {
				case gorm.IsRecordNotFoundError(err):
					tx.Rollback()
					if utils.Prod() {
						tx.Rollback()
						log.Panic("account not found", "account", apexAcct)
					}
					continue
				default:
					log.Panic("start of day database error", "file", p.ExtCode, "error", err)
				}
			}

			log.Debug("transfers in database",
				"acct", apexAcct,
				"transfers", len(acct.Transfers))

			oktransfers := []models.Transfer{}

			for ti, transfer := range acct.Transfers {
				var amt decimal.Decimal
				if transfer.Direction == apex.Incoming {
					amt = transfer.Amount
				} else {
					amt = transfer.Amount.Mul(decimal.New(-1, 0))
				}

				var sodtransfer *SoDCashActivity
				for i := range transfers {
					// minus = equity, plus = dead
					sodamt := transfers[i].Amount.Mul(decimal.New(-1, 0))
					if amt.Equal(sodamt) {
						sodtransfer = &transfers[i]
						transfers = append(transfers[:i], transfers[i+1:]...)
						break
					}
				}
				if sodtransfer == nil {
					p.AddError(
						apexAcct,
						"error", fmt.Errorf("could not find transfer in SoD file"),
						"db_amount", transfer.Amount,
					)
					continue
				}

				// already processed
				if transfer.BatchProcessedAt != nil {
					log.Debug("confirmed already processed",
						"acct", apexAcct,
						"transferID", transfer.ID,
						"amount", transfer.Amount)
					continue
				}

				if err := tx.Model(&transfer).Select("batch_processed_at").Update("batch_processed_at", p.asof.String()).Error; err != nil {
					tx.Rollback()
					log.Panic("start of day database error", "file", p.ExtCode, "error", err)
				}

				oktransfers = append(oktransfers, acct.Transfers[ti])
				log.Debug("done transfer",
					"acct", apexAcct,
					"transferID", transfer.ID,
					"amount", transfer.Amount)
			}

			if len(transfers) != 0 {
				p.AddError(
					apexAcct,
					"error", fmt.Errorf("activity in sod file is not matched with transfers in db"),
					"transfers", len(transfers),
				)
			}

			for _, activity := range dividends {
				var div models.Dividend
				if err := tx.Where("account_id = ? AND cusip = ? AND pay_date = ?", acct.ID, activity.Cusip, activity.ProcessDate).
					Find(&div).Error; err != nil {
					switch {
					case gorm.IsRecordNotFoundError(err):
						p.AddError(
							apexAcct,
							"error", fmt.Errorf("dividend not found in our db"),
							"cusip", activity.Cusip,
							"pay_date", activity.ProcessDate,
						)
						continue
					default:
						tx.Rollback()
						log.Panic("start of day database error", "file", p.ExtCode, "error", err)
					}
				}

				// already marked
				if div.PayedAt != nil {
					continue
				}

				// confirm amount matches
				activityAmount := activity.Amount.Mul(decimal.New(-1, 0))
				if !div.DividendInterest.Round(2).Equal(activityAmount) {
					p.AddError(
						apexAcct,
						"error", fmt.Errorf("mismatch dividend amount"),
						"sod_amount", activityAmount,
						"db_amount", div.DividendInterest,
						"cusip", activity.Cusip,
					)
					continue
				}

				// update state.
				now := clock.Now()
				if err := tx.Model(div).Select("payed_at").Updates(models.Dividend{PayedAt: &now}).Error; err != nil {
					tx.Rollback()
					log.Panic("start of day database error", "file", p.ExtCode, "error", err)
				}
			}

			// Everything looks good, then send email to user. there might issue sending email multiple times
			// or failed to send them, but leave them as it for now. This need to be handled w/ redundant job
			// queue system later.
			for _, transfer := range oktransfers {
				go p.sendMoneyTransferNotification(
					*acct.ApexAccount,
					*acct.PrimaryOwner().Details.GivenName,
					acct.PrimaryOwner().Email,
					transfer.CreatedAt,
					transfer.Direction,
					transfer.Amount,
					nil)
			}

			if err := tx.Commit().Error; err != nil {
				log.Panic("start of day database error", "file", p.ExtCode, "error", err)
			}
		}
	}

	StoreErrors(p.errors)
}

// Sync goes through the cash activities for the day, aggregates
// them on an account basis, and compares the net change to the
// aggregated transfers stored in the database. Any inconsistencies
// are stored in the batch_errors table.
func (car *CashActivityReport) Sync(asOf time.Time) (uint, uint) {

	asOfDate := date.DateOf(asOf)

	props := NewCashActivityReportProcessor(asOfDate, car.activities)
	props.Run()

	nerrors := len(props.Errors())

	return uint(len(car.activities) - nerrors), uint(nerrors)
}

// SyncForBackfill do Sync with out sending out emails.
func (car *CashActivityReport) SyncForBackfill(asOf time.Time) (uint, uint) {
	asOfDate := date.DateOf(asOf)

	props := NewCashActivityReportProcessor(asOfDate, car.activities).NoEmail()
	props.Run()

	nerrors := len(props.Errors())

	return uint(len(car.activities) - nerrors), uint(nerrors)
}
