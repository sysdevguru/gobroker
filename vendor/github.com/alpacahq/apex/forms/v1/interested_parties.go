package v1

type InterestedPartiesForm struct {
	FormID struct {
		Title   string `json:"title"`
		Version int64  `json:"version"`
	} `json:"formId"`
	FormSchemaHash struct {
		Algorithm string `json:"algorithm"`
		Hash      string `json:"hash"`
	} `json:"formSchemaHash"`
	JSONData struct {
		InterestedParties []InterestedParty `json:"interestedParties"`
	} `json:"jsonData"`
}

type InterestedParty struct {
	MailingAddress *Address `json:"mailingAddress,omitempty"`
	Name           struct {
		CompanyName string `json:"companyName,omitempty"`
	} `json:"name,omitempty"`
	AdditionalName string `json:"additionalName,omitempty"`
}

func (form *InterestedPartiesForm) SetHash(hash string) {
	form.FormSchemaHash.Hash = hash
}

func (form *InterestedPartiesForm) Title() string {
	return "interested_party_request_form"
}

func (form *InterestedPartiesForm) Version() int64 {
	return 1
}
