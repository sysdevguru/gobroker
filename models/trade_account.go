package models

import (
	"time"

	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/utils/date"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
)

// TradeAccount is used for both paper trading and real trading.
type TradeAccount struct {
	ID                       string
	ApexAccount              *string
	LegalName                string
	Status                   enum.AccountStatus
	Currency                 string
	ApexApprovalStatus       enum.ApexApprovalStatus
	ProtectPatternDayTrader  bool
	PatternDayTrader         bool
	MarkedPatternDayTraderAt *date.Date
	TradingBlocked           bool
	TransfersBlocked         bool
	AccountBlocked           bool
	Cash                     decimal.Decimal
	CashWithdrawable         decimal.Decimal // Not used in papertrading environment, but leave it for compatibility purpose (in balance calc)
	CreatedAt                time.Time
	TradeSuspendedByUser     bool
}

func (a *TradeAccount) IDAsUUID() uuid.UUID {
	return uuid.FromStringOrNil(a.ID)
}

func (a *TradeAccount) Tradable() bool {
	// should be same as Account.Tradable()
	return (a.ApexAccount != nil &&
		a.ApexApprovalStatus == enum.Complete &&
		a.Status == enum.Active &&
		!a.TradingBlocked &&
		!a.AccountBlocked)
}
