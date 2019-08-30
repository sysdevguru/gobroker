package email

import (
	"time"

	"github.com/alpacahq/gobroker/mailer/templates"
	"github.com/alpacahq/gobroker/mailer/templates/layouts"
	"github.com/alpacahq/gobroker/mailer/templates/partials"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/dustin/go-humanize"
	"github.com/kataras/iris"
)

func AccountApproved(ctx iris.Context) {
	tmplData := struct {
		Name    string
		Account string
	}{
		Name:    "First",
		Account: "AP000000",
	}

	html, err := templates.ExecuteTemplate(layouts.Base(), partials.AccountApproved, tmplData)
	if err != nil {
		ctx.WriteString(err.Error())
		return
	}
	ctx.HTML(html)
}

func OrderExecuted(ctx iris.Context) {
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
		Name:      "First",
		Account:   "AP000000",
		Symbol:    "ALPC",
		Side:      string(enum.Buy),
		OrderType: enum.StopLimit.Readable(),
		Qty:       "3",
		Price:     humanize.CommafWithDigits(float64(122.13), 2),
		FilledAt:  time.Now().Format("Jan 2, '06 15:04"),
	}

	html, err := templates.ExecuteTemplate(layouts.Base(), partials.OrderExecuted, tmplData)
	if err != nil {
		ctx.WriteString(err.Error())
		return
	}
	ctx.HTML(html)
}

func MoneyDeposit(ctx iris.Context) {
	tmplData := struct {
		Name         string
		Account      string
		Amount       string
		TransferDate string
	}{
		Name:         "First",
		Account:      "AP000000",
		Amount:       humanize.CommafWithDigits(float64(1000.00), 2),
		TransferDate: time.Now().Format("Jan 2, 2006"),
	}

	html, err := templates.ExecuteTemplate(layouts.Base(), partials.MoneyDeposit, tmplData)
	if err != nil {
		ctx.WriteString(err.Error())
		return
	}
	ctx.HTML(html)
}

func MoneyWithdrawal(ctx iris.Context) {
	tmplData := struct {
		Name         string
		Account      string
		Amount       string
		TransferDate string
	}{
		Name:         "First",
		Account:      "AP000000",
		Amount:       humanize.CommafWithDigits(float64(1000.00), 2),
		TransferDate: time.Now().Format("Jan 2, 2006"),
	}

	html, err := templates.ExecuteTemplate(layouts.Base(), partials.MoneyWithdrawal, tmplData)
	if err != nil {
		ctx.WriteString(err.Error())
		return
	}
	ctx.HTML(html)
}

func PatternDayTrader(ctx iris.Context) {
	tmplData := struct {
		Name    string
		Account string
		Amount  string
		DueDate string
	}{
		Name:    "First",
		Account: "AP000000",
		Amount:  humanize.CommafWithDigits(float64(1000.00), 2),
		DueDate: time.Now().Format("Jan 2, 2006"),
	}

	html, err := templates.ExecuteTemplate(layouts.Base(), partials.PatternDayTrader, tmplData)
	if err != nil {
		ctx.WriteString(err.Error())
		return
	}
	ctx.HTML(html)
}

func MarginCall(ctx iris.Context) {
	tmplData := struct {
		Name    string
		Account string
		Amount  string
		DueDate string
	}{
		Name:    "First",
		Account: "AP000000",
		Amount:  humanize.CommafWithDigits(float64(1000.00), 2),
		DueDate: time.Now().Format("Jan 2, 2006"),
	}

	html, err := templates.ExecuteTemplate(layouts.Base(), partials.MarginCall, tmplData)
	if err != nil {
		ctx.WriteString(err.Error())
		return
	}
	ctx.HTML(html)
}

func MicroDepositSuccess(ctx iris.Context) {
	tmplData := struct {
		Name     string
		Nickname string
		Reason   string
	}{
		Name:     "First",
		Nickname: "AP000000",
		Reason:   "",
	}

	html, err := templates.ExecuteTemplate(layouts.Base(), partials.MicroDepositSuccess, tmplData)
	if err != nil {
		ctx.WriteString(err.Error())
		return
	}
	ctx.HTML(html)
}

func MicroDepositFail(ctx iris.Context) {
	tmplData := struct {
		Name     string
		Nickname string
		Reason   string
	}{
		Name:     "First",
		Nickname: "AP000000",
		Reason:   "NACHA CODE REASON",
	}

	html, err := templates.ExecuteTemplate(layouts.Base(), partials.MicroDepositFail, tmplData)
	if err != nil {
		ctx.WriteString(err.Error())
		return
	}
	ctx.HTML(html)

}

func MFAPasswordChange(ctx iris.Context) {
	tmplData := struct {
		Name string
	}{
		Name: "First",
	}

	html, err := templates.ExecuteTemplate(layouts.Base(), partials.MFAPasswordChange, tmplData)
	if err != nil {
		ctx.WriteString(err.Error())
		return
	}
	ctx.HTML(html)
}

func BalanceLow(ctx iris.Context) {
	tmplData := struct {
		Name string
	}{
		Name: "First",
	}

	html, err := templates.ExecuteTemplate(layouts.Base(), partials.BalanceLow, tmplData)
	if err != nil {
		ctx.WriteString(err.Error())
		return
	}
	ctx.HTML(html)
}
