package entities

import (
	"time"

	"github.com/shopspring/decimal"
)

type AccountForTrading struct {
	ID                   string          `json:"id"`
	Status               string          `json:"status"`
	Currency             string          `json:"currency"`
	BuyingPower          decimal.Decimal `json:"buying_power"`
	Cash                 decimal.Decimal `json:"cash"`
	CashWithdrawable     decimal.Decimal `json:"cash_withdrawable"`
	PortfolioValue       decimal.Decimal `json:"portfolio_value"`
	PatternDayTrader     bool            `json:"pattern_day_trader"`
	TradingBlocked       bool            `json:"trading_blocked"`
	TransfersBlocked     bool            `json:"transfers_blocked"`
	AccountBlocked       bool            `json:"account_blocked"`
	CreatedAt            time.Time       `json:"created_at"`
	TradeSuspendedByUser bool            `json:"trade_suspended_by_user"`
}
