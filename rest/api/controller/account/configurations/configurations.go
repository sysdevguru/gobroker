package configurations

import (
	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/rest/api/controller/parameter"
	"github.com/alpacahq/gobroker/service/tradeaccount"
	"github.com/kataras/iris"
)

// Patch applies configuration change on account
func Patch(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	configureRequest := &tradeaccount.ConfigureRequest{}
	if err = ctx.Read(configureRequest); err != nil {
		ctx.RespondError(gberrors.RequestBodyLoadFailure)
		return
	}

	srv := ctx.Services().Account().WithTx(ctx.Tx())
	if updated, err := srv.Configure(accountID, configureRequest); err != nil {
		ctx.RespondError(err)
		return
	} else {
		if updated {
			ctx.RespondWithStatus(nil, iris.StatusNoContent)
			return
		}
		ctx.RespondError(gberrors.InvalidRequestParam.WithMsg("no fields are updated"))
	}
}
