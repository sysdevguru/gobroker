package tradeaccount

import (
	"time"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/service/op"
	"github.com/alpacahq/gobroker/utils/date"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

type TradeAccountService interface {
	GetByID(uuid.UUID) (*models.TradeAccount, error)
	GetByApexAccount(string) (*models.TradeAccount, error)
	GetBalancesByAccount(acct *models.TradeAccount, marketOpen time.Time) (*models.IntradayBalances, error)
	GetBalancesByID(accountID uuid.UUID, marketOpen time.Time) (*models.IntradayBalances, error)
	MarkPatternDayTrader(*models.TradeAccount) error
	WithTx(*gorm.DB) TradeAccountService
	ForUpdate() TradeAccountService
	Configure(uuid.UUID, *ConfigureRequest) (bool, error)
}

func Service() TradeAccountService {
	return &tradeAccountService{}
}

type tradeAccountService struct {
	tx          *gorm.DB
	queryOption *string
}

func (s *tradeAccountService) WithTx(tx *gorm.DB) TradeAccountService {
	s.tx = tx
	return s
}

func (s *tradeAccountService) GetByID(accountID uuid.UUID) (*models.TradeAccount, error) {
	acc, err := op.GetAccountByID(s.tx, accountID, s.queryOption)
	if err != nil {
		return nil, err
	}

	tradeAcc, err := acc.ToTradeAccount()
	if err != nil {
		return nil, err
	}

	return tradeAcc, nil
}

func (s *tradeAccountService) GetByApexAccount(apexAccount string) (*models.TradeAccount, error) {
	acc, err := op.GetAccountByApexAccount(s.tx, apexAccount, s.queryOption)
	if err != nil {
		return nil, err
	}

	tradeAcc, err := acc.ToTradeAccount()
	if err != nil {
		return nil, err
	}

	return tradeAcc, nil
}

func (s *tradeAccountService) GetBalancesByAccount(acct *models.TradeAccount, marketOpen time.Time) (*models.IntradayBalances, error) {
	return op.GetAccountBalances(s.tx, acct)
}

func (s *tradeAccountService) GetBalancesByID(accountID uuid.UUID, marketOpen time.Time) (*models.IntradayBalances, error) {
	acct, err := s.GetByID(accountID)
	if err != nil {
		return nil, err
	}
	return op.GetAccountBalances(s.tx, acct)
}

func (s *tradeAccountService) MarkPatternDayTrader(acct *models.TradeAccount) error {
	a, err := s.GetByID(acct.IDAsUUID())
	if err != nil {
		return errors.Wrap(err, "failed to query account")
	}

	d := date.DateOf(clock.Now().In(calendar.NY))

	a.MarkedPatternDayTraderAt = &d
	a.PatternDayTrader = true

	patch := models.Account{
		MarkedPatternDayTraderAt: &d,
		PatternDayTrader:         true,
	}

	return s.tx.Model(models.Account{}).Where("id = ?", a.ID).Updates(patch).Error
}

func (s *tradeAccountService) ForUpdate() TradeAccountService {
	forUpdate := db.ForUpdate
	s.queryOption = &forUpdate
	return s
}

type ConfigureRequest struct {
	SuspendTrade *bool `json:"suspend_trade"`
}

// Configure sets user-configure values to the account. Returns true if
// one of the fields is updated.
func (s *tradeAccountService) Configure(accountID uuid.UUID, req *ConfigureRequest) (bool, error) {
	forUpdate := db.ForUpdate
	acct, err := op.GetAccountByID(s.tx, accountID, &forUpdate)
	if err != nil {
		return false, errors.Wrap(err, "failed to get account")
	}
	toUpdate := false

	if req.SuspendTrade != nil {
		acct.TradeSuspendedByUser = *req.SuspendTrade
		toUpdate = true
	}

	if toUpdate {
		if err := s.tx.Save(acct).Error; err != nil {
			return false, err
		}
	}

	return toUpdate, nil
}
