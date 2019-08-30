package accesskey

import (
	"testing"

	"github.com/alpacahq/gobroker/dbtest"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gobroker/service/tradeaccount"
	"github.com/alpacahq/gopaca/auth"
	"github.com/alpacahq/gopaca/db"
	"github.com/go-redis/cache"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type AccessKeyTestSuite struct {
	dbtest.Suite
}

func TestAccessKeyTestSuite(t *testing.T) {
	suite.Run(t, new(AccessKeyTestSuite))
}

func (s *AccessKeyTestSuite) SetupSuite() {
	s.SetupDB()
}

func (s *AccessKeyTestSuite) TearDownSuite() {
	s.TeardownDB()
}

func (s *AccessKeyTestSuite) TestCreateAuth() {
	tx := db.Begin()

	acctSrv := account.Service().WithTx(tx)
	acct, err := acctSrv.Create(
		"test+create-auth@example.com",
		uuid.Must(uuid.NewV4()),
	)
	assert.Nil(s.T(), err)

	assert.Nil(s.T(), tx.Commit().Error)

	tx = db.Begin()
	srv := &accessKeyService{
		accService: tradeaccount.Service(),
		cacheVerify: func(id, secret string) (*models.AccessKey, bool, error) {
			return &models.AccessKey{
				ID:        id,
				Secret:    secret,
				Status:    enum.AccessKeyActive,
				AccountID: acct.IDAsUUID(),
			}, true, nil
		},
		cacheStore: func(info *auth.AuthInfo) error {
			return nil
		},
		cacheDelete: func(id string) error {
			return nil
		},
		tx: tx,
	}

	accountID := acct.IDAsUUID()

	key, err := srv.Create(accountID, enum.LiveAccount)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), key.AccountID, accountID)

	assert.Nil(s.T(), tx.Commit().Error)

	var nkey models.AccessKey
	tx = db.Begin()
	srv = &accessKeyService{
		accService: tradeaccount.Service(),
		cacheVerify: func(id, secret string) (*models.AccessKey, bool, error) {
			return nil, false, cache.ErrCacheMiss
		},
		cacheStore: func(info *auth.AuthInfo) error {
			return nil
		},
		cacheDelete: func(id string) error {
			return nil
		},
		tx: tx,
	}

	assert.Nil(s.T(), tx.Where("id = ?", key.ID).Preload("Account").Find(&nkey).Error)
	assert.Equal(s.T(), nkey.AccountID, accountID)

	// Verify
	vkey, err := srv.Verify(key.ID, key.Secret)
	assert.Equal(s.T(), vkey.AccountID, accountID)
	_, err = srv.Verify(key.ID, "inv")

	assert.NotNil(s.T(), err)
	tx.Commit()
	tx = db.Begin()
	srv = &accessKeyService{
		accService: tradeaccount.Service(),
		cacheVerify: func(id, secret string) (*models.AccessKey, bool, error) {
			return nil, false, nil
		},
		cacheStore: func(info *auth.AuthInfo) error {
			return nil
		},
		cacheDelete: func(id string) error {
			return nil
		},
		tx: tx,
	}

	// Sync (some keys)
	assert.Nil(s.T(), srv.Sync(false))

	// Disable
	dkey, err := srv.Disable(accountID, key.ID)
	require.Nil(s.T(), err)
	assert.Equal(s.T(), dkey.Status, enum.AccessKeyDisabled)
	vkey, err = srv.Verify(key.ID, key.Secret)
	assert.NotNil(s.T(), err)
	tx.Commit()

	// Sync (no keys)
	assert.Nil(s.T(), srv.Sync(false))
}
