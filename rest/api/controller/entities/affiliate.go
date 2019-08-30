package entities

import (
	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/utils/address"
	"github.com/gofrs/uuid"
)

type CreateAffiliateRequest struct {
	Type            enum.AffiliateType `json:"type"`
	StreetAddress   address.Address    `json:"street_address"`
	City            string             `json:"city"`
	State           string             `json:"state"`
	PostalCode      string             `json:"postal_code"`
	Country         string             `json:"country"`
	CompanyName     string             `json:"company_name"`
	CompanySymbol   string             `json:"company_symbol"`
	ComplianceEmail string             `json:"compliance_email"`
	AdditionalName  *string            `json:"additional_name,omitempty"`
}

func (r CreateAffiliateRequest) Verify() error {
	if r.Type != enum.FinraFirm && r.Type != enum.ControlledFirm {
		return gberrors.InvalidRequestParam.WithMsg("type must be FINRA_FIRM or CONTROLLED_FIRM")
	}
	if r.StreetAddress == nil {
		return gberrors.InvalidRequestParam.WithMsg("street_address is required")
	}

	if len(r.StreetAddress) > 3 {
		return gberrors.InvalidRequestParam.WithMsg("street_address length must be <= 3")
	}

	for _, line := range r.StreetAddress {
		if len(line) > 30 {
			return gberrors.InvalidRequestParam.WithMsg("street_address lines must be <= 30 characters")
		}
	}

	if r.City == "" {
		return gberrors.InvalidRequestParam.WithMsg("city is required")
	}

	if r.State == "" {
		return gberrors.InvalidRequestParam.WithMsg("state is required")
	}

	if r.PostalCode == "" {
		return gberrors.InvalidRequestParam.WithMsg("postal_code is required")
	}

	if r.Country == "" {
		return gberrors.InvalidRequestParam.WithMsg("country is required")
	}

	if r.CompanyName == "" {
		return gberrors.InvalidRequestParam.WithMsg("company_name is required")
	}

	if r.ComplianceEmail == "" {
		return gberrors.InvalidRequestParam.WithMsg("compliance_email is required")
	}

	return nil
}

func (r CreateAffiliateRequest) Model(accountID uuid.UUID) *models.Affiliate {
	streetAddress := address.Address{}

	for _, line := range r.StreetAddress {
		streetAddress = append(streetAddress, line)
	}

	return &models.Affiliate{
		AccountID:       accountID.String(),
		Type:            r.Type,
		StreetAddress:   streetAddress,
		City:            r.City,
		State:           r.State,
		PostalCode:      r.PostalCode,
		Country:         r.Country,
		CompanyName:     r.CompanyName,
		CompanySymbol:   r.CompanySymbol,
		AdditionalName:  r.AdditionalName,
		ComplianceEmail: r.ComplianceEmail,
	}
}
