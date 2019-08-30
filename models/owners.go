package models

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/utils/address"
	"github.com/alpacahq/gomarkets/sources"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/clock"
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/copier"
	"github.com/jinzhu/gorm"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

type Owner struct {
	ID        string       `json:"id" gorm:"primary_key" sql:"type:uuid;"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
	DeletedAt *time.Time   `json:"-"`
	Email     string       `valid:"email" json:"email" gorm:"type:varchar(100);unique_index"`
	Primary   bool         `json:"primary"`
	Accounts  []Account    `json:"-" gorm:"many2many:account_owners;"`
	Details   OwnerDetails `json:"details" gorm:"ForeignKey:OwnerID"`
}

func (o *Owner) IDAsUUID() uuid.UUID {
	id, _ := uuid.FromString(o.ID)
	return id
}

func (o *Owner) BeforeCreate(scope *gorm.Scope) error {
	if o.ID == "" {
		o.ID = uuid.Must(uuid.NewV4()).String()
	}
	return scope.SetColumn("id", o.ID)
}

type Horizon string

const (
	SHORT   Horizon = "short"
	AVERAGE Horizon = "average"
	LONGEST Horizon = "longest"
)

type Employment string

const (
	Employed   Employment = "EMPLOYED"
	Unemployed Employment = "UNEMPLOYED"
	Retired    Employment = "RETIRED"
	Student    Employment = "STUDENT"
)

type Visa string

const (
	E1  Visa = "E1"
	E2  Visa = "E2"
	E3  Visa = "E3"
	F1  Visa = "F1"
	H1B Visa = "H1B"
	TN1 Visa = "TN1"
	O1  Visa = "O1"
)

type Marital string

const (
	Single   Marital = "SINGLE"
	Married  Marital = "MARRIED"
	Divorced Marital = "DIVORCED"
	Widowed  Marital = "WIDOWED"
)

type OwnerDetails struct {
	ID             uint            `json:"id" gorm:"primary_key"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	DeletedAt      *time.Time      `json:"deleted_at"`
	OwnerID        string          `json:"owner_id"`
	Prefix         *string         `json:"prefix" gorm:"type:varchar(7)"`
	GivenName      *string         `json:"given_name" sql:"type:text"`
	AdditionalName *string         `json:"additional_name" sql:"type:text"`
	FamilyName     *string         `json:"family_name" sql:"type:text"`
	Suffix         *string         `json:"suffix" gorm:"type:varchar(3)"`
	LegalName      *string         `json:"legal_name" sql:"type:text"`
	DateOfBirth    *string         `json:"date_of_birth" sql:"type:date"`
	HashSSN        *[]byte         `json:"-" gorm:"type:bytea"`
	PhoneNumber    *string         `json:"phone_number" sql:"type:text"`
	StreetAddress  address.Address `json:"street_address" sql:"type:text[]"`
	Unit           *string         `json:"unit" gorm:"type:varchar(20)"`
	City           *string         `json:"city" sql:"type:text"`
	State          *string         `json:"state" gorm:"type:varchar(2)"`
	PostalCode     *string         `json:"postal_code" gorm:"type:varchar(5)"`
	// Country/citizenship info
	CountryOfCitizenship *string `json:"country_of_citizenship" sql:"type:text"`
	CountryOfBirth       *string `json:"country_of_birth" sql:"type:text"`
	PermanentResident    *bool   `json:"permanent_resident"`
	VisaType             *Visa   `json:"visa_type" gorm:"type:varchar(5)"`
	VisaExpirationDate   *string `json:"visa_expiration_date" sql:"type:date"`
	// Tax info
	EmploymentStatus   *Employment      `json:"employment_status" gorm:"type:varchar(12)"`
	Employer           *string          `json:"employer" sql:"type:text"`
	EmployerAddress    *string          `json:"employer_address" sql:"type:text"`
	Position           *string          `json:"position" sql:"type:text"`
	Function           *string          `json:"function" sql:"type:text"`
	YearsEmployed      *uint            `json:"years_employed"`
	TaxBracket         *decimal.Decimal `json:"tax_bracket" gorm:"type:decimal"`
	MaritalStatus      *Marital         `json:"marital_status" sql:"type:text"`
	NumberOfDependents *uint            `json:"number_of_dependents"`
	// Suitability info
	IsControlPerson             *bool            `json:"is_control_person"`
	ControllingFirms            *pq.StringArray  `json:"controlling_firms" gorm:"type:varchar(10)[]"`
	IsAffiliatedExchangeOrFINRA *bool            `json:"is_affiliated_exchange_or_finra"`
	AffiliatedFirm              *string          `json:"affiliated_firm" gorm:"type:varchar(100)"`
	IsPoliticallyExposed        *bool            `json:"is_politically_exposed"`
	ImmediateFamilyExposed      *pq.StringArray  `json:"immediate_family_exposed" gorm:"type:varchar(100)[]"`
	PoliticalOrganization       *string          `json:"political_organization" sql:"type:text"`
	TimeHorizon                 *Horizon         `json:"time_horizon" gorm:"type:varchar(7)"`
	LiquidityNeeds              *decimal.Decimal `json:"liquidity_needs" gorm:"type:decimal"`
	InvestmentObjective         *string          `json:"investment_objective" sql:"type:text"`
	InvestmentExperience        *string          `json:"investment_experience" sql:"type:text"`
	AnnualIncomeMin             *decimal.Decimal `json:"annual_income_min" gorm:"type:decimal"`
	AnnualIncomeMax             *decimal.Decimal `json:"annual_income_max" gorm:"type:decimal"`
	LiquidNetWorthMin           *decimal.Decimal `json:"liquid_net_worth_min" gorm:"type:decimal"`
	LiquidNetWorthMax           *decimal.Decimal `json:"liquid_net_worth_max" gorm:"type:decimal"`
	TotalNetWorthMin            *decimal.Decimal `json:"total_net_worth_min" gorm:"type:decimal"`
	TotalNetWorthMax            *decimal.Decimal `json:"total_net_worth_max" gorm:"type:decimal"`
	RiskTolerance               *string          `json:"risk_tolerance" gorm:"type:text"`
	IncludeTrustedContact       bool             `json:"include_trusted_contact"`
	MarginAgreementSigned       *bool            `json:"margin_agreement_signed"`
	MarginAgreementSignedAt     *time.Time       `json:"margin_agreement_signed_at"`
	AccountAgreementSigned      *bool            `json:"account_agreement_signed"`
	AccountAgreementSignedAt    *time.Time       `json:"account_agreement_signed_at"`
	NasdaqAgreementSignedAt     *time.Time       `json:"nasdaq_agreement_signed_at"`
	NyseAgreementSignedAt       *time.Time       `json:"nyse_agreement_signed_at"`
	// to indicate in API that SSN is set or not
	MaskedSSN string `json:"masked_ssn" sql:"-"`
	// for history
	ReplacedAt *time.Time `json:"replaced_at"`
	ReplacedBy *uint      `json:"replaced_by"`
	Replaces   *uint      `json:"replaces"`
	// Principal who approved (name)
	ApprovedBy *string    `json:"approved_by" sql:"type:text"`
	ApprovedAt *time.Time `json:"approved_at"`
	// Administrator assigned to account
	AssignedAdminID *string        `json:"assigned_admin_id" sql:"type:uuid;"`
	AssignedAdmin   *Administrator `json:"-" gorm:"ForeignKey:AdminID"`
}

func (od *OwnerDetails) Replacement() (repl OwnerDetails) {
	copier.Copy(&repl, od)

	repl.ID = 0
	repl.CreatedAt = time.Time{}
	repl.UpdatedAt = time.Time{}
	repl.Replaces = &od.ID
	repl.ApprovedBy = nil
	repl.ReplacedAt = nil

	return
}

func (od *OwnerDetails) DataSources() []string {
	srcs := []string{string(sources.IEX)}

	if od.NasdaqAgreementSignedAt != nil &&
		od.NyseAgreementSignedAt != nil {
		srcs = append(srcs, string(sources.SIP))
	}

	return srcs
}

func (od *OwnerDetails) FormatAddress() (string, error) {
	var addr string
	if od.StreetAddress != nil && len(od.StreetAddress) > 0 {
		addr = od.StreetAddress[0]
	} else {
		return "", gberrors.Forbidden.WithMsg("format address failed because no address was given")
	}

	if len(od.StreetAddress) > 1 {
		for _, line := range od.StreetAddress[1:] {
			addr = addr + " " + line
		}
	}

	return strings.Join(
		[]string{
			addr,
			*od.City,
			*od.State,
			*od.PostalCode}, ", "), nil
}

func (od *OwnerDetails) DateOfBirthString() *string {
	if od.DateOfBirth != nil {
		dob, _ := time.Parse(time.RFC3339, *od.DateOfBirth)
		dobStr := dob.Format("2006-01-02")
		return &dobStr
	}
	return nil
}

func (od *OwnerDetails) VisaExpirationDateString() *string {
	if od.VisaExpirationDate != nil {
		dob, _ := time.Parse(time.RFC3339, *od.VisaExpirationDate)
		dobStr := dob.Format("2006-01-02")
		return &dobStr
	}
	return nil
}

func (od *OwnerDetails) Validate() error {
	if od.DateOfBirth != nil {
		dob, err := time.Parse("2006-01-02", *od.DateOfBirth)
		if err != nil {
			return err
		}
		minDOB := time.Date(1900, 1, 1, 0, 0, 0, 0, calendar.NY)
		maxDOB := clock.Now().Add(-18 * calendar.Year)
		err = validation.Validate(
			dob,
			validation.Min(minDOB),
			validation.Max(maxDOB),
		)
		if err != nil {
			return fmt.Errorf(
				"Date of birth must be between %v and %v",
				minDOB.Year(),
				maxDOB.Year(),
			)
		}
	}

	if od.VisaExpirationDate != nil {
		dob, err := time.Parse("2006-01-02", *od.VisaExpirationDate)
		if err != nil {
			return err
		}
		minDOB := clock.Now().In(calendar.NY)
		err = validation.Validate(
			dob,
			validation.Min(minDOB),
		)
		if err != nil {
			return fmt.Errorf(
				"Visa expiration date must be greater than %v",
				minDOB.Format("2006-01-02"),
			)
		}
	}

	if od.EmploymentStatus != nil {
		switch *od.EmploymentStatus {
		case Unemployed:
		case Employed:
		case Student:
		case Retired:
		default:
			return fmt.Errorf(
				"Employment status must be one of: %v",
				[]Employment{Unemployed, Employed, Student, Retired},
			)
		}
	}

	if od.MaritalStatus != nil {
		switch *od.MaritalStatus {
		case Single:
		case Married:
		case Divorced:
		case Widowed:
		default:
			return fmt.Errorf(
				"Marital status must be one of: %v",
				[]Marital{Single, Married, Divorced, Widowed},
			)
		}
	}

	if od.YearsEmployed != nil {
		yearsEmp := int64(*od.YearsEmployed)
		err := validation.Validate(yearsEmp, validation.Min(0), validation.Max(100))
		if err != nil {
			return errors.New("Years employed must be between 0 and 100")
		}
	}

	if od.Prefix != nil {
		if len(*od.Prefix) < 2 || len(*od.Prefix) > 7 {
			return errors.New("Unsupported prefix")
		}
	}

	if od.Suffix != nil {
		if len(*od.Suffix) < 2 || len(*od.Suffix) > 3 {
			return errors.New("Unsupported suffix")
		}
	}

	if od.VisaType != nil {
		if len(*od.VisaType) < 2 || len(*od.VisaType) > 5 {
			return errors.New("Invalid visa type")
		}
	}

	if od.PhoneNumber != nil {
		err := validation.Validate(od.PhoneNumber, validation.Match(
			regexp.MustCompile(`^\s*(?:\+?(\d{1,3}))?[-. (]*(\d{3})[-. )]*(\d{3})[-. ]*(\d{4})(?: *x(\d+))?\s*$`)))
		if err != nil {
			return errors.New("Invalid phone number")
		}
	}

	if od.Unit != nil {
		if len(*od.Unit) > 20 {
			return errors.New("Unit/Apt # too long, must be less than 20 characters")
		}
	}

	if od.StreetAddress != nil {
		addr := []string{}
		for _, line := range od.StreetAddress {
			validation.Validate(line, validation.Length(1, 31))
			if len(line) < 1 || len(line) > 30 {
				return errors.New("line in street address is > 30 characters")
			}
			addr = append(addr, line)
		}
		if len(od.StreetAddress) < 1 || len(od.StreetAddress) > 3 {
			return errors.New("street address must be 3 lines or less")
		}
	}

	if od.City != nil {
		if len(*od.City) < 2 || len(*od.City) > 50 {
			return errors.New("Invalid city name")
		}
	}

	if od.State != nil {
		err := validation.Validate(od.State, validation.Match(regexp.MustCompile("^[A-Z]{2}$")))
		if err != nil {
			return errors.New("Invalid state abbreviation")
		}
	}

	if od.PostalCode != nil {
		err := validation.Validate(od.PostalCode, validation.Match(regexp.MustCompile("^[0-9]{5}$")))
		if err != nil {
			return errors.New("Invalid postal code")
		}
	}
	return nil
}

type Affiliate struct {
	gorm.Model
	AccountID       string             `json:"account_id" gorm:"not null;index" sql:"type:uuid;"`
	Type            enum.AffiliateType `json:"type" sql:"type:text"`
	StreetAddress   address.Address    `json:"street_address" sql:"type:text[]"`
	City            string             `json:"city" sql:"type:text;not null"`
	State           string             `json:"state" gorm:"type:varchar(2);not null"`
	PostalCode      string             `json:"postal_code" gorm:"type:varchar(5);not null"`
	Country         string             `json:"country" gorm:"type:text;not null" sql:"DEFAULT:'USA'"`
	CompanyName     string             `json:"company_name" gorm:"type:varchar(100)"`
	CompanySymbol   string             `json:"company_symbol" sql:"type:text"`
	AdditionalName  *string            `json:"additional_name" gorm:"type:varchar(100)"`
	ComplianceEmail string             `json:"compliance_email" gorm:"type:varchar(100)"`
}

type TrustedContact struct {
	gorm.Model
	AccountID     string          `json:"account_id" gorm:"not null;unique_index" sql:"type:uuid;"`
	EmailAddress  *string         `valid:"email" json:"email_address" sql:"type:text"`
	PhoneNumber   *string         `json:"phone_number" sql:"type:text"`
	StreetAddress address.Address `json:"street_address" sql:"type:text[]"`
	City          *string         `json:"city" sql:"type:text;"`
	State         *string         `json:"state" gorm:"type:varchar(2);"`
	PostalCode    *string         `json:"postal_code" gorm:"type:varchar(5);"`
	Country       *string         `json:"country" sql:"type:text;"`
	GivenName     string          `json:"given_name" sql:"type:text"`
	FamilyName    string          `json:"family_name" sql:"type:text"`
}

type EmailVerificationCode struct {
	ID       uint      `json:"id" gorm:"primary_key"`
	OwnerID  string    `json:"owner_id" sql:"type uuid references owners(id)"`
	Email    string    `json:"email" gorm:"not null"`
	Code     string    `json:"code" gorm:"not null;unique_index"`
	ExpireAt time.Time `json:"expire_at" gorm:"not null"`
}

func NewEmailVerificationCode(ownerID string, email string) (*EmailVerificationCode, error) {
	code := getStringWithCharset(5, "0123456789")
	return &EmailVerificationCode{
		OwnerID:  ownerID,
		Email:    email,
		Code:     code,
		ExpireAt: clock.Now().Add(1 * time.Hour),
	}, nil
}
