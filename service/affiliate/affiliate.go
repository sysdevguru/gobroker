package affiliate

import (
	"fmt"
	"strings"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/utils/address"
	"github.com/alpacahq/gopaca/db"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/pariz/gountries"
)

type AffiliateService interface {
	List(accountID uuid.UUID) ([]models.Affiliate, error)
	Create(affiliate *models.Affiliate) (*models.Affiliate, error)
	Patch(accountID uuid.UUID, affiliateID uint, patches map[string]interface{}) (*models.Affiliate, error)
	Delete(accountID uuid.UUID, affiliateID uint) error
	WithTx(tx *gorm.DB) AffiliateService
}

type affiliateService struct {
	AffiliateService
	tx *gorm.DB
}

func Service() AffiliateService {
	return &affiliateService{}
}

func (s *affiliateService) WithTx(tx *gorm.DB) AffiliateService {
	s.tx = tx
	return s
}

func (s *affiliateService) List(accountID uuid.UUID) ([]models.Affiliate, error) {
	a := []models.Affiliate{}
	q := s.tx.Where("account_id = ?", accountID).Find(&a)
	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}
	return a, nil
}

func (s *affiliateService) Create(affiliate *models.Affiliate) (*models.Affiliate, error) {
	query := gountries.New()

	country, err := query.FindCountryByName(strings.ToLower(affiliate.Country))
	if err != nil {
		country, err = query.FindCountryByAlpha(strings.ToUpper(affiliate.Country))
		if err != nil {
			return nil, gberrors.InvalidRequestParam.WithMsg("invalid country")
		}
	}

	affiliate.Country = country.Alpha3

	if affiliate.CompanySymbol != "" {
		affiliate.CompanySymbol = strings.ToUpper(affiliate.CompanySymbol)
	}

	if err := s.tx.Create(affiliate).Error; err != nil {
		return nil, gberrors.InternalServerError.WithError(err)
	}
	return affiliate, nil
}

func (s *affiliateService) Patch(accountID uuid.UUID, affiliateID uint, patches map[string]interface{}) (*models.Affiliate, error) {
	a := &models.Affiliate{}
	q := s.tx.
		Where("id = ? AND account_id = ?", affiliateID, accountID).
		Set("gorm:query_option", db.ForUpdate).
		Find(a)

	if q.RecordNotFound() {
		return nil, gberrors.InternalServerError.WithError(fmt.Errorf("affiliate is not associated with %v", accountID))
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
		case "country":
			query := gountries.New()

			country, err := query.FindCountryByName(strings.ToLower(value.(string)))
			if err != nil {
				country, err = query.FindCountryByAlpha(strings.ToUpper(value.(string)))
				if err != nil {
					return nil, gberrors.InvalidRequestParam.WithMsg("invalid country")
				}
			}

			if err := q.Update(field, country.Alpha3).Error; err != nil {
				return nil, gberrors.InternalServerError.WithError(err)
			}
		case "city":
			fallthrough
		case "state":
			fallthrough
		case "postal_code":
			fallthrough
		case "company_name":
			fallthrough
		case "additional_name":
			if err := q.Update(field, value).Error; err != nil {
				return nil, gberrors.InternalServerError.WithError(err)
			}
		default:
			return nil, gberrors.InvalidRequestParam.WithMsg(fmt.Sprintf("invalid field for affiliate: %v", field))
		}
	}
	return a, nil
}

func (s *affiliateService) Delete(accountID uuid.UUID, affiliateID uint) error {
	a := &models.Affiliate{}
	q := s.tx.Where("id = ? AND account_id = ?", affiliateID, accountID).Find(a)

	if q.RecordNotFound() {
		return gberrors.InternalServerError.WithError(fmt.Errorf("affiliate is not associated with %v", accountID))
	}

	if q.Error != nil {
		return gberrors.InternalServerError.WithError(q.Error)
	}

	return q.Delete(a).Error
}
