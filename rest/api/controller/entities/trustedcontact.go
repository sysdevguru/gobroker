package entities

import (
	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/utils/address"
	"github.com/gofrs/uuid"
)

type CreateTrustedContactRequest struct {
	EmailAddress  *string         `json:"email_address"`
	PhoneNumber   *string         `json:"phone_number"`
	StreetAddress address.Address `json:"street_address"`
	City          *string         `json:"city"`
	State         *string         `json:"state"`
	PostalCode    *string         `json:"postal_code"`
	Country       *string         `json:"country"`
	GivenName     string          `json:"given_name"`
	FamilyName    string          `json:"family_name"`
}

func (r CreateTrustedContactRequest) Verify() error {
	if r.GivenName == "" || r.FamilyName == "" {
		return gberrors.InvalidRequestParam.WithMsg("valid given_name and family_name required")
	}
	if r.EmailAddress == nil && r.PhoneNumber == nil && r.StreetAddress == nil {
		return gberrors.InvalidRequestParam.WithMsg("at least one contact method required")
	}
	if r.StreetAddress != nil {
		if r.City == nil || r.State == nil || r.PostalCode == nil || r.Country == nil {
			return gberrors.InvalidRequestParam.WithMsg("provided address must be valid")
		}
	}
	return nil
}

func (r CreateTrustedContactRequest) Model(accountID uuid.UUID) *models.TrustedContact {
	return &models.TrustedContact{
		AccountID:     accountID.String(),
		EmailAddress:  r.EmailAddress,
		PhoneNumber:   r.PhoneNumber,
		StreetAddress: r.StreetAddress,
		City:          r.City,
		State:         r.State,
		PostalCode:    r.PostalCode,
		Country:       r.Country,
		GivenName:     r.GivenName,
		FamilyName:    r.FamilyName,
	}
}
