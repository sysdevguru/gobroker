package models

import (
	"testing"

	"github.com/alpacahq/gobroker/dbtest"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gopaca/db"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type AccountSuite struct {
	dbtest.Suite
	account *models.Account
	asset   *models.Asset
}

func TestAccountSuite(t *testing.T) {
	suite.Run(t, new(AccountSuite))
}

func (s *AccountSuite) SetupSuite() {
	s.SetupDB()

	amt, _ := decimal.NewFromString("1000000")
	apexAcct := "apca_test"
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
	if err := db.DB().Create(s.account).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	s.asset = &models.Asset{
		Class:    enum.AssetClassUSEquity,
		Exchange: "NASDAQ",
		Symbol:   "AAPL",
		Status:   enum.AssetActive,
		Tradable: true,
	}
	if err := db.DB().Create(s.asset).Error; err != nil {
		assert.FailNow(s.T(), err.Error())
	}
}

func (s *AccountSuite) TearDownSuite() {
	s.TeardownDB()
}
