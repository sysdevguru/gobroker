package owner

import (
	"strings"
	"time"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/service/op"
	"github.com/alpacahq/gobroker/utils"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
)

type OwnerService interface {
	Patch(accountID uuid.UUID, patches map[string]interface{}) (*models.Owner, error)
	WithTx(tx *gorm.DB) OwnerService
}

type ownerService struct {
	tx *gorm.DB
}

func Service() OwnerService {
	return &ownerService{}
}

func WithTx(tx *gorm.DB) OwnerService {
	return &ownerService{tx: tx}
}

func (s *ownerService) WithTx(tx *gorm.DB) OwnerService {
	s.tx = tx
	return s
}

func (s *ownerService) Patch(accountID uuid.UUID, patches map[string]interface{}) (*models.Owner, error) {
	forUpdate := db.ForUpdate

	acct, err := op.GetAccountByID(s.tx, accountID, &forUpdate)
	if err != nil {
		return nil, err
	}

	// do not allow update for an account that has an ongoing investigation
	if !acct.OwnerUpdatable() {
		return nil, gberrors.Forbidden.WithMsg("account processing is ongoing")
	}

	// do no allow update for a new account unless it is at least an hour old
	if !utils.Dev() && acct.ApexAccount != nil && clock.Now().Sub(acct.CreatedAt) < time.Hour {
		return nil, gberrors.Forbidden.WithMsg("too soon for account update")
	}

	owner := acct.PrimaryOwner()

	for key, value := range patches {
		if key == "email" {
			if err = s.tx.
				Model(owner).
				Update(key, strings.ToLower(value.(string))).Error; err != nil {
				return nil, gberrors.InternalServerError.WithError(err)
			}
		} else {
			return nil, gberrors.Forbidden.WithMsg("field is not updatable")
		}
	}

	// update the account status for apex re-submission
	if acct.ApexAccount == nil {
		err = s.tx.
			Model(acct).
			Select("status").
			Update("status", enum.Onboarding).Error
	} else {
		err = s.tx.
			Model(acct).
			Select("status").
			Update("status", enum.AccountUpdated).Error
	}

	return owner, nil
}
