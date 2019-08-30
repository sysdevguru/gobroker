package mailer

import (
	"fmt"
	"time"

	"github.com/alpacahq/apex"
	"github.com/alpacahq/gobroker/external/mailgun"
	"github.com/alpacahq/gobroker/mailer/templates"
	"github.com/alpacahq/gobroker/mailer/templates/layouts"
	"github.com/alpacahq/gobroker/mailer/templates/partials"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/utils"
	"github.com/alpacahq/gopaca/log"
	humanize "github.com/dustin/go-humanize"
	"github.com/shopspring/decimal"
)

var (
	// external
	sender     = "Alpaca Support <support@alpaca.markets>"
	devSender  = "Dev Test<devtest@alpaca.markets>"
	archive    = "auto-archive-bd@alpaca.markets"
	devArchive = "DevTest Archive<devtestarchive@alpaca.markets>"
	// internal
	internalSender = "Alpaca System <system@alpaca.markets>"

	dateFormat = "Jan 2, 2006"
	timeFormat = "Jan 2, '06 15:04"
)

type MailType string

const (
	// external
	MarginCall          MailType = "margin_call"
	PatternDayTrader    MailType = "pattern_day_trader"
	MoneyDeposit        MailType = "money_deposit"
	MoneyWithdrawal     MailType = "money_withdrawal"
	AccountApproved     MailType = "account_approved"
	OrderExecuted       MailType = "order_fullfilled"
	EmailConfirm        MailType = "email_confirm"
	AdHoc               MailType = "ad_hoc"
	MicroDepositSuccess MailType = "microdeposit_success"
	MicroDepositFail    MailType = "microdeposit_fail"
	RelinkBankMFA       MailType = "relink_bank_mfa"
	// internal
	MonthlySettlement MailType = "monthly_settlement"
)

func getBcc() string {
	if utils.Prod() {
		return archive
	}
	return devArchive
}

func getSender() string {
	if utils.Prod() {
		return sender
	}
	return devSender
}

// 3AP12345 -> 3AP...45
func MaskApexAccount(acc string) string {
	masked := ""
	length := len(acc)
	maskEnd := length - 4
	// seriously Go, no math.Max(int, int)???
	if maskEnd < 5 {
		maskEnd = 5
	}
	for i := 0; i < len(acc); i++ {
		if i >= 3 && i <= maskEnd {
			masked += "."
		} else {
			masked += string(acc[i])
		}
	}
	return masked
}

// SendMarginCall notifies the account owner of a pending margin call
func SendMarginCall(
	acct, givenName, email string,
	dueDate time.Time,
	callAmount decimal.Decimal,
	deliverAt *time.Time) error {

	amt, _ := callAmount.Float64()

	tmplData := struct {
		Name    string
		Account string
		Amount  string
		DueDate string
	}{
		Name:    givenName,
		Account: MaskApexAccount(acct),
		Amount:  humanize.CommafWithDigits(amt, 2),
		DueDate: dueDate.Format(dateFormat),
	}

	html, err := templates.ExecuteTemplate(layouts.Base(), partials.MarginCall, tmplData)
	if err != nil {
		return err
	}

	msg := mailgun.Email{
		Sender:    getSender(),
		Subject:   "Margin Call Issued",
		HTML:      html,
		Recipient: email,
		DeliverAt: deliverAt,
		Bcc:       getBcc(),
	}

	if err = mailgun.Send(msg); err != nil {
		log.Error(
			"mailer send error",
			"type", MarginCall,
			"account", acct,
			"error", err)
	}

	return err
}

// SendPDTCall notifies the account owner that they have been marked a pattern day trader
// and that there is an open equity maintenance call that must be met.
func SendPDTCall(
	acct, givenName, email string,
	dueDate time.Time,
	callAmount decimal.Decimal,
	deliverAt *time.Time) error {

	amt, _ := callAmount.Float64()

	tmplData := struct {
		Name    string
		Account string
		Amount  string
		DueDate string
	}{
		Name:    givenName,
		Account: MaskApexAccount(acct),
		Amount:  humanize.CommafWithDigits(amt, 2),
		DueDate: dueDate.Format(dateFormat),
	}

	html, err := templates.ExecuteTemplate(layouts.Base(), partials.PatternDayTrader, tmplData)
	if err != nil {
		return err
	}

	msg := mailgun.Email{
		Sender:    getSender(),
		Subject:   "Pattern Day Trader Margin Call Issued",
		HTML:      html,
		Recipient: email,
		DeliverAt: deliverAt,
		Bcc:       getBcc(),
	}

	if err = mailgun.Send(msg); err != nil {
		log.Error(
			"mailer send error",
			"type", PatternDayTrader,
			"account", acct,
			"error", err)
	}

	return err
}

// SendMoneyTransferNotification notifies the account owner that money transfer is complete
func SendMoneyTransferNotification(
	acct, givenName, email string,
	transferredOn time.Time,
	direction apex.TransferDirection,
	transferAmount decimal.Decimal,
	deliverAt *time.Time) error {

	amt, _ := transferAmount.Float64()

	tmplData := struct {
		Name         string
		Account      string
		Amount       string
		TransferDate string
	}{
		Name:         givenName,
		Account:      MaskApexAccount(acct),
		Amount:       humanize.CommafWithDigits(amt, 2),
		TransferDate: transferredOn.Format(dateFormat),
	}

	var (
		html, subject string
		mailType      MailType
		err           error
	)

	switch direction {
	case apex.Incoming:
		subject = "Funds Available for Trading"
		mailType = MoneyDeposit

		if html, err = templates.ExecuteTemplate(
			layouts.Base(),
			partials.MoneyDeposit,
			tmplData); err != nil {
			return err
		}
	case apex.Outgoing:
		subject = "Funds Have Been Withdrawn"
		mailType = MoneyWithdrawal

		if html, err = templates.ExecuteTemplate(
			layouts.Base(),
			partials.MoneyWithdrawal,
			tmplData); err != nil {
			return err
		}
	}

	msg := mailgun.Email{
		Sender:    getSender(),
		Subject:   subject,
		HTML:      html,
		Recipient: email,
		DeliverAt: deliverAt,
		Bcc:       getBcc(),
	}

	if err = mailgun.Send(msg); err != nil {
		log.Error(
			"mailer send error",
			"type", mailType,
			"account", acct,
			"error", err)
	}

	return err
}

//SendAccountApprovedNotification Sending email on Account Approval
func SendAccountApprovedNotification(
	acct, givenName, email string,
	deliverAt *time.Time) error {

	tmplData := struct {
		Name    string
		Account string
	}{
		Name:    givenName,
		Account: MaskApexAccount(acct),
	}

	html, err := templates.ExecuteTemplate(layouts.Base(), partials.AccountApproved, tmplData)
	if err != nil {
		return err
	}

	msg := mailgun.Email{
		Sender:    getSender(),
		Subject:   "Alpaca Account Approved",
		HTML:      html,
		Recipient: email,
		DeliverAt: deliverAt,
		Bcc:       getBcc(),
	}

	if err = mailgun.Send(msg); err != nil {
		log.Error(
			"mailer send error",
			"type", AccountApproved,
			"account", acct,
			"error", err)
	}

	return err
}

// SendOrderExecutedNotification notifies the account owner that order is executed
func SendOrderExecutedNotification(
	acct, givenName, email string,
	e *models.Execution,
	deliverAt *time.Time) error {

	px, _ := e.AvgPrice.Float64()

	tmplData := struct {
		Name      string
		Account   string
		Symbol    string
		Side      string
		OrderType string
		Qty       string
		Price     string
		FilledAt  string
	}{
		Name:      givenName,
		Account:   MaskApexAccount(acct),
		Symbol:    e.Symbol,
		Side:      string(e.Side),
		OrderType: e.OrderType.Readable(),
		Qty:       e.CumQty.String(),
		Price:     humanize.CommafWithDigits(px, 2),
		FilledAt:  e.TransactionTime.Format(timeFormat),
	}

	html, err := templates.ExecuteTemplate(layouts.Base(), partials.OrderExecuted, tmplData)
	if err != nil {
		return err
	}

	msg := mailgun.Email{
		Sender:    getSender(),
		Subject:   fmt.Sprintf("Your %s Order Has Been Executed", e.Symbol),
		HTML:      html,
		Recipient: email,
		DeliverAt: deliverAt,
		Bcc:       getBcc(),
	}

	if err = mailgun.Send(msg); err != nil {
		log.Error(
			"mailer send error",
			"type", OrderExecuted,
			"account", acct,
			"error", err)
	}

	return err
}

// SendMonthlySettlement sends the monthly Apex settlement files to
// our internal email for review
func SendMonthlySettlement(date, fileName string, file []byte) (err error) {
	msg := mailgun.Email{
		Sender:    internalSender,
		Subject:   fmt.Sprintf("%s Monthly Settlement Files", date),
		Recipient: "ap@alpaca.markets",
		PlainText: "Please see attached settlement files.",
		Attachment: &mailgun.Attachment{
			Name: fileName,
			Data: file,
		},
	}

	if err = mailgun.Send(msg); err != nil {
		log.Error(
			"mailer send error",
			"type", MonthlySettlement,
			"error", err)
	}

	return
}

// SendAdHoc sends an ad-hoc email crafted by an Alpaca administrator.
func SendAdHoc(subject, body, email string) (err error) {
	tmplData := struct {
		Body string
	}{
		Body: body,
	}

	html, err := templates.ExecuteTemplate(layouts.Base(), partials.AdHoc, tmplData)
	if err != nil {
		return err
	}

	msg := mailgun.Email{
		Sender:    getSender(),
		Subject:   subject,
		HTML:      html,
		Recipient: email,
		DeliverAt: nil,
		Bcc:       getBcc(),
	}

	if err = mailgun.Send(msg); err != nil {
		log.Error(
			"mailer send error",
			"type", AdHoc,
			"error", err)
	}

	return
}

func SendMicroDeposit(success bool, acct, givenName, nickname, email, reason string) (err error) {
	if success == false && reason == "" {
		reason = "Unkown Reason"
	}

	tmplData := struct {
		Name     string
		Nickname string
		Reason   string
	}{
		Name:     givenName,
		Nickname: nickname,
		Reason:   reason,
	}

	var (
		emailTmpl partials.Partial
		subject   string
		errType   MailType
	)
	if success {
		emailTmpl = partials.MicroDepositSuccess
		subject = "Verify Your Bank Account"
		errType = MicroDepositSuccess
	} else {
		emailTmpl = partials.MicroDepositFail
		subject = "Bank Verification Failed"
		errType = MicroDepositFail
	}

	html, err := templates.ExecuteTemplate(layouts.Base(), emailTmpl, tmplData)
	if err != nil {
		return err
	}

	msg := mailgun.Email{
		Sender:    getSender(),
		Subject:   subject,
		HTML:      html,
		Recipient: email,
		DeliverAt: nil,
		Bcc:       getBcc(),
	}

	if err = mailgun.Send(msg); err != nil {
		log.Error(
			"mailer send error",
			"type", errType,
			"account", acct,
			"error", err)
	}

	return
}

func SendPassMFAChange(name, email string) (err error) {
	tmplData := struct {
		Name string
	}{
		Name: name,
	}
	subject := "Action Required - Re-link Bank Account"
	html, err := templates.ExecuteTemplate(layouts.Base(), partials.MFAPasswordChange, tmplData)
	if err != nil {
		return err
	}
	msg := mailgun.Email{
		Sender:    getSender(),
		Subject:   subject,
		HTML:      html,
		Recipient: email,
		DeliverAt: nil,
		Bcc:       getBcc(),
	}
	if err = mailgun.Send(msg); err != nil {
		log.Error(
			"mailer send error",
			"type", RelinkBankMFA,
			"error", err)
	}
	return
}

func SendBalanceLow(name, email string) (err error) {
	tmplData := struct {
		Name string
	}{
		Name: name,
	}
	subject := "Action Required - Confirm Your Bank Account Balance"
	html, err := templates.ExecuteTemplate(layouts.Base(), partials.BalanceLow, tmplData)
	if err != nil {
		return err
	}
	msg := mailgun.Email{
		Sender:    getSender(),
		Subject:   subject,
		HTML:      html,
		Recipient: email,
		DeliverAt: nil,
		Bcc:       getBcc(),
	}
	if err = mailgun.Send(msg); err != nil {
		log.Error(
			"mailer send error",
			"type", RelinkBankMFA,
			"error", err)
	}
	return
}
