package paper

import (
	"testing"

	"github.com/alpacahq/gobroker/dbtest"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/rest/api/controller/entities"
	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gopaca/auth"
	"github.com/alpacahq/gopaca/db"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type PaperTestSuite struct {
	dbtest.Suite
}

type testPaperClient struct {
}

func (c *testPaperClient) CreateAccount(cash decimal.Decimal) (uuid.UUID, error) {
	return uuid.Must(uuid.NewV4()), nil
}

func (c *testPaperClient) CreateAccessKey(accID uuid.UUID) (entities.AccessKeyEntity, error) {
	return entities.AccessKeyEntity{ID: "keyID"}, nil
}

func (c *testPaperClient) ListAccessKeys(accID uuid.UUID) ([]models.AccessKey, error) {
	return []models.AccessKey{}, nil
}

func (c *testPaperClient) DeleteAccessKey(accID uuid.UUID, keyID string) error {
	return nil
}

func (c *testPaperClient) PolygonAuth(apiKeyID string) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (c *testPaperClient) PolygonList(apiKeyIDs []string) (map[string]uuid.UUID, error) {
	return nil, nil
}

func TestPaperTestSuite(t *testing.T) {
	suite.Run(t, new(PaperTestSuite))
}

func (s *PaperTestSuite) SetupSuite() {
	s.SetupDB()
}

func (s *PaperTestSuite) TearDownSuite() {
	s.TeardownDB()
}

func (s *PaperTestSuite) TestCreateAccount() {
	// Account creation
	tx := db.Begin()
	accSrv := account.Service().WithTx(tx)
	acct, _ := accSrv.Create(
		"test+paper@example.com",
		uuid.Must(uuid.NewV4()),
	)
	assert.NotNil(s.T(), acct)
	err := tx.Commit().Error

	svc := paperService{
		tx:       db.DB(),
		papercli: &testPaperClient{},
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
	}

	pAcct, err := svc.Create(acct.IDAsUUID(), decimal.RequireFromString("100"))
	assert.Nil(s.T(), err)
	require.NotNil(s.T(), pAcct)

	pAccts, err := svc.List(acct.IDAsUUID())
	assert.Nil(s.T(), err)
	assert.Len(s.T(), pAccts, 1)

	pAcct, err = svc.GetByID(acct.IDAsUUID(), pAcct.PaperAccountID)
	assert.Nil(s.T(), err)
	require.NotNil(s.T(), pAcct)

	_, err = svc.CreateAccessKey(acct.IDAsUUID(), pAcct.PaperAccountID)
	assert.Nil(s.T(), err)

	_, err = svc.GetAccessKeys(acct.IDAsUUID(), pAcct.PaperAccountID)
	assert.Nil(s.T(), err)

	err = svc.DeleteAccessKey(acct.IDAsUUID(), pAcct.PaperAccountID, "keyID")
	assert.Nil(s.T(), err)

	err = svc.Delete(acct.IDAsUUID(), pAcct.PaperAccountID)
	assert.Nil(s.T(), err)

	pAccts, err = svc.List(acct.IDAsUUID())
	assert.Nil(s.T(), err)
	assert.Empty(s.T(), pAccts)

	_, err = svc.Create(acct.IDAsUUID(), decimal.Zero)
	assert.NotNil(s.T(), err)
}
