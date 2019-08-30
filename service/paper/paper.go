package paper

import (
	"bytes"

	"github.com/alpacahq/gobroker/external/paper"
	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/rest/api/controller/entities"
	"github.com/alpacahq/gobroker/service/assetcache"
	"github.com/alpacahq/gobroker/service/position"
	"github.com/alpacahq/gobroker/service/tradeaccount"
	"github.com/alpacahq/gomarkets/sources"
	"github.com/alpacahq/gopaca/auth"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/encryption"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

type ppcli interface {
	CreateAccount(decimal.Decimal) (uuid.UUID, error)
	CreateAccessKey(uuid.UUID) (entities.AccessKeyEntity, error)
	ListAccessKeys(uuid.UUID) ([]models.AccessKey, error)
	DeleteAccessKey(uuid.UUID, string) error
	PolygonAuth(apiKeyID string) (uuid.UUID, error)
	PolygonList(apiKeyIDs []string) (map[string]uuid.UUID, error)
}

type PaperService interface {
	Create(accountID uuid.UUID, cash decimal.Decimal) (*models.PaperAccount, error)
	GetByID(accountID, paperAccountID uuid.UUID) (*models.PaperAccount, error)
	Delete(accountID, paperAccountID uuid.UUID) error
	List(accountID uuid.UUID) ([]models.PaperAccount, error)
	CreateAccessKey(accountID uuid.UUID, paperAccountID uuid.UUID) (*models.AccessKey, error)
	GetAccessKeys(accountID uuid.UUID, paperAccountID uuid.UUID) ([]models.AccessKey, error)
	DeleteAccessKey(accountID uuid.UUID, paperAccountID uuid.UUID, keyID string) error
	ListKeysForPolygon(apiKeyIDs []string) (map[string]models.Account, error)
	VerifyKeyForPolygon(apiKeyID string) (*models.Account, error)
	WithTx(tx *gorm.DB) PaperService
}

type paperService struct {
	tx          *gorm.DB
	papercli    ppcli
	cacheVerify func(string, string) (*models.AccessKey, bool, error)
	cacheStore  func(*auth.AuthInfo) error
}

func Service() PaperService {
	return &paperService{
		papercli: paper.NewClient(),
		cacheVerify: func(id, secret string) (*models.AccessKey, bool, error) {
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
		},
		cacheStore: func(info *auth.AuthInfo) error {
			return auth.Set(info)
		},
	}
}

func (s *paperService) WithTx(tx *gorm.DB) PaperService {
	s.tx = tx
	return s
}

func (s *paperService) Create(accountID uuid.UUID, cash decimal.Decimal) (*models.PaperAccount, error) {
	if cash.Equal(decimal.Zero) {
		srv := tradeaccount.Service().WithTx(s.tx)
		balances, err := srv.GetBalancesByID(accountID, clock.Now())
		if err != nil {
			return nil, err
		}

		cash = balances.Cash

		pSrv := position.Service(assetcache.GetAssetCache()).WithTx(s.tx)

		positions, err := pSrv.List(accountID)
		if err != nil {
			return nil, err
		}

		for _, p := range positions {
			cash = cash.Add(p.MarketValue)
		}

		if cash.Equal(decimal.Zero) {
			return nil, gberrors.Forbidden.WithMsg("account must have funds to create a paper account")
		}
	}

	paperAccID, err := s.papercli.CreateAccount(cash)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create paper trade account")
	}

	pAcct := models.PaperAccount{
		AccountID:      accountID,
		PaperAccountID: paperAccID,
	}

	if err := s.tx.Create(&pAcct).Error; err != nil {
		return nil, errors.Wrap(err, "failed to store paper trade account id")
	}

	return &pAcct, nil
}

func (s *paperService) GetByID(accountID, paperAccountID uuid.UUID) (*models.PaperAccount, error) {
	acct := &models.PaperAccount{}

	q := s.tx.
		Where(
			"account_id = ? AND paper_account_id = ?",
			accountID.String(),
			paperAccountID.String()).
		First(&acct)

	if q.RecordNotFound() {
		return nil, gberrors.NotFound.WithMsg("paper account not found")
	}

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(q.Error)
	}

	return acct, nil
}

func (s *paperService) Delete(accountID, paperAccountID uuid.UUID) error {
	q := s.tx.
		Where("account_id = ? AND paper_account_id = ?",
			accountID.String(),
			paperAccountID.String()).
		Delete(&models.PaperAccount{})

	if q.RecordNotFound() {
		return gberrors.NotFound.WithMsg("paper account not found")
	}

	if q.Error != nil {
		return gberrors.InternalServerError.WithError(q.Error)
	}

	return nil
}

func (s *paperService) List(accountID uuid.UUID) ([]models.PaperAccount, error) {
	var accounts []models.PaperAccount
	if err := s.tx.Where("account_id = ?", accountID).Find(&accounts).Error; err != nil {
		return nil, errors.Wrap(err, "failed to query paper accounts")
	}
	return accounts, nil
}

func (s *paperService) CreateAccessKey(accountID uuid.UUID, paperAccountID uuid.UUID) (*models.AccessKey, error) {
	paperAcct, err := s.getPaperAccount(accountID, paperAccountID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find paper account")
	} else if paperAcct == nil {
		return nil, gberrors.NotFound.WithMsg("paper account not found")
	}

	keyEntity, err := s.papercli.CreateAccessKey(paperAccountID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create access key")
	}

	key := keyEntity.Model()

	key.Account = paperAcct.Account

	if err = s.store(key); err != nil {
		return nil, gberrors.InternalServerError.WithError(err)
	}

	return key, nil
}

func (s *paperService) GetAccessKeys(accountID uuid.UUID, paperAccountID uuid.UUID) ([]models.AccessKey, error) {
	if acc, err := s.getPaperAccount(accountID, paperAccountID); err != nil {
		return nil, errors.Wrap(err, "failed to find paper account")
	} else if acc == nil {
		return nil, gberrors.NotFound.WithMsg("paper account not found")
	}

	keys, err := s.papercli.ListAccessKeys(paperAccountID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list access keys")
	}

	return keys, nil
}

func (s *paperService) DeleteAccessKey(accountID uuid.UUID, paperAccountID uuid.UUID, keyID string) (err error) {
	if paperAcct, err := s.getPaperAccount(accountID, paperAccountID); err != nil {
		return errors.Wrap(err, "failed to find paper account")
	} else if paperAcct == nil {
		return gberrors.NotFound.WithMsg("paper account not found")
	}

	if err = s.papercli.DeleteAccessKey(paperAccountID, keyID); err != nil {
		return
	}

	return
}

func (s *paperService) getPaperAccount(accountID uuid.UUID, paperAccountID uuid.UUID) (*models.PaperAccount, error) {
	var acc models.PaperAccount

	q := s.tx.
		Where(
			"account_id = ? AND paper_account_id = ?",
			accountID, paperAccountID).
		Preload("Account").
		Preload("Account.Owners").
		Preload("Account.Owners.Details").
		First(&acc)

	if q.RecordNotFound() {
		return nil, nil
	}

	if q.Error != nil {
		return nil, q.Error
	}

	return &acc, nil
}

func (s *paperService) VerifyKeyForPolygon(apiKeyID string) (*models.Account, error) {
	id, err := s.papercli.PolygonAuth(apiKeyID)
	if err != nil {
		return nil, err
	}

	paperAccount := &models.PaperAccount{}

	q := s.tx.Where("paper_account_id = ?", id.String()).First(paperAccount)

	if q.RecordNotFound() {
		return nil, gberrors.NotFound.WithMsg("paper account not found for key")
	}

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(err)
	}

	acct := &models.Account{}

	q = s.tx.
		Where("id = ?", paperAccount.AccountID.String()).
		Preload("Owners").
		Preload("Owners.Details", "replaced_by IS NULL").
		First(acct)

	if q.RecordNotFound() {
		return nil, gberrors.NotFound.WithMsg("account not found for key")
	}

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(err)
	}

	if acct.Status == enum.PaperOnly ||
		acct.PrimaryOwner().Details.AccountAgreementSignedAt == nil {

		return nil, gberrors.Unauthorized
	}

	return acct, nil
}

func (s *paperService) ListKeysForPolygon(apiKeyIDs []string) (map[string]models.Account, error) {
	ids, err := s.papercli.PolygonList(apiKeyIDs)
	if err != nil {
		return nil, err
	}

	// find paper accounts
	paperIDs := []string{}

	for _, id := range ids {
		paperIDs = append(paperIDs, id.String())
	}

	paperAccounts := []models.PaperAccount{}

	q := s.tx.Where("paper_account_id in (?)", paperIDs).Find(&paperAccounts)

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(err)
	}

	if len(paperAccounts) == 0 {
		return map[string]models.Account{}, nil
	}

	// find real accounts
	accountIDs := make([]string, len(paperAccounts))

	for i, acct := range paperAccounts {
		accountIDs[i] = acct.AccountID.String()
	}

	accounts := []models.Account{}

	q = s.tx.Where("id in (?)", accountIDs).Find(&accounts)

	if q.Error != nil {
		return nil, gberrors.InternalServerError.WithError(err)
	}

	if len(accounts) == 0 {
		return map[string]models.Account{}, nil
	}

	m := make(map[string]models.Account, len(accounts))

	for key, paperID := range ids {
		for _, paperAcct := range paperAccounts {
			if paperAcct.PaperAccountID == paperID {
				for _, acct := range accounts {
					if paperAcct.AccountID == acct.IDAsUUID() {
						m[key] = acct
					}
				}
			}
		}
	}

	return m, nil
}

func (s *paperService) store(key *models.AccessKey) error {
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
