package form

import (
	"regexp"
	"strings"
	"time"

	"github.com/alpacahq/apex/forms"
	"github.com/alpacahq/apex/forms/v1"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/encryption"
	"github.com/alpacahq/gopaca/env"
	"github.com/alpacahq/gopaca/log"
	"github.com/pkg/errors"
)

func PrepareSubmission(account *models.Account, details *models.OwnerDetails, modifyType string) *forms.FormSubmission {
	submissionForms := []v1.Form{&v1.NewAccountForm{}, &v1.MarginForm{}}

	if (details.IsAffiliatedExchangeOrFINRA != nil && *details.IsAffiliatedExchangeOrFINRA) ||
		(details.IsControlPerson != nil && *details.IsControlPerson) {
		submissionForms = append(submissionForms, &v1.InterestedPartiesForm{})
	}
	if details.IncludeTrustedContact {
		submissionForms = append(submissionForms, &v1.TrustedContactForm{})
	}

	var err error

	for _, form := range submissionForms {
		log.Debug("account worker filling apex form", "form", form.Title())

		switch v := form.(type) {
		case *v1.TrustedContactForm:
			err = FillTrustedContactForm(v, account)
		case *v1.InterestedPartiesForm:
			err = FillInterestedPartiesForm(v, account, details)
		case *v1.MarginForm:
			err = FillMarginForm(v, account, details)
		case *v1.NewAccountForm:
			err = FillNewAccountForm(v, account, details)
		}
		if err != nil {
			log.Error(
				"account worker form fill failure",
				"form", form.Title(),
				"account", account.ID,
				"error", err)
			return nil
		}
	}

	acct := ""
	if modifyType == "UPDATE" && account.ApexAccount != nil {
		acct = *account.ApexAccount
	}

	return &forms.FormSubmission{
		ModifyType: modifyType,
		RepCode:    env.GetVar("APEX_REP_CODE"),
		Branch:     env.GetVar("APEX_BRANCH"),
		Forms:      submissionForms,
		Account:    acct,
	}
}

func FillInterestedPartiesForm(form *v1.InterestedPartiesForm, account *models.Account, details *models.OwnerDetails) error {
	form.FormID.Title = form.Title()
	form.FormID.Version = form.Version()
	form.FormSchemaHash.Algorithm = "SHA-256"

	if (details.IsAffiliatedExchangeOrFINRA != nil && *details.IsAffiliatedExchangeOrFINRA) ||
		(details.IsControlPerson != nil && *details.IsControlPerson) {

		affiliates := []models.Affiliate{}
		if err := db.DB().Where("account_id = ?", account.ID).Find(&affiliates).Error; err != nil {
			return err
		}

		form.JSONData.InterestedParties = make([]v1.InterestedParty, len(affiliates))

		for i, affiliate := range affiliates {
			party := v1.InterestedParty{}
			if len(affiliate.CompanyName) > 20 {
				party.Name.CompanyName = affiliate.CompanyName[:20]
			} else {
				party.Name.CompanyName = affiliate.CompanyName
			}
			if affiliate.Country != "" {
				party.MailingAddress = &v1.Address{
					Country:       affiliate.Country,
					City:          CityForApex(affiliate.City),
					PostalCode:    affiliate.PostalCode,
					State:         affiliate.State,
					StreetAddress: affiliate.StreetAddress,
				}
			}
			if affiliate.AdditionalName != nil {
				party.AdditionalName = *affiliate.AdditionalName
			}
			form.JSONData.InterestedParties[i] = party
		}
	}
	return nil
}

func FillMarginForm(form *v1.MarginForm, account *models.Account, details *models.OwnerDetails) error {
	form.FormID.Title = form.Title()
	form.FormID.Version = form.Version()
	form.FormSchemaHash.Algorithm = "SHA-256"

	// delivery date
	form.JSONData.DeliveryDate = details.MarginAgreementSignedAt.Format("2006-01-02")
	form.JSONData.Signature.ESigned = "YES"
	form.ESigned = true
	return nil
}

func FillNewAccountForm(form *v1.NewAccountForm, account *models.Account, details *models.OwnerDetails) error {
	form.FormID.Title = form.Title()
	form.FormID.Version = form.Version()
	form.FormSchemaHash.Algorithm = "SHA-256"

	// json form
	form.JSONData.CustomerType = "INDIVIDUAL"
	form.JSONData.ApplicantSignature.ESigned = "YES"
	form.JSONData.TradeAuthorization.IsTradeAuthorization = "NO"
	form.JSONData.ServiceProfile.DividendReinvestment = "DO_NOT_REINVEST"
	form.JSONData.ServiceProfile.IssuerDirectCommunication = "ACCEPT"

	// trusted contact (should not be set if this is an update to an existing account)
	if account.ApexApprovalStatus != enum.Complete {
		if details.IncludeTrustedContact {
			form.JSONData.TrustedContact = "INCLUDE"
		} else {
			form.JSONData.TrustedContact = "EXCLUDE"
		}
	}

	app := v1.Applicant{}

	// contact info
	app.Contact.EmailAddresses = []string{account.Owners[0].Email}
	if details.StreetAddress != nil && len(details.StreetAddress) > 0 {
		app.Contact.HomeAddress.StreetAddress = details.StreetAddress
	} else {
		return errors.New("StreetAddress is required")
	}
	if details.Unit != nil && *details.Unit != "" {
		app.Contact.HomeAddress.StreetAddress = append(app.Contact.HomeAddress.StreetAddress, *details.Unit)
	}
	if details.City != nil {
		app.Contact.HomeAddress.City = CityForApex(*details.City)
	} else {
		return errors.New("City is required")
	}
	if details.State != nil {
		app.Contact.HomeAddress.State = *details.State
	}
	if details.PostalCode != nil {
		app.Contact.HomeAddress.PostalCode = *details.PostalCode
	} else {
		return errors.New("PostalCode is required")
	}
	app.Contact.HomeAddress.Country = "USA"

	if details.PhoneNumber != nil {
		reg, _ := regexp.Compile("[^0-9]+")
		phone := reg.ReplaceAllString(*details.PhoneNumber, "")
		app.Contact.PhoneNumbers = []v1.PhoneNumber{
			v1.PhoneNumber{
				PhoneNumber:     phone,
				PhoneNumberType: "MOBILE",
			},
		}
	} else {
		return errors.New("PhoneNumber is required")
	}

	// employment
	if details.EmploymentStatus != nil {
		app.Employment.EmploymentStatus = string(*details.EmploymentStatus)
		if *details.EmploymentStatus == models.Employed {
			if details.Employer != nil {
				app.Employment.Employer = *details.Employer
			}
			if details.Position != nil {
				app.Employment.PositionEmployed = *details.Position
			}
		}
	} else {
		return errors.New("EmploymentStatus is required")
	}

	// identity
	if details.CountryOfCitizenship != nil {
		app.Identity.CitizenshipCountry = *details.CountryOfCitizenship
	}
	if details.VisaType != nil {
		app.Identity.VisaType = string(*details.VisaType)
	}
	if details.VisaExpirationDate != nil {
		{
			str := details.VisaExpirationDateString()
			app.Identity.VisaExpirationDate = *str
		}
	}
	if details.PermanentResident != nil {
		if *details.PermanentResident == true {
			app.Identity.PermanentResident = "YES"
		} else {
			app.Identity.PermanentResident = "NO"
		}
	}
	if details.CountryOfBirth != nil {
		app.Identity.BirthCountry = *details.CountryOfBirth
	}

	if details.DateOfBirth != nil {
		dob, err := time.Parse(time.RFC3339, *details.DateOfBirth)
		if err != nil {
			return err
		}
		app.Identity.DateOfBirth = dob.Format("2006-01-02")
	} else {
		return errors.New("DateOfBirth is required")
	}
	if details.Prefix != nil && *details.Prefix != "" {
		app.Identity.Name.Prefix = *details.Prefix
	}
	if details.Suffix != nil && *details.Suffix != "" {
		app.Identity.Name.Suffix = *details.Suffix
	}
	if details.FamilyName != nil && *details.FamilyName != "" {
		app.Identity.Name.FamilyName = *details.FamilyName
	} else {
		return errors.New("FamilyName is required")
	}
	if details.GivenName != nil && *details.GivenName != "" {
		app.Identity.Name.GivenName = *details.GivenName
	} else {
		return errors.New("GivenName is required")
	}
	if details.LegalName != nil && *details.LegalName != "" {
		app.Identity.Name.LegalName = *details.LegalName
	} else {
		return errors.New("LegalName is required")
	}
	if details.HashSSN == nil {
		return errors.New("SSN is required")
	}
	ssn, err := encryption.DecryptWithkey(*details.HashSSN, []byte(env.GetVar("BROKER_SECRET")))
	if err != nil {
		return err
	}
	app.Identity.SocialSecurityNumber = string(ssn)

	// disclosures
	if details.IsAffiliatedExchangeOrFINRA != nil && *details.IsAffiliatedExchangeOrFINRA {
		app.Disclosures.IsAffiliatedExchangeOrFINRA = "YES"
		affiliate := models.Affiliate{}
		if err := db.DB().Where("account_id = ? AND type = ?", account.ID, enum.FinraFirm).First(&affiliate).Error; err != nil {
			return err
		}
		if affiliate.CompanyName != "" {
			if len(affiliate.CompanyName) > 20 {
				app.Disclosures.FirmName = affiliate.CompanyName[:20]
			} else {
				app.Disclosures.FirmName = affiliate.CompanyName
			}
		} else {
			return errors.New("Firm Name is required")
		}
	} else {
		app.Disclosures.IsAffiliatedExchangeOrFINRA = "NO"
	}
	if details.IsControlPerson != nil && *details.IsControlPerson {
		app.Disclosures.IsControlPerson = "YES"
		firms := []models.Affiliate{}
		if err := db.DB().Where("account_id = ? AND type = ?", account.ID, enum.ControlledFirm).Find(&firms).Error; err != nil {
			return err
		}
		if len(firms) == 0 {
			return errors.New("Firms are required")
		}
		for _, firm := range firms {
			if firm.CompanySymbol != "" {
				app.Disclosures.CompanySymbols = append(app.Disclosures.CompanySymbols, firm.CompanySymbol)
			} else {
				return errors.New("Firm Name is required")
			}
		}
	} else {
		app.Disclosures.IsControlPerson = "NO"
	}
	if details.IsPoliticallyExposed != nil && *details.IsPoliticallyExposed {
		app.Disclosures.IsPoliticallyExposed = "YES"
		app.Disclosures.PoliticalExposureDetail = &struct {
			ImmediateFamily       []string `json:"immediateFamily"`
			PoliticalOrganization string   `json:"politicalOrganization"`
		}{}
		if details.ImmediateFamilyExposed != nil {
			app.Disclosures.PoliticalExposureDetail.ImmediateFamily = *details.ImmediateFamilyExposed
		}
		if details.PoliticalOrganization != nil {
			app.Disclosures.PoliticalExposureDetail.PoliticalOrganization = *details.PoliticalOrganization
		}
	} else {
		app.Disclosures.IsPoliticallyExposed = "NO"
	}

	// tax info
	if details.MaritalStatus != nil {
		app.MaritalStatus = string(*details.MaritalStatus)
	}
	if details.NumberOfDependents != nil {
		app.NumDependents = int64(*details.NumberOfDependents)
	}
	form.JSONData.Applicants = []v1.Applicant{app}
	return nil
}

func FillTrustedContactForm(form *v1.TrustedContactForm, account *models.Account) error {
	form.FormID.Title = form.Title()
	form.FormID.Version = form.Version()
	form.FormSchemaHash.Algorithm = "SHA-256"

	trustedContact := models.TrustedContact{}
	if err := db.DB().Where("account_id = ?", account.ID).Find(&trustedContact).Error; err != nil {
		return err
	}
	form.JSONData.GivenName = trustedContact.GivenName
	form.JSONData.FamilyName = trustedContact.FamilyName
	if trustedContact.EmailAddress != nil {
		form.JSONData.EmailAddress = *trustedContact.EmailAddress
	}
	if trustedContact.PhoneNumber != nil {
		form.JSONData.PhoneNumber = v1.PhoneNumber{
			PhoneNumber:     *trustedContact.PhoneNumber,
			PhoneNumberType: "MOBILE",
		}
	}
	if trustedContact.Country != nil {
		addr := &v1.Address{}
		addr.StreetAddress = trustedContact.StreetAddress
		if trustedContact.City != nil {
			addr.City = CityForApex(*trustedContact.City)
		}
		if trustedContact.State != nil {
			addr.State = *trustedContact.State
		}
		if trustedContact.PostalCode != nil {
			addr.PostalCode = *trustedContact.PostalCode
		}
		if trustedContact.Country != nil {
			addr.Country = *trustedContact.Country
		}
		form.JSONData.MailingAddress = addr
	}
	return nil
}

// CityForApex returns the name of the city formatted for
// Apex since they can't handle certain things. The known
// issues they have are: [St. vs. Saint]
func CityForApex(city string) string {
	return strings.Replace(city, "St.", "Saint", 1)
}
