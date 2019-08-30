package apex

import (
	"fmt"
	"os"
)

var sketchPath = "/sketch/api/v1/"

type GetSketchInvestigationResponse struct {
	ID       *string `json:"id"`
	Status   *string `json:"status"`
	Archived *bool   `json:"archived"`
	History  []struct {
		User        *string     `json:"user"`
		Timestamp   *string     `json:"timestamp"`
		StateChange string      `json:"stateChange"`
		Comment     interface{} `json:"comment"`
		Archived    interface{} `json:"archived"`
	} `json:"history"`
	Request *struct {
		Identity *struct {
			Name *struct {
				Prefix          interface{}   `json:"prefix"`
				GivenName       *string       `json:"givenName"`
				AdditionalNames []interface{} `json:"additionalNames"`
				FamilyName      *string       `json:"familyName"`
				Suffix          interface{}   `json:"suffix"`
			} `json:"name"`
			HomeAddress *struct {
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
			PhoneNumber          interface{} `json:"phoneNumber"`
			SocialSecurityNumber *string     `json:"socialSecurityNumber"`
			CitizenshipCountry   *string     `json:"citizenshipCountry"`
			DateOfBirth          *string     `json:"dateOfBirth"`
		} `json:"identity"`
		IncludeIdentityVerification *bool   `json:"includeIdentityVerification"`
		CorrespondentCode           *string `json:"correspondentCode"`
		Branch                      *string `json:"branch"`
		Account                     *string `json:"account"`
		Source                      *string `json:"source"`
		SourceID                    *string `json:"sourceId"`
	} `json:"request"`
	Result *struct {
		Evaluation *struct {
			EvaluatedState *string  `json:"evaluatedState"`
			DataSources    []string `json:"dataSources"`
			Comment        *string  `json:"comment"`
		} `json:"evaluation"`
		EquifaxResult *struct {
			State   *string     `json:"state"`
			Reasons interface{} `json:"reasons"`
			Results *struct {
				Reject        interface{} `json:"reject"`
				Accept        interface{} `json:"accept"`
				Indeterminate interface{} `json:"indeterminate"`
			} `json:"results"`
			ErrorCode    interface{} `json:"errorCode"`
			ErrorMessage interface{} `json:"errorMessage"`
		} `json:"equifaxResult"`
		DowJonesResult *struct {
			Profiles []interface{} `json:"profiles"`
		} `json:"dowJonesResult"`
		DndbResult *struct {
			Profiles []interface{} `json:"profiles"`
		} `json:"dndbResult"`
	} `json:"result"`
}

type AppealSketchInvestigationParams struct {
	Text string `json:"text"`
	Cip  struct {
		Vendors       []string `json:"vendors"`
		Documentation []string `json:"documentation"`
		SnapIDs       []string `json:"snapIDs"`
	} `json:"cip"`
}

type AppealSketchInvestigationResponse struct {
	ID       *string `json:"id"`
	Status   *string `json:"status"`
	Archived *bool   `json:"archived"`
	History  []struct {
		User        *string `json:"user"`
		Timestamp   *string `json:"timestamp"`
		StateChange *string `json:"stateChange"`
		Comment     *string `json:"comment"`
		Archived    *bool   `json:"archived"`
	} `json:"history"`
	Request *struct {
		Identity *struct {
			Name *struct {
				Prefix          *string  `json:"prefix"`
				GivenName       *string  `json:"givenName"`
				AdditionalNames []string `json:"additionalNames"`
				FamilyName      *string  `json:"familyName"`
				Suffix          *string  `json:"suffix"`
			} `json:"name"`
			HomeAddress *struct {
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
			PhoneNumber          *string `json:"phoneNumber"`
			SocialSecurityNumber *string `json:"socialSecurityNumber"`
			CitizenshipCountry   *string `json:"citizenshipCountry"`
			DateOfBirth          *string `json:"dateOfBirth"`
		} `json:"identity"`
		IncludeIdentityVerification *bool   `json:"includeIdentityVerification"`
		CorrespondentCode           *string `json:"correspondentCode"`
		Branch                      *string `json:"branch"`
		Account                     *string `json:"account"`
		Source                      *string `json:"source"`
		SourceID                    *string `json:"sourceId"`
	} `json:"request"`
	Result *struct {
		Evaluation *struct {
			EvaluatedState *string  `json:"evaluatedState"`
			DataSources    []string `json:"dataSources"`
			Comment        *string  `json:"comment"`
		} `json:"evaluation"`
		EquifaxResult *struct {
			State   *string `json:"state"`
			Results *struct {
			} `json:"results"`
			ErrorCode    *string `json:"errorCode"`
			ErrorMessage *string `json:"errorMessage"`
		} `json:"equifaxResult"`
		DowJonesResult *struct {
			Profiles []struct {
				Name         *string  `json:"name"`
				Summary      []string `json:"summary"`
				ShortSummary *string  `json:"shortSummary"`
				Certainty    *string  `json:"certainty"`
				Reasons      []struct {
					Field    *string `json:"field"`
					Strength *string `json:"strength"`
				} `json:"reasons"`
			} `json:"profiles"`
		} `json:"dowJonesResult"`
		DndbResult *struct {
			Profiles []struct {
				Record *struct {
					GivenName            *string  `json:"givenName"`
					AdditionalNames      []string `json:"additionalNames"`
					FamilyName           *string  `json:"familyName"`
					SocialSecurityNumber *string  `json:"socialSecurityNumber"`
					DateOfBirth          *string  `json:"dateOfBirth"`
					Telephone            *string  `json:"telephone"`
					Email                *string  `json:"email"`
					BusinessName         *string  `json:"businessName"`
					TIN                  *string  `json:"tIN"`
					ID                   *int     `json:"id"`
					Comments             *string  `json:"comments"`
					CreatedBy            *string  `json:"createdBy"`
					CreatedDate          *string  `json:"createdDate"`
				} `json:"record"`
				Certainty *string `json:"certainty"`
				Reasons   []struct {
					Field    *string `json:"field"`
					Strength *string `json:"strength"`
				} `json:"reasons"`
			} `json:"profiles"`
		} `json:"dndbResult"`
	} `json:"result"`
}

func (a *Apex) GetSketchInvestigation(id string) (*GetSketchInvestigationResponse, error) {
	uri := fmt.Sprintf(
		"%v%v/investigations/%v",
		os.Getenv("APEX_URL"),
		sketchPath,
		id,
	)
	m := GetSketchInvestigationResponse{}
	if _, err := a.getJSON(uri, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (a *Apex) AppealSketchInvestigation(id string, action string, params *AppealSketchInvestigationParams) (*AppealSketchInvestigationResponse, error) {
	uri := fmt.Sprintf(
		"%v%v/investigations/%v?action=%v",
		os.Getenv("APEX_URL"),
		sketchPath,
		id,
		action,
	)
	m := AppealSketchInvestigationResponse{}
	if _, err := a.call(uri, "PUT", params, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
