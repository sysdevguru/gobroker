package tradeaccount

import (
	"testing"

	models "github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gobroker/utils/testdb"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TradeAccountSuite struct {
	suite.Suite
	account *models.Account
}

func TestTradeAccount(t *testing.T) {
	suite.Run(t, new(TradeAccountSuite))
}

func (s *TradeAccountSuite) SetupSuite() {
	testdb.SetUp()

	tx := db.Begin()
	srv := account.Service().WithTx(tx)

	acct, err := srv.Create(
		"test+tradeaccount@example.com",
		uuid.Must(uuid.NewV4()),
	)

	require.Nil(s.T(), err)
	assert.NotNil(s.T(), acct)
	assert.Nil(s.T(), tx.Commit().Error)

	s.account = acct
}

func (s *TradeAccountSuite) TeardownSuite() {
	testdb.TearDown()
}

func (s *TradeAccountSuite) TestGetByAccount() {
	svc := Service().WithTx(db.DB())

	{
		a, err := svc.GetByID(s.account.IDAsUUID())
		assert.Nil(s.T(), err)
		assert.NotNil(s.T(), a)
		assert.Equal(s.T(), a.ID, s.account.ID)
	}

}

func (s *TradeAccountSuite) TestmarkPatternDayTrader() {
	svc := Service().WithTx(db.DB())

	{
		ta, err := s.account.ToTradeAccount()
		assert.Nil(s.T(), err)

		err = svc.MarkPatternDayTrader(ta)
		assert.Nil(s.T(), err)

		a, err := svc.GetByID(s.account.IDAsUUID())
		assert.Nil(s.T(), err)
		assert.True(s.T(), a.PatternDayTrader)

		assert.Equal(
			s.T(),
			a.MarkedPatternDayTraderAt.String(),
			clock.Now().In(calendar.NY).Format("2006-01-02"),
		)
	}
}

func (s *TradeAccountSuite) TestConfigure() {
	svc := Service().WithTx(db.DB())

	{
		ta, err := s.account.ToTradeAccount()

		suspend := true
		updated, err := svc.Configure(ta.IDAsUUID(), &ConfigureRequest{
			SuspendTrade: &suspend,
		})

		assert.Nil(s.T(), err)
		assert.Equal(s.T(), true, updated)

		a := &models.Account{}
		q := db.DB().Where("id = ?", ta.ID).Find(&a)
		assert.Nil(s.T(), q.Error)
		assert.Equal(s.T(), true, a.TradeSuspendedByUser)
	}

	{
		ta, err := s.account.ToTradeAccount()

		suspend := false
		updated, err := svc.Configure(ta.IDAsUUID(), &ConfigureRequest{
			SuspendTrade: &suspend,
		})

		assert.Nil(s.T(), err)
		assert.Equal(s.T(), true, updated)

		a := &models.Account{}
		q := db.DB().Where("id = ?", ta.ID).Find(&a)
		assert.Nil(s.T(), q.Error)
		assert.Equal(s.T(), false, a.TradeSuspendedByUser)
	}
}
