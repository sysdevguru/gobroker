package v1

type MarginForm struct {
	FormID struct {
		Title   string `json:"title"`
		Version int64  `json:"version"`
	} `json:"formId"`
	FormSchemaHash struct {
		Algorithm string `json:"algorithm"`
		Hash      string `json:"hash"`
	} `json:"formSchemaHash"`
	JSONData struct {
		DeliveryDate string `json:"deliveryDateMarginDisclosure"`
		Signature    struct {
			ESigned string `json:"eSigned"`
		} `json:"signature"`
	} `json:"jsonData"`
	ESigned bool `json:"eSigned"`
}

func (form *MarginForm) SetHash(hash string) {
	form.FormSchemaHash.Hash = hash
}

func (form *MarginForm) Title() string {
	return "margin_agreement_form"
}

func (form *MarginForm) Version() int64 {
	return 1
}
