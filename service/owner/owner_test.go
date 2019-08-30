package owner

import (
	"testing"

	"github.com/alpacahq/gobroker/dbtest"
	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gopaca/db"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type OwnerTestSuite struct {
	dbtest.Suite
}

func TestOwnerTestSuite(t *testing.T) {
	suite.Run(t, new(OwnerTestSuite))
}

func (s *OwnerTestSuite) SetupSuite() {
	s.SetupDB()
}

func (s *OwnerTestSuite) TearDownSuite() {
	s.TeardownDB()
}

func (s *OwnerTestSuite) TestOwner() {
	// Account creation

	tx := db.Begin()

	acctSrv := account.Service().WithTx(tx)

	emailX := "test+x@example.com"

	acct, _ := acctSrv.Create(
		emailX,
		uuid.Must(uuid.NewV4()),
	)
	assert.NotNil(s.T(), acct)
	err := tx.Commit().Error
	assert.Nil(s.T(), err)

	srv := Service().WithTx(db.DB())

	emailY := "test+y@example.com"

	owner, err := srv.Patch(acct.IDAsUUID(), map[string]interface{}{"email": emailY})
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), owner)
	assert.Equal(s.T(), emailY, owner.Email)

	// forbidden field
	owner, err = srv.Patch(acct.IDAsUUID(), map[string]interface{}{"id": "some_id"})
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), owner)
}
