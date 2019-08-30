package files

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/utils"
	"github.com/alpacahq/gobroker/workers/account/form"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/log"
)

type SoDAccount struct {
	AccountNumber          string `gorm:"type:varchar(13);index"`
	RecordTypeCode         string `csv:"skip" sql:"-"`
	RecordTypeSubTypeCode  string `csv:"skip" sql:"-"`
	RRIndexCode            string `csv:"skip" sql:"-"`
	OfficeCode             string `sql:"type:text"`
	RegisteredRepCode      string `sql:"type:text"`
	FederalIDIndicator     string `csv:"skip" sql:"-"`
	TaxIDNumber            string `sql:"type:text"`
	NameIndex              string `csv:"skip" sql:"-"`
	ShortName              string `sql:"type:text"`
	PartyIndex             string `csv:"skip" sql:"-"`
	RelatedParty           string `sql:"type:text"`
	AccountName            string `sql:"type:text"`
	AddressLine1           string `sql:"type:text"`
	AddressLine2           string `sql:"type:text"`
	AddressLine3           string `sql:"type:text"`
	AddressLine4           string `sql:"type:text"`
	City                   string `sql:"type:text"`
	State                  string `sql:"type:text"`
	ZipCode                string `sql:"type:text"`
	IRSControl             string `sql:"type:text"`
	MultiRRS               string `csv:"skip" sql:"-"`
	KND                    string `csv:"skip" sql:"-"`
	MapAccount             string `csv:"skip" sql:"-"`
	Fee                    string `csv:"skip" sql:"-"`
	MMSweep                string `sql:"type:text"`
	Del                    string `csv:"skip" sql:"-"`
	Pay                    string `sql:"type:text"`
	Div                    string `sql:"type:text"`
	Pri                    string `csv:"skip" sql:"-"`
	ValCashPosition        string `csv:"skip" sql:"-"`
	AccountClass           string `sql:"type:text"`
	AccountBalanceCategory string `csv:"skip" sql:"-"`
	PrintStatement         string `sql:"type:text"`
	FundPrincipal          string `csv:"skip" sql:"-"`
	FundInterest           string `csv:"skip" sql:"-"`
	Discretion             string `sql:"type:text"`
	PortfolioIndicator     string `sql:"type:text"`
	IRA                    string `sql:"type:text"`
	CreditInterestSweep    string `sql:"type:text"`
	MoneySweep             string `sql:"type:text"`
	OptionLevel            string `sql:"type:text"`
	Exposure               int
	NYTax                  string `sql:"type:text"`
	StateTax               string `sql:"type:text"`
	IRSCode                string `sql:"type:text"`
	FedTypeCode            string `csv:"skip" sql:"-"`
	NonObjecting           string `sql:"type:text"`
	IRSExempt              string `sql:"type:text"`
	ForeignCode            string `sql:"type:text"`
	Mature                 string `csv:"skip" sql:"-"`
	FundFees               string `csv:"skip" sql:"-"`
	OptionLimit            int
	ForwardLimit           int     `csv:"skip" sql:"-"`
	SuppressConfirm        string  `sql:"type:text"`
	RestrictReasonCode     string  `sql:"type:text"`
	CommissionSchedule     string  `csv:"skip" sql:"-"`
	W8                     string  `sql:"type:text"`
	FundDividend           string  `csv:"skip" sql:"-"`
	FundMature             string  `csv:"skip" sql:"-"`
	Joint                  string  `sql:"type:text"`
	Legal                  string  `csv:"skip" sql:"-"`
	Margin                 string  `sql:"type:text"`
	OptionCode             string  `sql:"type:text"`
	CorpRes                string  `csv:"skip" sql:"-"`
	Loan                   string  `csv:"skip" sql:"-"`
	NPLoan                 string  `csv:"skip" sql:"-"`
	Commodities            string  `csv:"skip" sql:"-"`
	PowerAttorney          string  `sql:"type:text"`
	IntFutures             string  `csv:"skip" sql:"-"`
	CurFutures             string  `csv:"skip" sql:"-"`
	SegCode                string  `csv:"skip" sql:"-"`
	PubEntity              string  `csv:"skip" sql:"-"`
	TradeAuthorize         string  `csv:"skip" sql:"-"`
	TradeAccountAuthorize  string  `csv:"skip" sql:"-"`
	ForwardDel             string  `csv:"skip" sql:"-"`
	Repo                   string  `csv:"skip" sql:"-"`
	Customer               string  `csv:"skip" sql:"-"`
	RiskDisclose           string  `csv:"skip" sql:"-"`
	NonCashDisclose        string  `csv:"skip" sql:"-"`
	HedgeLetter            string  `csv:"skip" sql:"-"`
	Unsolicited            string  `csv:"skip" sql:"-"`
	Partnership            string  `csv:"skip" sql:"-"`
	SoleOwner              string  `csv:"skip" sql:"-"`
	FedFunds               string  `csv:"skip" sql:"-"`
	SafeKeeping            string  `csv:"skip" sql:"-"`
	ACH                    string  `csv:"skip" sql:"-"`
	MAS                    string  `csv:"skip" sql:"-"`
	DVP                    string  `sql:"type:text"`
	DiscoluserStatement    string  `csv:"skip" sql:"-"`
	Sweep                  string  `sql:"type:text"`
	FundRateClass          string  `csv:"skip" sql:"-"`
	SICCode                string  `csv:"skip" sql:"-"`
	UserCode1              string  `csv:"skip" sql:"-"`
	UserCode2              string  `csv:"skip" sql:"-"`
	UserCode3              string  `csv:"skip" sql:"-"`
	UserCode4              string  `csv:"skip" sql:"-"`
	UserCode5              string  `csv:"skip" sql:"-"`
	UserCode6              string  `csv:"skip" sql:"-"`
	Institution            string  `sql:"type:text"`
	AgentBank              string  `sql:"type:text"`
	CustomerAccount        string  `csv:"skip" sql:"-"`
	ABANumber              string  `csv:"skip" sql:"-"`
	AlternateAccountTypeA  string  `csv:"skip" sql:"-"`
	ABANumberB             string  `csv:"skip" sql:"-"`
	AlternateAccountTypeB  string  `csv:"skip" sql:"-"`
	TelcoExtension1        string  `sql:"type:text"`
	TelcoExtension2        string  `sql:"type:text"`
	RestrDate              *string `sql:"type:date"`
	OpenDDate              *string `sql:"type:date"`
	LastChangeDate         *string `sql:"type:date"`
	LastActivityDate       *string `sql:"type:date"`
	LastTradeDate          string  `csv:"skip" sql:"-"`
	StockBorrowLoan        string  `csv:"skip" sql:"-"`
	InventoryCarryType     string  `csv:"skip" sql:"-"`
	NumberID               string  `csv:"skip" sql:"-"`
	AddressIndicator       string  `sql:"type:text"`
	TelcoCode1             string  `sql:"type:text"`
	TelcoCode2             string  `sql:"type:text"`
	TelcoAreaCode1         string  `sql:"type:text"`
	TelcoExchange1         string  `sql:"type:text"`
	TelcoBase1             string  `sql:"type:text"`
	TelcoAreaCode2         string  `sql:"type:text"`
	TelcoExchange2         string  `sql:"type:text"`
	TelcoBase2             string  `sql:"type:text"`
	DryTradeCounter        int     `csv:"skip" sql:"-"`
	OldSystemAccountNumber string  `sql:"type:text"`
	AlternateAccountB      string  `csv:"skip" sql:"-"`
	AlternateAccountA      string  `csv:"skip" sql:"-"`
	ClosedDate             *string `sql:"type:date"`
	PreEffAYY              string  `csv:"skip" sql:"-"`
	PreEffBYY              string  `csv:"skip" sql:"-"`
	TefraChangeYY          string  `sql:"type:text"`
	BankFailReport         string  `csv:"skip" sql:"-"`
	IssuerStatus           string  `csv:"skip" sql:"-"`
	ARBCode                string  `csv:"skip" sql:"-"`
	ProcessDate            *string `sql:"type:date"`
	AccountNature          string  `sql:"type:text"`
}

type AccountMaster struct {
	accounts []SoDAccount
}

func (am *AccountMaster) ExtCode() string {
	return "EXT765"
}

func (am *AccountMaster) Delimiter() string {
	return "|"
}

func (am *AccountMaster) Header() bool {
	return false
}

func (am *AccountMaster) Extension() string {
	return "txt"
}

func (am *AccountMaster) Value() reflect.Value {
	return reflect.ValueOf(am.accounts)
}

func (am *AccountMaster) Append(v interface{}) {
	am.accounts = append(am.accounts, v.(SoDAccount))
}

// Sync compares the start of day account records Apex has
// with the records in the DB. It does so by comparing name,
// city, state, and zip code.
func (am *AccountMaster) Sync(asOf time.Time) (uint, uint) {
	errors := []models.BatchError{}

	for _, sodAcct := range am.accounts {
		acct := &models.Account{}

		if IsFirmAccount(sodAcct.AccountNumber) {
			continue
		}

		// find the account
		q := db.DB().
			Where("apex_account = ?", sodAcct.AccountNumber).
			Preload("Owners").
			Preload("Owners.Details", "replaced_by IS NULL").
			Find(&acct)

		if q.RecordNotFound() {
			if utils.Prod() {
				errors = append(errors, am.genError(asOf, sodAcct, fmt.Errorf("account not found")))
			}
			continue
		}

		if q.Error != nil {
			log.Panic("start of day database error", "file", am.ExtCode(), "error", q.Error)
		}

		if err := am.validate(acct, sodAcct); err != nil {
			errors = append(errors, am.genError(asOf, sodAcct, err))
		}
	}

	StoreErrors(errors)

	return uint(len(am.accounts) - len(errors)), uint(len(errors))
}

func (am *AccountMaster) genError(asOf time.Time, sodAcct SoDAccount, err error) models.BatchError {
	log.Error("start of day error", "file", am.ExtCode(), "error", err)
	buf, _ := json.Marshal(map[string]interface{}{
		"error":       err.Error(),
		"sod_account": sodAcct,
	})
	return models.BatchError{
		ProcessDate:             asOf.Format("2006-01-02"),
		FileCode:                am.ExtCode(),
		PrimaryRecordIdentifier: sodAcct.AccountNumber,
		Error:                   buf,
	}
}

func (am *AccountMaster) validate(acct *models.Account, sodAcct SoDAccount) error {
	details := acct.Owners[0].Details

	switch {
	case !strings.Contains(sodAcct.AccountName, strings.ToUpper(*details.GivenName)):
		fallthrough
	case !strings.Contains(sodAcct.AccountName, strings.ToUpper(*details.FamilyName)):
		return fmt.Errorf("names don't match")
	case !strings.EqualFold(strings.ToUpper(form.CityForApex(*details.City)), sodAcct.City):
		return fmt.Errorf("cities don't match")
	case !strings.EqualFold(*details.State, sodAcct.State):
		return fmt.Errorf("states don't match")
	case !strings.EqualFold(*details.PostalCode, sodAcct.ZipCode[0:5]):
		return fmt.Errorf("zip codes don't match")
	default:
		return nil
	}
}
