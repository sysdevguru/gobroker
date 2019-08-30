package v1

type TrustedContactForm struct {
	FormID struct {
		Title   string `json:"title"`
		Version int64  `json:"version"`
	} `json:"formId"`
	FormSchemaHash struct {
		Algorithm string `json:"algorithm"`
		Hash      string `json:"hash"`
	} `json:"formSchemaHash"`
	JSONData struct {
		EmailAddress   string      `json:"emailAddress,omitempty"`
		PhoneNumber    PhoneNumber `json:"phoneNumber,omitempty"`
		MailingAddress *Address    `json:"mailingAddress,omitempty"`
		GivenName      string      `json:"givenName"`
		FamilyName     string      `json:"familyName"`
	} `json:"jsonData"`
}

type Address struct {
	Country       string   `json:"country,omitempty"`
	StreetAddress []string `json:"streetAddress,omitempty"`
	City          string   `json:"city,omitempty"`
	PostalCode    string   `json:"postalCode,omitempty"`
	State         string   `json:"state,omitempty"`
}

func (form *TrustedContactForm) SetHash(hash string) {
	form.FormSchemaHash.Hash = hash
}

func (form *TrustedContactForm) Title() string {
	return "trusted_contact_form"
}

func (form *TrustedContactForm) Version() int64 {
	return 1
}
