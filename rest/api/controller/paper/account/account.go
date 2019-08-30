package account

import (
	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/rest/api/controller/parameter"
	"github.com/alpacahq/gobroker/service/paper"
	"github.com/kataras/iris"
	"github.com/shopspring/decimal"
)

func List(ctx api.Context) {
	accID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	svc := paper.Service().WithTx(ctx.Tx())

	accounts, err := svc.List(accID)
	if err != nil {
		ctx.RespondError(err)
		return
	} else {
		ctx.Respond(accounts)
	}
}

func Create(ctx api.Context) {
	accID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	params := struct {
		Cash decimal.Decimal `json:"cash"`
	}{}

	if err := ctx.Read(&params); err != nil {
		ctx.RespondError(gberrors.RequestBodyLoadFailure.WithError(err))
		return
	}

	if params.Cash.Equal(decimal.Zero) {
		ctx.RespondError(gberrors.InvalidRequestParam.WithMsg("initial capital must be greater than zero"))
		return
	} else if params.Cash.LessThan(decimal.Zero) {
		// set zero, and let service create w/ real account's total portfolio value
		params.Cash = decimal.Zero
	}

	// might better to have upper limit

	svc := paper.Service().WithTx(ctx.Tx())

	account, err := svc.Create(accID, params.Cash)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	ctx.Respond(account)
}

func Delete(ctx api.Context) {
	acctID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	paperAcctID, err := parameter.GetParamPaperAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	svc := paper.Service().WithTx(ctx.Tx())

	if err := svc.Delete(acctID, paperAcctID); err != nil {
		ctx.RespondError(err)
	} else {
		ctx.RespondWithStatus(nil, iris.StatusNoContent)
	}
}

func Get(ctx api.Context) {
	acctID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	paperAcctID, err := parameter.GetParamPaperAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	svc := paper.Service().WithTx(ctx.Tx())

	if acct, err := svc.GetByID(acctID, paperAcctID); err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond(acct)
	}
}
