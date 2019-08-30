package email

import (
	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/rest/api/controller/entities"
	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gobroker/service/email"
	"github.com/kataras/iris"
)

func Create(ctx api.Context) {
	req := entities.EmailRequest{}

	if err := ctx.Read(&req); err != nil {
		ctx.RespondError(gberrors.RequestBodyLoadFailure.WithError(err))
		return
	}

	acctID, err := req.Validate()
	if err != nil {
		ctx.RespondError(err)
		return
	}

	aSrv := account.Service().WithTx(ctx.Tx())

	acct, err := aSrv.GetByID(*acctID)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	eSrv := email.Service().WithTx(ctx.Tx())

	if err := eSrv.Create(acct, req.Type, req.DeliverAt); err != nil {
		ctx.RespondError(err)
	} else {
		ctx.RespondWithStatus(nil, iris.StatusNoContent)
	}
}
