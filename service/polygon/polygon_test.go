package polygon

import (
	"testing"
	"time"

	"github.com/alpacahq/gobroker/dbtest"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gopaca/db"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type PolygonTestSuite struct {
	dbtest.Suite
	account     *models.Account
	activeKey   *models.AccessKey
	inactiveKey *models.AccessKey
}

func TestPolygonTestSuite(t *testing.T) {
	suite.Run(t, new(PolygonTestSuite))
}

func (s *PolygonTestSuite) SetupSuite() {
	s.SetupDB()
	amt, _ := decimal.NewFromString("1000000")
	apexAcct := "apca_test"
	name := "Test Trader"
	s.account = &models.Account{
		ApexAccount:        &apexAcct,
		Status:             enum.Active,
		Cash:               amt,
		CashWithdrawable:   amt,
		ApexApprovalStatus: enum.Complete,
		Owners: []models.Owner{
			models.Owner{
				Email:   "trader@test.db",
				Primary: true,
			},
		},
	}

	require.Nil(s.T(), db.DB().Create(s.account).Error)

	street := []string{"2 S B St.", "St. 2"}
	city := "San Mateo"
	state := "CA"
	zip := "94402"
	now := time.Now()

	details := &models.OwnerDetails{
		OwnerID:                  s.account.Owners[0].ID,
		LegalName:                &name,
		StreetAddress:            street,
		City:                     &city,
		State:                    &state,
		PostalCode:               &zip,
		NasdaqAgreementSignedAt:  &now,
		NyseAgreementSignedAt:    &now,
		AccountAgreementSignedAt: &now,
	}

	require.Nil(s.T(), db.DB().Create(details).Error)

	s.activeKey = &models.AccessKey{
		ID:         uuid.Must(uuid.NewV4()).String()[0:20],
		AccountID:  s.account.IDAsUUID(),
		HashSecret: []byte("secret"),
		Secret:     "secret",
		Salt:       "salt",
		Status:     enum.AccessKeyActive,
	}

	require.Nil(s.T(), db.DB().Create(s.activeKey).Error)

	s.inactiveKey = &models.AccessKey{
		ID:         uuid.Must(uuid.NewV4()).String()[0:20],
		AccountID:  s.account.IDAsUUID(),
		HashSecret: []byte("secret"),
		Secret:     "secret",
		Salt:       "salt",
		Status:     enum.AccessKeyDisabled,
	}

	require.Nil(s.T(), db.DB().Create(s.inactiveKey).Error)
}

func (s *PolygonTestSuite) TearDownSuite() {
	s.TeardownDB()
}

func (s *PolygonTestSuite) TestPolygon() {
	srv := Service().WithTx(db.DB())

	// verify random unknown key
	{
		acct, err := srv.VerifyKey(uuid.Must(uuid.NewV4()).String())
		assert.NotNil(s.T(), err)
		assert.Nil(s.T(), acct)
	}

	// verify inactive key
	{
		acct, err := srv.VerifyKey(s.inactiveKey.ID)
		assert.NotNil(s.T(), err)
		assert.Nil(s.T(), acct)
	}

	// verify valid key
	{
		acct, err := srv.VerifyKey(s.activeKey.ID)
		assert.Nil(s.T(), err)
		require.NotNil(s.T(), acct)
		assert.Equal(s.T(), s.account.ID, acct.ID)
	}

	// list with random api keys
	{
		apiKeys := []string{uuid.Must(uuid.NewV4()).String(), uuid.Must(uuid.NewV4()).String()}
		accts, err := srv.List(apiKeys)
		assert.NotNil(s.T(), err)
		assert.Nil(s.T(), accts)
	}

	// list with valid keys
	{
		// note inactive keys should still work in case users
		// regenerate their API key in between when the batch
		// runs and their next auth() call to subscribe to
		// data from polygon
		apiKeys := []string{s.activeKey.ID, s.inactiveKey.ID}
		accts, err := srv.List(apiKeys)
		assert.Nil(s.T(), err)
		require.NotNil(s.T(), accts)
		_, activeOk := accts[s.activeKey.ID]
		_, inactiveOk := accts[s.inactiveKey.ID]
		assert.True(s.T(), activeOk && inactiveOk)
	}
}
