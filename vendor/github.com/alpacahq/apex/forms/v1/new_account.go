package v1

type Form interface {
	Title() string
	SetHash(hash string)
	Version() int64
}

type NewAccountForm struct {
	FormID struct {
		Title   string `json:"title"`
		Version int64  `json:"version"`
	} `json:"formId"`
	FormSchemaHash struct {
		Algorithm string `json:"algorithm"`
		Hash      string `json:"hash"`
	} `json:"formSchemaHash"`
	JSONData struct {
		ApplicantSignature struct {
			ESigned string `json:"eSigned"`
		} `json:"applicantSignature"`
		Applicants        []Applicant `json:"applicants"`
		CustomerType      string      `json:"customerType"`
		InvestmentProfile *struct {
			AnnualIncomeUSD *struct {
				Max int64 `json:"max"`
				Min int64 `json:"min"`
			} `json:"annualIncomeUSD,omitempty"`
			FederalTaxBracketPercent int64  `json:"federalTaxBracketPercent,omitempty"`
			InvestmentExperience     string `json:"investmentExperience,omitempty"`
			InvestmentObjective      string `json:"investmentObjective,omitempty"`
			LiquidNetWorthUSD        *struct {
				Max int64 `json:"max"`
				Min int64 `json:"min"`
			} `json:"liquidNetWorthUSD,omitempty"`
			RiskTolerance    string `json:"riskTolerance,omitempty"`
			TotalNetWorthUSD *struct {
				Max int64 `json:"max"`
				Min int64 `json:"min"`
			} `json:"totalNetWorthUSD,omitempty"`
		} `json:"investmentProfile,omitempty"`
		ServiceProfile struct {
			DividendReinvestment      string `json:"dividendReinvestment,omitempty"`
			IssuerDirectCommunication string `json:"issuerDirectCommunication,omitempty"`
		} `json:"serviceProfile"`
		TrustedContact     string `json:"trustedContact,omitempty"`
		SuitabilityProfile *struct {
			LiquidityNeeds string `json:"liquidityNeeds,omitempty"`
			TimeHorizon    string `json:"timeHorizon,omitempty"`
		} `json:"suitabilityProfile,omitempty"`
		TradeAuthorization struct {
			IsTradeAuthorization string `json:"isTradeAuthorization"`
		} `json:"tradeAuthorization"`
	} `json:"jsonData"`
}

type Applicant struct {
	Contact struct {
		EmailAddresses []string      `json:"emailAddresses"`
		HomeAddress    Address       `json:"homeAddress"`
		PhoneNumbers   []PhoneNumber `json:"phoneNumbers"`
	} `json:"contact"`
	Disclosures struct {
		IsAffiliatedExchangeOrFINRA string   `json:"isAffiliatedExchangeOrFINRA"`
		FirmName                    string   `json:"firmName,omitempty"`
		IsControlPerson             string   `json:"isControlPerson"`
		CompanySymbols              []string `json:"companySymbols,omitempty"`
		IsPoliticallyExposed        string   `json:"isPoliticallyExposed"`
		PoliticalExposureDetail     *struct {
			ImmediateFamily       []string `json:"immediateFamily"`
			PoliticalOrganization string   `json:"politicalOrganization"`
		} `json:"politicalExposureDetail,omitempty"`
	} `json:"disclosures"`
	Employment struct {
		Employer         string `json:"employer,omitempty"`
		EmploymentStatus string `json:"employmentStatus"`
		PositionEmployed string `json:"positionEmployed,omitempty"`
	} `json:"employment"`
	Identity struct {
		CitizenshipCountry string `json:"citizenshipCountry"`
		DateOfBirth        string `json:"dateOfBirth"`
		Name               struct {
			Prefix     string `json:"prefix,omitempty"`
			Suffix     string `json:"suffix,omitempty"`
			FamilyName string `json:"familyName"`
			GivenName  string `json:"givenName"`
			LegalName  string `json:"legalName"`
		} `json:"name"`
		SocialSecurityNumber string `json:"socialSecurityNumber"`
		VisaType             string `json:"visaType,omitempty"`
		PermanentResident    string `json:"permanentResident,omitempty"`
		VisaExpirationDate   string `json:"visaExpirationDate,omitempty"`
		BirthCountry         string `json:"birthCountry,omitempty"`
	} `json:"identity"`
	MaritalStatus string `json:"maritalStatus"`
	NumDependents int64  `json:"numDependents"`
}

type PhoneNumber struct {
	PhoneNumber     string `json:"phoneNumber"`
	PhoneNumberType string `json:"phoneNumberType"`
}

func (form *NewAccountForm) SetHash(hash string) {
	form.FormSchemaHash.Hash = hash
}

func (form *NewAccountForm) Title() string {
	return "new_account_form"
}

func (form *NewAccountForm) Version() int64 {
	return 1
}
