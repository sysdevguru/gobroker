package account

import (
	"strings"
	"testing"

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

type AccountTestSuite struct {
	dbtest.Suite
}

func TestAccountTestSuite(t *testing.T) {
	suite.Run(t, new(AccountTestSuite))
}

func (s *AccountTestSuite) SetupSuite() {
	s.SetupDB()
}

func (s *AccountTestSuite) TearDownSuite() {
	s.TeardownDB()
}

func (s *AccountTestSuite) TestAccount() {
	// Account creation

	var srv AccountService

	tx := db.Begin()

	srv = &accountService{
		tx: tx,
	}
	cid := uuid.Must(uuid.NewV4())
	acct, _ := srv.Create(
		"test@example.com",
		cid,
	)
	assert.NotNil(s.T(), acct)

	err := tx.Commit().Error
	assert.Nil(s.T(), err)

	var rcvAcc models.Account
	require.Nil(s.T(),
		db.DB().
			Where("id = ?", acct.ID).
			Preload("Owners").
			Preload("Owners.Details", "replaced_by IS NULL").
			Find(&rcvAcc).Error)

	status := rcvAcc.Owners[0].Details.MaritalStatus
	dependends := rcvAcc.Owners[0].Details.NumberOfDependents

	assert.Equal(s.T(), *status, models.Single)
	assert.Equal(s.T(), *dependends, uint(0))

	// Account updates
	tx = db.Begin()
	srv = srv.WithTx(tx)

	plan := "PREMIUM"
	acct, _ = srv.Patch(acct.IDAsUUID(), map[string]interface{}{"plan": plan})
	assert.NotNil(s.T(), acct)
	assert.True(s.T(), strings.EqualFold(string(acct.Plan), plan))

	err = tx.Commit().Error
	assert.Nil(s.T(), err)

	// Internal account updates
	tx = db.Begin()
	srv = srv.WithTx(tx)

	cash := decimal.New(10000, 0)
	apexAcct := "apex_test"
	acct, _ = srv.PatchInternal(acct.IDAsUUID(), map[string]interface{}{
		"cash_withdrawable": cash,
		"apex_account":      apexAcct,
	})
	assert.NotNil(s.T(), acct)
	assert.True(s.T(), acct.CashWithdrawable.Equal(cash))
	assert.Equal(s.T(), *acct.ApexAccount, apexAcct)

	err = tx.Commit().Error
	require.Nil(s.T(), err)

	// Get account
	srv = srv.WithTx(db.DB())
	acc, err := srv.GetByID(acct.IDAsUUID())
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), acc.ID, acct.ID)
	assert.Len(s.T(), acc.Owners, 1)

	// Get account by email
	srv = srv.WithTx(db.DB())
	acc, err = srv.GetByEmail("test@example.com")
	assert.Nil(s.T(), err)
	require.NotNil(s.T(), acc)
	assert.Equal(s.T(), acct.ID, acc.ID)
	assert.Len(s.T(), acc.Owners, 1)

	// Get account by apex account
	srv = srv.WithTx(db.DB())
	acc, err = srv.GetByApexAccount(apexAcct)
	require.Nil(s.T(), err)
	require.NotNil(s.T(), acc)
	assert.Equal(s.T(), acct.ID, acc.ID)
	assert.Len(s.T(), acc.Owners, 1)

	// List of accounts
	accQuery := &AccountQuery{
		AccountStatus: []enum.AccountStatus{enum.PaperOnly},
		Page:          1,
		Per:           20,
	}

	srv = srv.WithTx(db.DB())
	accList, meta, err := srv.List(*accQuery)
	require.Nil(s.T(), err)
	require.Equal(s.T(), meta.TotalCount, int64(1))
	require.Len(s.T(), accList, 1)
}
