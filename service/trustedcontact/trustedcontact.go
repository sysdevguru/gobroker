package trustedcontact

import (
	"fmt"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/utils/address"
	"github.com/alpacahq/gopaca/db"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
)

type TrustedContactService interface {
	GetByID(accountID uuid.UUID) (*models.TrustedContact, error)
	Create(contact *models.TrustedContact) (*models.TrustedContact, error)
	Upsert(contact *models.TrustedContact) (*models.TrustedContact, error)
	Patch(accountID uuid.UUID, patches map[string]interface{}) (*models.TrustedContact, error)
	Delete(accountID uuid.UUID) error
	WithTx(tx *gorm.DB) TrustedContactService
}

type trustedContactService struct {
	TrustedContactService
	tx *gorm.DB
}

func Service() TrustedContactService {
	return &trustedContactService{}
}

func (s *trustedContactService) WithTx(tx *gorm.DB) TrustedContactService {
	s.tx = tx
	return s
}

func (s *trustedContactService) GetByID(accountID uuid.UUID) (*models.TrustedContact, error) {
	tc := &models.TrustedContact{}
	q := s.tx.Where("account_id = ?", accountID).Find(tc)

	if q.RecordNotFound() {
		return nil, gberrors.NotFound.WithMsg(fmt.Sprintf("trusted contact not found for %v", accountID.String()))
	}

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}

	return tc, nil
}

func (s *trustedContactService) Create(tc *models.TrustedContact) (*models.TrustedContact, error) {
	if err := s.tx.Create(tc).Error; err != nil {
		return nil, gberrors.InternalServerError.WithError(err)
	}
	return tc, nil
}

func (s *trustedContactService) Patch(accountID uuid.UUID, patches map[string]interface{}) (*models.TrustedContact, error) {
	tc := &models.TrustedContact{}
	q := s.tx.
		Where("account_id = ?", accountID).
		Set("gorm:query_option", db.ForUpdate).
		Find(tc)

	if q.RecordNotFound() {
		return nil, gberrors.InternalServerError.WithError(fmt.Errorf("no trusted contact associated with account: %v", accountID))
	}

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}

	for field, value := range patches {
		switch field {
		case "street_address":
			addr, err := address.HandleApiAddress(value)
			if err != nil {
				return nil, gberrors.InvalidRequestParam.WithError(err)
			}
			if err := q.Update(field, addr).Error; err != nil {
				return nil, gberrors.InternalServerError.WithError(err)
			}
		case "phone_number":
			fallthrough
		case "email_address":
			fallthrough
		case "city":
			fallthrough
		case "state":
			fallthrough
		case "postal_code":
			fallthrough
		case "country":
			fallthrough
		case "given_name":
			fallthrough
		case "family_name":
			if err := q.Update(field, value).Error; err != nil {
				return nil, gberrors.InternalServerError.WithError(err)
			}
		default:
			return nil, gberrors.InvalidRequestParam.WithMsg(fmt.Sprintf("invalid field for affiliate: %v", field))
		}
	}
	return tc, nil
}

func (s *trustedContactService) Upsert(updates *models.TrustedContact) (*models.TrustedContact, error) {
	tc := &models.TrustedContact{}
	q := s.tx.
		Where("account_id = ?", updates.AccountID).
		Set("gorm:query_option", db.ForUpdate).
		Find(tc)

	if q.RecordNotFound() {
		return s.Create(updates)
	}

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}

	// record found with no errors, do an update
	tc.EmailAddress = updates.EmailAddress
	tc.PhoneNumber = updates.PhoneNumber
	tc.StreetAddress = updates.StreetAddress
	tc.City = updates.City
	tc.State = updates.State
	tc.PostalCode = updates.PostalCode
	tc.Country = updates.Country
	tc.GivenName = updates.GivenName
	tc.FamilyName = updates.FamilyName

	if err := s.tx.Save(tc).Error; err != nil {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}
	return tc, nil
}

func (s *trustedContactService) Delete(accountID uuid.UUID) error {
	tc := &models.TrustedContact{}
	q := s.tx.Where("account_id = ?", accountID).Find(tc)

	if q.RecordNotFound() {
		return gberrors.InternalServerError.WithError(fmt.Errorf("no trusted contact associated with account: %v", accountID))
	}

	if q.Error != nil {
		return gberrors.InternalServerError.WithError(q.Error)
	}
	return q.Delete(tc).Error
}
