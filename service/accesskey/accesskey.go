package accesskey

import (
	"bytes"
	"fmt"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/service/tradeaccount"
	"github.com/alpacahq/gomarkets/sources"
	"github.com/alpacahq/gopaca/auth"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/encryption"
	"github.com/alpacahq/gopaca/env"
	"github.com/alpacahq/gopaca/log"
	"github.com/go-redis/cache"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

type AccessKeyService interface {
	WithCache() AccessKeyService
	WithTx(*gorm.DB) AccessKeyService
	Disable(accountID uuid.UUID, accessKenID string) (*models.AccessKey, error)
	Verify(accessKeyID, accessKeySecret string) (*models.AccessKey, error)
	Create(accountID uuid.UUID, version enum.AccountType) (*models.AccessKey, error)
	List(accountID uuid.UUID) ([]*models.AccessKey, error)
	Sync(paper bool) error
}

type accessKeyService struct {
	AccessKeyService
	tx          *gorm.DB
	cache       bool
	accService  tradeaccount.TradeAccountService
	cacheVerify func(string, string) (*models.AccessKey, bool, error)
	cacheStore  func(*auth.AuthInfo) error
	cacheDelete func(string) error
}

func Service(accService tradeaccount.TradeAccountService) AccessKeyService {
	s := &accessKeyService{accService: accService}

	s.cacheVerify = func(id, secret string) (*models.AccessKey, bool, error) {
		if !s.cache {
			return nil, false, nil
		}

		pl, err := auth.Get(id)
		if err != nil {
			return nil, false, err
		}

		hashed, err := encryption.SaltEncrypt([]byte(secret), pl.Salt)
		if err != nil {
			return nil, false, err
		}

		if bytes.Equal(hashed, pl.HashedSecret) {
			return &models.AccessKey{
				ID:          id,
				Status:      enum.AccessKeyStatus(pl.Status),
				HashSecret:  pl.HashedSecret,
				Salt:        string(pl.Salt),
				AccountID:   pl.AccountID,
				DataSources: pl.DataSources,
			}, true, nil
		}

		return nil, false, nil
	}

	s.cacheStore = func(info *auth.AuthInfo) error {
		if !s.cache {
			return nil
		}

		return auth.Set(info)
	}

	s.cacheDelete = func(id string) error {
		if !s.cache {
			return nil
		}

		return auth.Delete(id)
	}

	return s
}

func getKey(tx *gorm.DB, id string) (*models.AccessKey, error) {
	var akey models.AccessKey

	q := tx.
		Where("id = ?", id).
		Preload("Account").
		Preload("Account.Owners").
		Preload("Account.Owners.Details").
		First(&akey)

	if q.RecordNotFound() {
		return nil, gberrors.NotFound
	}

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}
	return &akey, nil
}

func (s *accessKeyService) WithCache() AccessKeyService {
	s.cache = true
	return s
}

func (s *accessKeyService) WithTx(tx *gorm.DB) AccessKeyService {
	s.tx = tx
	return s
}

func (s *accessKeyService) Disable(accountID uuid.UUID, accessKeyID string) (*models.AccessKey, error) {
	var aKey models.AccessKey

	q := s.tx.
		Where(
			"id = ? AND account_id = ?",
			accessKeyID, accountID.String()).
		First(&aKey)

	if q.RecordNotFound() {
		return nil, gberrors.NotFound.WithMsg(fmt.Sprintf("key id = %s not found", accessKeyID))
	}

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}

	if aKey.Status != enum.AccessKeyActive {
		return nil, gberrors.InvalidRequestParam.WithMsg(fmt.Sprintf("access key %v is disabled", accessKeyID))
	}

	q = s.tx.Model(&aKey).Update("status", enum.AccessKeyDisabled)

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}

	// remove from auth cache
	if err := s.cacheDelete(aKey.ID); err != nil && err != cache.ErrCacheMiss {
		return nil, gberrors.InternalServerError.WithError(err)
	}

	return &aKey, nil
}

func (s *accessKeyService) Verify(accessKeyID string, accessKeySecret string) (*models.AccessKey, error) {
	// attempt to verify with cached value
	if aKey, verified, err := s.cacheVerify(accessKeyID, accessKeySecret); err == nil && verified {
		return aKey, nil
	}

	aKey, err := getKey(s.tx, accessKeyID)

	if err != nil {
		// For security reason, treat not found error as unauthorized on verification
		if err == gberrors.NotFound {
			return nil, gberrors.Unauthorized.WithMsg("access key not found")
		}
		return nil, err
	}

	if aKey.Status != enum.AccessKeyActive {
		return nil, gberrors.Unauthorized
	}

	if err = aKey.Verify(accessKeySecret); err != nil {
		return nil, err
	}

	if err = s.store(aKey); err != nil {
		return nil, gberrors.InternalServerError.WithError(err)
	}

	return aKey, nil
}

// Create AccessKey for an account. Right now, an access key is associated with an account.
func (s *accessKeyService) Create(accountID uuid.UUID, version enum.AccountType) (*models.AccessKey, error) {

	acc, err := s.accService.WithTx(s.tx).ForUpdate().GetByID(accountID)
	if err != nil {
		return nil, err
	}

	var keys []models.AccessKey

	if err := s.tx.
		Where("account_id = ? AND status = ?",
			acc.ID, enum.AccessKeyActive).
		Find(&keys).Error; err != nil {

		return nil, errors.Wrap(err, "failed to load access keys")
	}

	if len(keys) > 1 {
		return nil, gberrors.InvalidRequestParam.WithMsg(
			"currently account can have only 2 access keys")
	}

	key, err := models.NewAccessKey(accountID, version)
	if err != nil {
		return nil, gberrors.InternalServerError.WithError(
			fmt.Errorf("failed to initialize access key %v", err.Error()))
	}

	if err = s.tx.Create(key).Error; err != nil {
		return nil, gberrors.InternalServerError.WithError(
			fmt.Errorf("failed to create access key %v", err.Error()))
	}

	q := s.tx.Where("id = ?", key.AccountID)

	if version != enum.PaperAccount {
		q = q.Preload("Owners").Preload("Owners.Details", "replaced_by IS NULL")
	}

	if err = q.First(&key.Account).Error; err != nil {
		return nil, gberrors.InternalServerError.WithError(
			fmt.Errorf("failed to create access key %v", err.Error()))
	}

	if err = s.store(key); err != nil {
		return nil, gberrors.InternalServerError.WithError(err)
	}

	return key, nil
}

func (s *accessKeyService) List(accountID uuid.UUID) ([]*models.AccessKey, error) {
	var keys []*models.AccessKey

	q := s.tx.
		Where("account_id = ? AND status = ?",
			accountID.String(),
			enum.AccessKeyActive).
		Find(&keys)

	if q.RecordNotFound() || len(keys) == 0 {
		return make([]*models.AccessKey, 0), nil
	}

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}

	return keys, nil
}

func (s *accessKeyService) Sync(paper bool) error {
	var keys []*models.AccessKey

	// live
	q := s.tx.
		Where("status = ?", enum.AccessKeyActive).
		Preload("Account").
		Preload("Account.Owners").
		Preload("Account.Owners.Details").
		Find(&keys)

	if len(keys) == 0 {
		goto Paper
	}

	if q.Error != nil {
		return gberrors.InternalServerError.WithError(q.Error)
	}

	for _, key := range keys {
		if err := s.store(key); err != nil {
			return gberrors.InternalServerError.WithError(err)
		}
	}

Paper:
	if paper {
		opts := map[string]string{"PGDATABASE": env.GetVar("PAPER_DB")}

		database, err := db.NewDB(opts)
		if err != nil {
			log.Fatal("database error", "error", err)
		}

		defer database.Close()

		q = database.
			Where("status = ?", enum.AccessKeyActive).
			Preload("Account").
			Find(&keys)

		if len(keys) == 0 {
			return nil
		}

		if q.Error != nil {
			return gberrors.InternalServerError.WithError(q.Error)
		}

		paperAccountIDs := make([]string, len(keys))
		keysByAccount := make(map[uuid.UUID]*models.AccessKey, len(keys))

		for i, key := range keys {
			paperAccountIDs[i] = key.Account.ID
			keysByAccount[key.Account.IDAsUUID()] = key
		}

		paperAccounts := []models.PaperAccount{}

		q = s.tx.
			Where("paper_account_id in (?)", paperAccountIDs).
			Preload("Account").
			Preload("Account.Owners").
			Preload("Account.Owners.Details").
			Find(&paperAccounts)

		if len(paperAccounts) == 0 {
			return nil
		}

		if q.Error != nil {
			return gberrors.InternalServerError.WithError(q.Error)
		}

		for _, paperAcct := range paperAccounts {
			if key, ok := keysByAccount[paperAcct.PaperAccountID]; ok {
				key.Account = paperAcct.Account

				if err = s.store(key); err != nil {
					return gberrors.InternalServerError.WithError(err)
				}
			}
		}
	}

	return nil
}

func (s *accessKeyService) store(key *models.AccessKey) error {
	info := &auth.AuthInfo{
		ID: key.ID,
		Payload: auth.Payload{
			AccountID:    key.AccountID,
			Status:       string(key.Status),
			HashedSecret: key.HashSecret,
			Salt:         []byte(key.Salt),
			DataSources:  []string{string(sources.IEX)},
		},
	}

	if owner := key.Account.PrimaryOwner(); owner != nil {
		info.Payload.DataSources = owner.Details.DataSources()
	}

	return s.cacheStore(info)
}
