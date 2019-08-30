package debugui

import (
	"github.com/alpacahq/gobroker/debugui/email"
	"github.com/kataras/iris"
)

type DebugUI struct {
}

func (ui *DebugUI) Bind(r iris.Party) {
	r.Get("/email/account_approved", email.AccountApproved)
	r.Get("/email/order_executed", email.OrderExecuted)
	r.Get("/email/money_deposit", email.MoneyDeposit)
	r.Get("/email/money_withdrawal", email.MoneyWithdrawal)
	r.Get("/email/pattern_day_trader", email.PatternDayTrader)
	r.Get("/email/margin_call", email.MarginCall)
	r.Get("/email/microdeposit_success", email.MicroDepositSuccess)
	r.Get("/email/microdeposit_fail", email.MicroDepositFail)
	r.Get("/email/mfa_password_change", email.MFAPasswordChange)
	r.Get("/email/balance_low", email.BalanceLow)
}
