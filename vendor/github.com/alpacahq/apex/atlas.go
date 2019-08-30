package apex

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/alpacahq/apex/encryption"
	"github.com/alpacahq/apex/forms"
	"github.com/alpacahq/apex/http"
)

var (
	atlasPath     = "/atlas/api/"
	requiredForms = []string{"new_account_form", "margin_agreement_form"}
)

func (a *Apex) ListForms() []string {
	uri := fmt.Sprintf(
		"%v%v/v1/forms",
		os.Getenv("APEX_URL"),
		atlasPath,
	)
	forms := []string{}
	if _, err := a.getJSON(uri, &forms); err != nil {
		return nil
	}
	return forms
}

func (a *Apex) GetFormVersions(form string) []int {
	uri := fmt.Sprintf(
		"%v%v/v1/forms/%v/versions",
		os.Getenv("APEX_URL"),
		atlasPath,
		form,
	)
	versions := []int{}
	if _, err := a.getJSON(uri, &versions); err != nil {
		return nil
	}
	return versions
}

func (a *Apex) GetForm(form string, version int) map[string]interface{} {
	uri := fmt.Sprintf(
		"%v%v/v1/forms/%v/versions/%v",
		os.Getenv("APEX_URL"),
		atlasPath,
		form,
		version,
	)
	m := map[string]interface{}{}
	if _, err := a.getJSON(uri, &m); err != nil {
		return nil
	}
	return m
}

func (a *Apex) GetFormHash(form string, version int) *string {
	uri := fmt.Sprintf(
		"%v%v/v1/forms/%v/versions/%v",
		os.Getenv("APEX_URL"),
		atlasPath,
		form,
		version,
	)
	var body []byte
	if _, err := a.call(uri, "GET", nil, &body); err != nil {
		return nil
	}
	h := sha256.New()
	h.Write(body)
	hash := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return &hash
}

func (a *Apex) RequiredForms() map[string]interface{} {
	m := make(map[string]interface{}, len(requiredForms))
	for _, form := range requiredForms {
		versions := a.GetFormVersions(form)
		formBody := a.GetForm(form, versions[len(versions)-1])
		if formBody == nil {
			return nil
		}
		m[form] = formBody
	}
	return m
}

type Reason struct {
	Comment string `json:"comment"`
	Body    string `json:"body"`
}

type AccountRequestResponse struct {
	ID                *string  `json:"id"`
	Version           *int     `json:"version"`
	Account           *string  `json:"account"`
	Status            *string  `json:"status"`
	ExternalRequestID *string  `json:"externalRequestId"`
	SketchIDs         []string `json:"sketchIds"`
	Reason            []Reason `json:"reason"`
}

type CancelAccountRequestResponse struct {
	ID                *string  `json:"id"`
	Version           *int     `json:"version"`
	Account           *string  `json:"account"`
	Status            *string  `json:"status"`
	ExternalRequestID *string  `json:"externalRequestId"`
	SketchIds         []string `json:"sketchIds"`
}

type CancelAccountRequestBody struct {
	Comment string `json:"comment"`
}

type AccountOwnerInfoResponse []struct {
	GivenName          *string `json:"givenName"`
	FamilyName         *string `json:"familyName"`
	EntityName         *string `json:"entityName"`
	OwnerType          *string `json:"ownerType"`
	DateOfBirth        *string `json:"dateOfBirth"`
	CitizenshipCountry *string `json:"citizenshipCountry"`
	HomeAddress        *struct {
		StreetAddress []string `json:"streetAddress"`
		City          *string  `json:"city"`
		State         *string  `json:"state"`
		PostalCode    *string  `json:"postalCode"`
		Country       *string  `json:"country"`
	} `json:"homeAddress"`
	MailingAddress *struct {
		StreetAddress []string `json:"streetAddress"`
		City          *string  `json:"city"`
		State         *string  `json:"state"`
		PostalCode    *string  `json:"postalCode"`
		Country       *string  `json:"country"`
	} `json:"mailingAddress"`
	PhoneNumbers []struct {
		PhoneNumber     *string `json:"phoneNumber"`
		PhoneNumberType *string `json:"phoneNumberType"`
		Extension       *string `json:"extension"`
	} `json:"phoneNumbers"`
	EmailAddresses []string `json:"emailAddresses"`
	MaritalStatus  *string  `json:"maritalStatus"`
	NumDependents  *int     `json:"numDependents"`
	Employment     *struct {
		Employer         *string `json:"employer"`
		YearsEmployed    *int    `json:"yearsEmployed"`
		PositionEmployed *string `json:"positionEmployed"`
		BusinessAddress  *struct {
			StreetAddress []string `json:"streetAddress"`
			City          *string  `json:"city"`
			State         *string  `json:"state"`
			PostalCode    *string  `json:"postalCode"`
			Country       *string  `json:"country"`
		} `json:"businessAddress"`
	} `json:"employment"`
	IsDeceased     *bool   `json:"isDeceased"`
	TaxID          *string `json:"taxId"`
	TaxIDType      *string `json:"taxIdType"`
	FullTaxIDToken *string `json:"fullTaxIdToken"`
}

type AccountInfoResponse []struct {
	AccountNumber  *string  `json:"accountNumber"`
	AccountTitle   *string  `json:"accountTitle"`
	AccountNames   []string `json:"accountNames"`
	AccountAddress *struct {
		StreetAddress []string `json:"streetAddress"`
		City          *string  `json:"city"`
		State         *string  `json:"state"`
		PostalCode    *string  `json:"postalCode"`
		Country       *string  `json:"country"`
	} `json:"accountAddress"`
	AccountType  *string `json:"accountType"`
	Last4TIN     *string `json:"last4TIN"`
	OfficeCode   *string `json:"officeCode"`
	RepCode      *string `json:"repCode"`
	PhoneNumbers []struct {
		PhoneNumber     *string `json:"phoneNumber"`
		PhoneNumberType *string `json:"phoneNumberType"`
		Extension       *string `json:"extension"`
	} `json:"phoneNumbers"`
}

type AccountDocumentsResponse []struct {
	DocumentCode      *string `json:"documentCode"`
	DocumentName      *string `json:"documentName"`
	Status            *string `json:"status"`
	StatusUpdatedDate *string `json:"statusUpdatedDate"`
	IsRequired        *bool   `json:"isRequired"`
	IsMissing         *bool   `json:"isMissing"`
	ReceivedDate      *string `json:"receivedDate"`
}

func (arr *AccountRequestResponse) Mock() {
	key := encryption.GenRandomKey(13)
	num := 1
	status := "COMPLETE"
	reqID := "apex_dev_external_id"
	id := "apex_dev_id"

	arr.Account = &key
	arr.Version = &num
	arr.Status = &status
	arr.ExternalRequestID = &reqID
	arr.ID = &id
}

func (a *Apex) PostAccountRequest(sub forms.FormSubmission) (arr *AccountRequestResponse, body []byte, err error) {
	arr = &AccountRequestResponse{}
	if !a.Dev {
		// generate hashes
		for _, form := range sub.Forms {
			hash := a.GetFormHash(form.Title(), int(form.Version()))
			if hash == nil {
				return nil, nil, fmt.Errorf("failed to generate hash for apex account request")
			}
			form.SetHash(*hash)
		}

		uri := fmt.Sprintf(
			"%v%v/v2/account_requests",
			os.Getenv("APEX_URL"),
			atlasPath,
		)

		resp, err := a.call(uri, "POST", sub, arr)

		// get the body regardless for debug purposes
		if resp != nil {
			body, _ = http.GetResponseBody(resp)
		}

		if err != nil {
			return nil, body, err
		}
	} else {
		arr.Mock()
	}

	return arr, body, nil
}

func (a *Apex) GetAccountRequest(requestId string) (*AccountRequestResponse, error) {
	uri := fmt.Sprintf(
		"%v%v/v2/account_requests/%v",
		os.Getenv("APEX_URL"),
		atlasPath,
		requestId,
	)
	arr := &AccountRequestResponse{}
	if _, err := a.getJSON(uri, arr); err != nil {
		return nil, err
	}
	return arr, nil
}

func (a *Apex) CancelAccountRequest(requestId string, comment string) (*CancelAccountRequestResponse, error) {
	uri := fmt.Sprintf(
		"%v%v/v2/account_requests/%v/cancel",
		os.Getenv("APEX_URL"),
		atlasPath,
		requestId,
	)
	reqBody := CancelAccountRequestBody{}
	arr := &CancelAccountRequestResponse{}
	if _, err := a.call(uri, "POST", reqBody, arr); err != nil {
		return nil, err
	}
	return arr, nil
}

type CloseAccountRequestResponse struct {
	ID *string `json:"id"`
}

func (a *Apex) CloseAccountRequest(apexAcctNum string) (*CloseAccountRequestResponse, error) {
	uri := fmt.Sprintf(
		"%v%v/v1/accounts/%v/restrictions/CLOSED_BY_FIRM",
		os.Getenv("APEX_URL"),
		atlasPath,
		apexAcctNum,
	)
	arr := &CloseAccountRequestResponse{}
	if _, err := a.call(uri, "PUT", nil, arr); err != nil {
		return nil, err
	}
	return arr, nil
}

func (a *Apex) AccountInfo(apexAccount string) (*AccountInfoResponse, error) {
	uri := fmt.Sprintf(
		"%v%v/v1/accounts?q=%v",
		os.Getenv("APEX_URL"),
		atlasPath,
		apexAccount,
	)
	info := &AccountInfoResponse{}
	if _, err := a.getJSON(uri, info); err != nil {
		return nil, err
	}
	return info, nil
}

func (a *Apex) AccountOwnerInfo(apexAccount string) (*AccountOwnerInfoResponse, error) {
	uri := fmt.Sprintf(
		"%v%v/v1/accounts/%v/owners",
		os.Getenv("APEX_URL"),
		atlasPath,
		apexAccount,
	)
	info := &AccountOwnerInfoResponse{}
	if _, err := a.getJSON(uri, &info); err != nil {
		return nil, err
	}
	return info, nil
}

func (a *Apex) AccountDocuments(apexAccount string) (*AccountDocumentsResponse, error) {
	uri := fmt.Sprintf(
		"%v%v/v1/accounts/%v/documents",
		os.Getenv("APEX_URL"),
		atlasPath,
		apexAccount,
	)
	docs := &AccountDocumentsResponse{}
	if _, err := a.getJSON(uri, docs); err != nil {
		return nil, err
	}
	return docs, nil
}
