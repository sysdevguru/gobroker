package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/utils"
	"github.com/alpacahq/gobroker/utils/date"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/shopspring/decimal"
)

type Account struct {
	ID                       string                  `json:"id" gorm:"primary_key" sql:"type:uuid;"`
	CognitoID                *string                 `json:"-" sql:"type:uuid;"`
	CreatedAt                time.Time               `json:"created_at"`
	UpdatedAt                time.Time               `json:"updated_at"`
	DeletedAt                *time.Time              `json:"deleted_at"`
	Status                   enum.AccountStatus      `json:"status" gorm:"type:text;not null"`
	Currency                 string                  `json:"currency" sql:"default:'USD'"`
	Cash                     decimal.Decimal         `json:"-" gorm:"type:decimal;not null"`
	CashWithdrawable         decimal.Decimal         `json:"-" gorm:"type:decimal;not null"`
	Plan                     enum.AccountPlan        `json:"plan" gorm:"type:varchar(20);not null" sql:"default:'REGULAR'"`
	ApexAccount              *string                 `json:"apex_account" gorm:"type:varchar(13);unique_index"`
	ApexRequestID            string                  `json:"apex_request_id" gorm:"type:varchar(36)"`
	ApexApprovalStatus       enum.ApexApprovalStatus `json:"apex_approval_status" gorm:"type:varchar(24);not null"`
	ProtectPatternDayTrader  bool                    `json:"-" gorm:"not null" sql:"default:TRUE"`
	PatternDayTrader         bool                    `json:"pattern_day_trader"`
	MarkedPatternDayTraderAt *date.Date              `json:"marked_pattern_day_trader_at" sql:"type:date"`
	AccReqResult             string                  `json:"acc_req_result" sql:"type:text"`
	TradingBlocked           bool                    `json:"trading_blocked"`
	RiskyTransfers           bool                    `json:"risky_transfers"` // Risky Business
	MarkedRiskyTransfersAt   *date.Date              `json:"marked_risky_transfers_at" sql:"type:date"`
	AccountBlocked           bool                    `json:"account_blocked"`
	TradeSuspendedByUser     bool                    `json:"trade_suspended_by_user" gorm:"not null" sql:"default:FALSE"`
	Positions                []Position              `json:"-" gorm:"ForeignKey:AccountID"`
	Relationships            []ACHRelationship       `json:"-" gorm:"ForeignKey:AccountID"`
	Transfers                []Transfer              `json:"-" gorm:"ForeignKey:AccountID"`
	Snaps                    []Snap                  `json:"-" gorm:"ForeignKey:AccountID"`
	Investigations           []Investigation         `json:"-" gorm:"ForeignKey:AccountID"`
	Owners                   []Owner                 `json:"-" gorm:"many2many:account_owners;"`
	AccessKeys               []AccessKey             `json:"-" gorm:"ForeignKey:AccountID"`
	MarginCalls              []MarginCall            `json:"-" gorm:"ForeignKey:AccountID"`

	Name  string `json:"name" gorm:"-"`  // for compatibility
	Email string `json:"email" gorm:"-"` // for compatibility
}

func (a *Account) ToTradeAccount() (*TradeAccount, error) {
	acc := TradeAccount{
		ID:                       a.ID,
		Status:                   a.Status,
		Currency:                 a.Currency,
		Cash:                     a.Cash,
		CashWithdrawable:         a.CashWithdrawable,
		ApexAccount:              a.ApexAccount,
		ApexApprovalStatus:       a.ApexApprovalStatus,
		ProtectPatternDayTrader:  a.ProtectPatternDayTrader,
		PatternDayTrader:         a.PatternDayTrader,
		MarkedPatternDayTraderAt: a.MarkedPatternDayTraderAt,
		TradingBlocked:           a.TradingBlocked,
		AccountBlocked:           a.AccountBlocked,
		CreatedAt:                a.CreatedAt,
		TradeSuspendedByUser:     a.TradeSuspendedByUser,
	}

	o := a.PrimaryOwner()

	if o == nil {
		return nil, errors.New("primary owner required")
	}

	if o.Details.LegalName != nil {
		acc.LegalName = *o.Details.LegalName
	}

	return &acc, nil
}

func (a *Account) IDAsUUID() uuid.UUID {
	id, _ := uuid.FromString(a.ID)
	return id
}

func (a *Account) BeforeCreate(scope *gorm.Scope) error {
	if a.ID == "" {
		a.ID = uuid.Must(uuid.NewV4()).String()
	}
	return scope.SetColumn("id", a.ID)
}

func (a *Account) PrimaryOwner() *Owner {
	if len(a.Owners) > 0 {
		if a.Owners[0].Details.HashSSN != nil {
			a.Owners[0].Details.MaskedSSN = "xxx-xx-xxxx"
		}
		return &a.Owners[0]
	}
	return nil
}

// OwnerUpdatable returns whether the account has reached
// a state that will allow updates to the owner's details
func (a *Account) OwnerUpdatable() bool {
	// don't block for dev
	if utils.Dev() {
		return true
	}

	if a.Status == enum.Edited {
		return false
	}

	switch a.ApexApprovalStatus {
	// new account (no investigation started)
	case "":
		fallthrough
	// complete investigation
	case enum.Complete:
		return true
	default:
		return false
	}
}

// Modifiable returns whether the specified field of the
// account object is modifiable via the API
func (a *Account) Modifiable(field string) bool {
	// modify anything in DEV mode
	if utils.Dev() {
		return true
	}

	switch field {
	case "plan":
		return true
	default:
		return false
	}
}

// Tradable returns whether or not the account is
// allowed to trade
func (a *Account) Tradable() bool {
	return (a.ApexAccount != nil &&
		(a.Status == enum.Active ||
			a.Status == enum.Resubmitted ||
			a.Status == enum.ReapprovalPending) &&
		!a.TradingBlocked &&
		!a.AccountBlocked)
}

// Fundable returns whether or not the account is
// allowed to be funded
func (a *Account) Fundable() bool {
	return (a.ApexAccount != nil &&
		a.ApexApprovalStatus == enum.Complete &&
		a.Status == enum.Active &&
		!a.AccountBlocked)
}

func (a *Account) Linkable() bool {
	return (a.ApexAccount != nil &&
		a.Status != enum.Rejected &&
		!a.AccountBlocked)
}

// hack for now to make object compatibile with previous DB model
func (a *Account) ForJSON() Account {
	if len(a.Owners) > 0 {
		primaryOwner := a.Owners[0]
		if primaryOwner.Details.LegalName != nil {
			a.Name = *primaryOwner.Details.LegalName
		}
		a.Email = primaryOwner.Email
	}
	return *a
}

type IntradayBalances struct {
	CashWithdrawable decimal.Decimal
	Cash             decimal.Decimal
	BuyingPower      decimal.Decimal
}

type DocumentRequestStatus string

var (
	DocumentRequestRequested DocumentRequestStatus = "REQUESTED"
	DocumentRequestUploaded  DocumentRequestStatus = "UPLOADED"
)

type DocumentCategory string

func (d DocumentCategory) String() string {
	return string(d)
}

func (d DocumentCategory) Types() []DocumentType {
	return docCategoryToTypeM[d]
}

func NewDocumentCategory(str string) (*DocumentCategory, error) {
	docCat := DocumentCategory(str)

	if _, ok := docCategoryToTypeM[docCat]; !ok {
		return nil, fmt.Errorf("invalid document category")
	}

	return &docCat, nil
}

const (
	UPIC DocumentCategory = "UPIC" // Non-Expired Government issued ID
	UTIN DocumentCategory = "UTIN" // SSN card or a certified letter from the Social Security Administration
	UIRS DocumentCategory = "UIRS" // IRS assignment letter of an Individual Taxpayer Identification Number
	UCIP DocumentCategory = "UCIP" // Firm’s CIP passing results, dated within 30 days of the associated account opening
	UPTA DocumentCategory = "UPTA" // Letter 407
)

var (
	docTypeToCategoryM = map[DocumentType]DocumentCategory{
		DriverLicense:         UPIC,
		StateID:               UPIC,
		Passport:              UPIC,
		SSNCard:               UTIN,
		SSACertificate:        UTIN,
		IRSIssuanceLetter:     UIRS,
		PermanentResidentCard: UCIP,
		UtilityBill:           UCIP,
		Letter407:             UPTA,
	}

	docCategoryToTypeM = map[DocumentCategory][]DocumentType{
		UPIC: []DocumentType{DriverLicense, StateID, Passport},
		UTIN: []DocumentType{SSNCard, SSACertificate},
		UIRS: []DocumentType{IRSIssuanceLetter},
		UCIP: []DocumentType{PermanentResidentCard, UtilityBill},
		UPTA: []DocumentType{Letter407},
	}
)

type DocumentType string

const (
	// UPIC
	DriverLicense DocumentType = "DRIVER_LICENSE"
	StateID       DocumentType = "STATE_ID"
	Passport      DocumentType = "PASSPORT"
	// UTIN
	SSNCard        DocumentType = "SSN_CARD"
	SSACertificate DocumentType = "SSA_CERTIFICATE"
	// UIRS
	IRSIssuanceLetter DocumentType = "IRS_ISSUANCE_LETTER"
	// UCIP
	PermanentResidentCard DocumentType = "PERMANENT_RESIDENT_CARD"
	UtilityBill           DocumentType = "UTILITY_BILL"
	// UPTA
	Letter407 DocumentType = "407_LETTER"
)

func (d DocumentType) String() string {
	return string(d)
}

func (d DocumentType) Category() DocumentCategory {
	return docTypeToCategoryM[d]
}

var frontBackTypes = []DocumentType{
	StateID,
	DriverLicense,
	PermanentResidentCard,
}

// SupportFrontBack returns whether it requires both front / back image of the document or not.
// In some document, user need to send both front / back image.
func (d DocumentType) SupportFrontBack() bool {
	for _, t := range frontBackTypes {
		if d == t {
			return true
		}
	}
	return false
}

func NewDocumentType(str string) (*DocumentType, error) {
	docType := DocumentType(str)

	if _, ok := docTypeToCategoryM[docType]; !ok {
		return nil, fmt.Errorf("invalid document type")
	}

	return &docType, nil
}

type DocumentSubType string

func (d DocumentSubType) String() string {
	return string(d)
}

func NewDocumentSubType(str string) (DocumentSubType, error) {
	dtype := DocumentSubType(str)

	types := []DocumentSubType{
		Front,
		Back,
		Simple,
	}

	for _, t := range types {
		if dtype == t {
			return dtype, nil
		}
	}

	return dtype, fmt.Errorf("invalid document_sub_type")
}

const (
	Front  DocumentSubType = "FRONT"
	Back   DocumentSubType = "BACK"
	Simple DocumentSubType = "SIMPLE" // front only
)

type DocumentRequest struct {
	ID              string                `json:"id" gorm:"primary_key" sql:"type:uuid;"`
	AccountID       string                `json:"account_id" gorm:"not null;index" sql:"type:uuid;"`
	InvestigationID string                `json:"investigation_id" gorm:"not null;index" sql:"type:varchar(100);"`
	DocumentType    DocumentType          `json:"document_type" gorm:"not null"`
	CreatedAt       time.Time             `json:"created_at" gorm:"not null;index;"`
	UpdatedAt       time.Time             `json:"updated_at" gorm:"not null;"`
	Status          DocumentRequestStatus `json:"status" gorm:"type:text;not null"`
	Snaps           []Snap                `json:"-" gorm:"foreignkey:DocumentRequestID"`
}

func (d *DocumentRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"id":                d.ID,
		"account_id":        d.AccountID,
		"investigation_id":  d.InvestigationID,
		"document_type":     d.DocumentType,
		"document_category": d.DocumentType.Category(),
		"created_at":        d.CreatedAt.Format(time.RFC3339),
		"updated_at":        d.UpdatedAt.Format(time.RFC3339),
		"status":            d.Status,
	})
}

func (d *DocumentRequest) BeforeCreate(scope *gorm.Scope) error {
	if d.ID == "" {
		d.ID = uuid.Must(uuid.NewV4()).String()
	}
	return scope.SetColumn("id", d.ID)
}

func (d *DocumentRequest) IsComplete(snap *Snap) bool {
	if d.DocumentType.SupportFrontBack() {
		fname := Front.String()
		bname := Back.String()
		hasFront := snap.Name == fname
		hasBack := snap.Name == bname
		for i := range d.Snaps {
			if d.Snaps[i].Name == bname {
				hasBack = true
				continue
			}
			if d.Snaps[i].Name == fname {
				hasFront = true
				continue
			}
		}
		if hasBack && hasFront {
			return true
		}
	} else {
		return true
	}
	return false
}

type Cash struct {
	ID        string          `json:"id" gorm:"primary_key" sql:"type:uuid;"`
	AccountID string          `json:"account_id" gorm:"not null;unique_index:uix_cashes" sql:"type:uuid references accounts(id);"`
	Value     decimal.Decimal `json:"value" gorm:"type:decimal;not null"`
	Date      date.Date       `json:"date" gorm:"not null;unique_index:uix_cashes" sql:"type:date"`
}

func (c *Cash) BeforeCreate(scope *gorm.Scope) error {
	if c.ID == "" {
		c.ID = uuid.Must(uuid.NewV4()).String()
	}
	return scope.SetColumn("id", c.ID)
}
