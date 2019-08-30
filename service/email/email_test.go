package email

import (
	"testing"
	"time"

	"github.com/alpacahq/gobroker/dbtest"
	"github.com/alpacahq/gobroker/mailer"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gopaca/db"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type EmailTestSuite struct {
	dbtest.Suite
	account    *models.Account
	marginCall *models.MarginCall
}

func TestEmailTestSuite(t *testing.T) {
	suite.Run(t, new(EmailTestSuite))
}

func (s *EmailTestSuite) SetupSuite() {
	s.SetupDB()
	apexAcct := "apca_test"
	givenName := "Test"
	s.account = &models.Account{
		ApexAccount:        &apexAcct,
		Status:             enum.Active,
		ApexApprovalStatus: enum.Complete,
		Owners: []models.Owner{
			models.Owner{
				Email:   "trader@test.db",
				Primary: true,
			},
		},
	}

	if err := db.DB().Create(s.account).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}

	details := &models.OwnerDetails{
		OwnerID:   s.account.Owners[0].ID,
		GivenName: &givenName,
	}

	if err := db.DB().Create(details).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}

	require.Nil(s.T(), db.DB().Model(&s.account.Owners[0]).Related(&s.account.Owners[0].Details).Error)

	s.marginCall = &models.MarginCall{
		AccountID:  s.account.ID,
		CallType:   enum.EquityMaintenance,
		CallAmount: decimal.New(1000, 0),
		DueDate:    "2019-01-01",
		TradeDate:  "2018-05-06",
	}

	if err := db.DB().Create(s.marginCall).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
}

func (s *EmailTestSuite) TearDownSuite() {
	s.TeardownDB()
}

func (s *EmailTestSuite) TestCreate() {
	srv := emailService{
		tx: db.DB(),
		sendMarginCall: func(
			acct, givenName, email string,
			dueDate time.Time,
			callAmount decimal.Decimal,
			deliverAt *time.Time) error {
			return nil
		},
		sendPatternDayTrader: func(
			acct, givenName, email string,
			dueDate time.Time,
			callAmount decimal.Decimal,
			deliverAt *time.Time) error {
			return nil
		},
	}

	// margin call
	assert.Nil(s.T(), srv.Create(s.account, mailer.MarginCall, nil))

	// pdt
	assert.Nil(s.T(), srv.Create(s.account, mailer.PatternDayTrader, nil))
}
