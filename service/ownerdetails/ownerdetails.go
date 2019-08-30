package ownerdetails

import (
	"fmt"
	"strings"
	"time"

	"github.com/alpacahq/gopaca/db"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/service/op"
	"github.com/alpacahq/gobroker/utils"
	"github.com/alpacahq/gobroker/utils/address"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/encryption"
	"github.com/alpacahq/gopaca/env"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/lib/pq"
	"github.com/pariz/gountries"
)

type OwnerDetailsService interface {
	GetPrimaryByAccountID(accountID uuid.UUID) (*models.OwnerDetails, error)
	Patch(accountID uuid.UUID, patches map[string]interface{}) (*models.OwnerDetails, error)
	WithTx(tx *gorm.DB) OwnerDetailsService
}

type ownerDetailsService struct {
	tx *gorm.DB
}

func Service() OwnerDetailsService {
	return &ownerDetailsService{}
}

func (s *ownerDetailsService) WithTx(tx *gorm.DB) OwnerDetailsService {
	s.tx = tx
	return s
}

func (s *ownerDetailsService) GetPrimaryByAccountID(accountID uuid.UUID) (*models.OwnerDetails, error) {
	acct, err := op.GetAccountByID(s.tx, accountID, nil)
	if err != nil {
		return nil, err
	}

	if acct.PrimaryOwner() == nil {
		return nil, gberrors.InternalServerError.WithError(fmt.Errorf("primary owner is not associated with %v", accountID))
	}

	return &acct.PrimaryOwner().Details, nil
}

func (s *ownerDetailsService) Patch(accountID uuid.UUID, patches map[string]interface{}) (*models.OwnerDetails, error) {
	forUpdate := db.ForUpdate

	acct, err := op.GetAccountByID(s.tx, accountID, &forUpdate)
	if err != nil {
		return nil, err
	}

	// the account has transitioned from a paper-only account to starting
	// full brokerage account onboarding - let's update the status
	if acct.Status == enum.PaperOnly {
		if err = s.tx.Model(acct).Update("status", enum.Onboarding).Error; err != nil {
			return nil, gberrors.InternalServerError.WithError(err)
		}
	}

	// do not allow update for an account that has an ongoing investigation
	if !acct.OwnerUpdatable() {
		return nil, gberrors.Forbidden.WithMsg("account processing is ongoing")
	}

	// do no allow update for a new account unless it is at least an hour old
	if !utils.Dev() && acct.ApexAccount != nil && clock.Now().Sub(acct.CreatedAt) < time.Hour {
		return nil, gberrors.Forbidden.WithMsg("too soon for account update")
	}

	// if this is a new account, onboarding for the first time,
	// just update the existing owner details until it is submitted.
	// if the account is being updated post-approval, then we need
	// to create a replacement record.
	if acct.ApexAccount == nil {
		details, err := applyPatches(s.tx, acct, patches)
		if err != nil {
			return nil, err
		}
		acct.PrimaryOwner().Details = *details
	} else {
		details, err := replaceDetails(s.tx, acct, patches)
		if err != nil {
			return nil, err
		}
		acct.PrimaryOwner().Details = *details
	}

	return &acct.PrimaryOwner().Details, nil
}

func applyPatches(tx *gorm.DB, acct *models.Account, patches map[string]interface{}) (*models.OwnerDetails, error) {
	markAccountUpdated := false

	q := tx.Model(&acct.PrimaryOwner().Details)

	for field, value := range patches {
		switch field {
		case "hash_ssn":
			fallthrough
		case "id":
			fallthrough
		case "account_id":
			continue
		case "given_name":
			if len(value.(string)) > 20 {
				return nil, gberrors.InvalidRequestParam.WithMsg("first name must be less than 21 characters")
			}
		case "family_name":
			if len(value.(string)) > 20 {
				return nil, gberrors.InvalidRequestParam.WithMsg("last name must be less than 21 characters")
			}
		case "legal_name":
			if len(value.(string)) > 30 {
				return nil, gberrors.InvalidRequestParam.WithMsg("full name must be less than 31 characters")
			}
		case "ssn":
			if value != nil {
				var hash []byte
				hash, err := encryption.EncryptWithKey([]byte(value.(string)), []byte(env.GetVar("BROKER_SECRET")))
				if err != nil {
					return nil, err
				}
				if err = q.Update("hash_ssn", hash).Error; err != nil {
					return nil, gberrors.InternalServerError.WithError(err)
				}
			} else {
				if err := q.Update("hash_ssn", nil).Error; err != nil {
					return nil, gberrors.InternalServerError.WithError(err)
				}
			}
			markAccountUpdated = true
			continue
		case "margin_agreement_signed":
			if err := q.Update("margin_agreement_signed_at", clock.Now().In(calendar.NY)).Error; err != nil {
				return nil, err
			}
		case "account_agreement_signed":
			if err := q.Update("account_agreement_signed_at", clock.Now().In(calendar.NY)).Error; err != nil {
				return nil, err
			}
		case "street_address":
			if value != nil && len(value.(address.Address)) > 3 {
				return nil, gberrors.InvalidRequestParam.WithMsg("street address too long")
			}
		case "controlling_firms":
			fallthrough
		case "immediate_family_exposed":
			if value != nil {
				arr := pq.StringArray{}
				for _, val := range value.([]interface{}) {
					arr = append(arr, val.(string))
				}
				if err := q.Update(field, arr).Error; err != nil {
					return nil, gberrors.InternalServerError.WithError(err)
				}
				markAccountUpdated = true
				continue
			}
		case "country_of_citizenship":
			fallthrough
		case "country_of_birth":
			if value != nil {
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
			} else {
				if err := q.Update(field, nil).Error; err != nil {
					return nil, gberrors.InternalServerError.WithError(err)
				}
			}
			markAccountUpdated = true
			continue
		}

		if err := q.Update(field, value).Error; err != nil {
			return nil, gberrors.InternalServerError.WithError(err)
		}

		markAccountUpdated = true
	}

	// if this was one that failed before, mark it to retry given the
	// new field values.
	if markAccountUpdated {
		return &acct.PrimaryOwner().Details, q.
			Model(acct).
			Select("status").
			Update("status", enum.Onboarding).Error
	}

	return &acct.PrimaryOwner().Details, nil
}

func replaceDetails(tx *gorm.DB, acct *models.Account, patches map[string]interface{}) (*models.OwnerDetails, error) {
	markAccountUpdated := false

	repl := acct.PrimaryOwner().Details.Replacement()

	for field, value := range patches {
		switch field {
		case "phone_number":
			if value != nil {
				phone := value.(string)
				repl.PhoneNumber = &phone
			}
			markAccountUpdated = true
		case "city":
			if value != nil {
				city := value.(string)
				repl.City = &city
			}
			markAccountUpdated = true
		case "state":
			if value != nil {
				state := value.(string)
				repl.State = &state
			}
			markAccountUpdated = true
		case "postal_code":
			if value != nil {
				postalCode := value.(string)
				repl.PostalCode = &postalCode
			}
			markAccountUpdated = true
		case "unit":
			if value != nil {
				unit := value.(string)
				repl.Unit = &unit
			}
			markAccountUpdated = true
		case "street_address":
			if value != nil && len(value.(address.Address)) > 2 {
				return nil, gberrors.InvalidRequestParam.WithMsg("street address too long")
			}
			repl.StreetAddress = value.(address.Address)
			markAccountUpdated = true
		case "nasdaq_agreement_signed_at":
			if value != nil {
				signed := value.(time.Time)
				repl.NasdaqAgreementSignedAt = &signed
			}
			markAccountUpdated = true
		case "nyse_agreement_signed_at":
			if value != nil {
				signed := value.(time.Time)
				repl.NyseAgreementSignedAt = &signed
			}
			markAccountUpdated = true
		default:
			return nil, gberrors.Forbidden.WithMsg("field cannot be updated")
		}
	}

	if markAccountUpdated {
		// create the replacement
		if err := tx.Create(&repl).Error; err != nil {
			return nil, gberrors.InternalServerError.WithError(err)
		}

		m := map[string]interface{}{
			"replaced_at": &repl.CreatedAt,
			"replaced_by": &repl.ID,
		}

		// update the original
		if err := tx.Model(&acct.PrimaryOwner().Details).Updates(m).Error; err != nil {
			return nil, gberrors.InternalServerError.WithError(err)
		}
		return &repl, tx.
			Model(acct).
			Select("status").
			Update("status", enum.AccountUpdated).Error
	}

	return nil, gberrors.InvalidRequestParam.WithMsg("invalid patch")
}
