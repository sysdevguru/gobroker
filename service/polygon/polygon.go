package polygon

import (
	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/jinzhu/gorm"
)

type PolygonService interface {
	List(apiKeyIDs []string) (map[string]models.Account, error)
	VerifyKey(apiKeyID string) (*models.Account, error)
	WithTx(tx *gorm.DB) PolygonService
}

type polygonService struct {
	tx *gorm.DB
}

func Service() PolygonService {
	return &polygonService{}
}

func (s *polygonService) WithTx(tx *gorm.DB) PolygonService {
	s.tx = tx
	return s
}

func (s *polygonService) List(apiKeyIDs []string) (map[string]models.Account, error) {
	keys := []models.AccessKey{}

	q := s.tx.
		Where("id IN (?)", apiKeyIDs).
		Preload("Account").
		Preload("Account.Owners").
		Preload("Account.Owners.Details").Find(&keys)

	if q.RecordNotFound() || (q.Error == nil && len(keys) == 0) {
		return nil, gberrors.NotFound
	}

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}

	accts := make(map[string]models.Account, len(keys))

	for _, key := range keys {
		accts[key.ID] = key.Account
	}

	return accts, nil
}

func (s *polygonService) VerifyKey(apiKeyID string) (*models.Account, error) {
	key := models.AccessKey{}

	q := s.tx.Where(
		"id = ? AND status = ?",
		apiKeyID,
		enum.AccessKeyActive).Find(&key)

	if q.RecordNotFound() {
		return nil, gberrors.NotFound.WithMsg("api key id not found")
	}

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}

	acct := models.Account{}

	q = s.tx.
		Where("id = ?", key.AccountID.String()).
		Preload("Owners").
		Preload("Owners.Details", "replaced_by IS NULL").
		First(&acct)

	if q.RecordNotFound() {
		return nil, gberrors.NotFound.WithMsg("account not found for api key id")
	}

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}

	// paper only accounts should not be authorized w/ polygon
	// if acct.Status == enum.PaperOnly ||
	// 	acct.PrimaryOwner().Details.NasdaqAgreementSignedAt == nil ||
	// 	acct.PrimaryOwner().Details.NyseAgreementSignedAt == nil {

	// 	return nil, gberrors.Unauthorized
	// }

	if acct.Status == enum.PaperOnly ||
		acct.PrimaryOwner().Details.AccountAgreementSignedAt == nil {

		return nil, gberrors.Unauthorized
	}

	return &acct, nil
}
