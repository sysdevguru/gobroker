package account

import (
	"fmt"
	"strings"
	"time"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/service/op"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/validate"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	try "gopkg.in/matryer/try.v1"
)

type AccountService interface {
	GetByID(accountID uuid.UUID) (*models.Account, error)
	GetByCognitoID(cognitoID uuid.UUID) (*models.Account, error)
	GetByEmail(email string) (*models.Account, error)
	GetByApexAccount(apexAccount string) (*models.Account, error)
	Create(email string, cognitoID uuid.UUID) (*models.Account, error)
	Patch(accountID uuid.UUID, patches map[string]interface{}) (*models.Account, error)
	PatchInternal(accountID uuid.UUID, patches map[string]interface{}) (*models.Account, error)
	List(query AccountQuery) ([]models.Account, *PaginationMeta, error)
	WithTx(tx *gorm.DB) AccountService
	SetReadOnly()
	SetForUpdate()
}

type accountService struct {
	AccountService
	tx          *gorm.DB
	queryOption *string
}

func Service() AccountService {
	return &accountService{}
}

func (s *accountService) SetReadOnly() {
	forShare := db.ForShare
	s.queryOption = &forShare
}

func (s *accountService) SetForUpdate() {
	forUpdate := db.ForUpdate
	s.queryOption = &forUpdate
}

func (s *accountService) WithTx(tx *gorm.DB) AccountService {
	s.tx = tx
	return s
}

func (s *accountService) GetByID(accountID uuid.UUID) (*models.Account, error) {
	return op.GetAccountByID(s.tx, accountID, s.queryOption)
}

func (s *accountService) GetByCognitoID(cognitoID uuid.UUID) (*models.Account, error) {
	return op.GetAccountByCognitoID(s.tx, cognitoID, s.queryOption)
}

func (s *accountService) GetByEmail(email string) (*models.Account, error) {
	owner := &models.Owner{}
	q := s.tx.Where("email = ?", email).Preload("Accounts").Find(&owner)

	if q.RecordNotFound() {
		return nil, gberrors.NotFound.WithMsg(fmt.Sprintf("account not found for email %v", email))
	}

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}

	acct := &owner.Accounts[0]

	q.Model(acct).Related(&acct.Owners, "Owners")

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}

	return acct, nil
}

func (s *accountService) GetByApexAccount(apexAccount string) (*models.Account, error) {
	return op.GetAccountByApexAccount(s.tx, apexAccount, s.queryOption)
}

func (s *accountService) Create(email string, cognitoID uuid.UUID) (acct *models.Account, err error) {
	if err = validate.Email(email); err != nil {
		return nil, gberrors.InvalidRequestParam.WithMsg(err.Error())
	}

	owner := &models.Owner{
		Email:   email,
		Primary: true,
	}

	if err = s.tx.Create(owner).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			return nil, gberrors.Conflict.WithMsg("duplicate email")
		}
		return nil, gberrors.InternalServerError.WithError(err)
	}

	if cognitoID == uuid.Nil {
		return nil, gberrors.InvalidRequestParam.WithMsg("cognito_id is required")
	}

	cid := cognitoID.String()

	acct = &models.Account{
		Status:    enum.PaperOnly,
		CognitoID: &cid,
	}

	if err = s.tx.Create(acct).Error; err != nil {
		return nil, gberrors.InternalServerError.WithError(err)
	}

	if err = s.tx.Model(acct).Association("Owners").Append(owner).Error; err != nil {
		return nil, gberrors.InternalServerError.WithError(err)
	}

	defaultMaritalStatus := models.Single
	defaultDependends := uint(0)

	od := models.OwnerDetails{
		OwnerID:            owner.ID,
		MaritalStatus:      &defaultMaritalStatus,
		NumberOfDependents: &defaultDependends,
	}

	if err = s.tx.Create(&od).Error; err != nil {
		return nil, gberrors.InternalServerError.WithError(err)
	}

	var code *models.EmailVerificationCode

	if err = try.Do(func(attempt int) (bool, error) {
		code, err = models.NewEmailVerificationCode(owner.ID, owner.Email)
		if err != nil {
			return false, err
		}

		var count int
		if err := s.tx.Model(&models.EmailVerificationCode{}).Where("code = ?", code.Code).Count(&count).Error; err != nil {
			return false, err
		}

		if count > 0 {
			return attempt <= 3, errors.New("duplicated code generated")
		}

		if err := s.tx.Create(code).Error; err != nil {
			return false, err
		}

		return false, nil
	}); err != nil {
		return nil, gberrors.InternalServerError.WithError(err)
	}

	return
}

func (s *accountService) Patch(accountID uuid.UUID, patches map[string]interface{}) (*models.Account, error) {
	acct, err := s.GetByID(accountID)
	if err != nil {
		return nil, err
	}

	for field, value := range patches {
		if !acct.Modifiable(field) {
			return nil, gberrors.InvalidRequestParam.WithMsg(fmt.Sprintf("field %v not found", field))
		}

		if err = s.tx.Model(acct).Update(field, value).Error; err != nil {
			return nil, gberrors.InternalServerError.WithError(err)
		}
	}

	return acct, nil
}

// PatchInternal doesn't protect the values on the account from being modified.
// This method is not to be exposed through the API, for internal usage only.
func (s *accountService) PatchInternal(accountID uuid.UUID, patches map[string]interface{}) (*models.Account, error) {
	acct, err := s.GetByID(accountID)
	if err != nil {
		return nil, err
	}

	for field, value := range patches {
		if err = s.tx.Model(acct).Update(field, value).Error; err != nil {
			return nil, err
		}
	}

	return acct, nil
}

type AccountQuery struct {
	ApexApprovalStatus []enum.ApexApprovalStatus
	AccountStatus      []enum.AccountStatus
	CreatedBefore      *time.Time
	CreatedAfter       *time.Time
	Page               int
	Per                int
	AccountID          *uuid.UUID
	ApexAccount        *string
}

type PaginationMeta struct {
	TotalCount int64 `json:"total_count"`
}

func (s *accountService) List(query AccountQuery) ([]models.Account, *PaginationMeta, error) {
	accounts := []models.Account{}

	q := s.tx

	if query.ApexApprovalStatus != nil {
		q = q.Where("apex_approval_status IN (?)", query.ApexApprovalStatus)
	}

	if query.AccountStatus != nil {
		q = q.Where("status IN (?)", query.AccountStatus)
	}

	if query.CreatedBefore != nil {
		q = q.Where("created_at <= ?", query.CreatedBefore)
	}

	if query.CreatedAfter != nil {
		q = q.Where("created_at >= ?", query.CreatedAfter)
	}

	if query.ApexAccount != nil {
		q = q.Where("apex_account = ?", query.ApexAccount)
	}

	if query.AccountID != nil {
		q = q.Where("id = ?", query.AccountID.String())
	}

	meta := PaginationMeta{}

	if err := q.Model(&models.Account{}).Count(&meta.TotalCount).Error; err != nil {
		return nil, nil, gberrors.InternalServerError.WithError(err)
	}

	offset := (query.Page - 1) * query.Per

	q = q.
		Preload("Owners").
		Preload("Owners.Details", "replaced_by IS NULL").
		Limit(query.Per).
		Offset(offset).
		Order("updated_at DESC")

	if err := q.Find(&accounts).Error; err != nil && err != gorm.ErrRecordNotFound {
		return nil, nil, gberrors.InternalServerError.WithError(err)
	}

	return accounts, &meta, nil
}
