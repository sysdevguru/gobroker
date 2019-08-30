package trustedcontact

import (
	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/rest/api/controller/entities"
	"github.com/alpacahq/gobroker/rest/api/controller/parameter"
	"github.com/alpacahq/gobroker/service/trustedcontact"
	"github.com/kataras/iris"
)

func Create(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	tReq := entities.CreateTrustedContactRequest{}

	if err := ctx.Read(&tReq); err != nil {
		ctx.RespondError(gberrors.RequestBodyLoadFailure)
		return
	}

	if err := tReq.Verify(); err != nil {
		ctx.RespondError(err)
		return
	}

	srv := trustedcontact.Service().WithTx(ctx.Tx())

	tc, err := srv.Upsert(tReq.Model(accountID))

	if err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond(tc)
	}
}

func Patch(ctx api.Context) {
	var (
		tc  *models.TrustedContact
		err error
	)

	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	tReq := map[string]interface{}{}
	if err := ctx.Read(&tReq); err != nil {
		ctx.RespondError(gberrors.RequestBodyLoadFailure)
		return
	}

	srv := trustedcontact.Service().WithTx(ctx.Tx())

	tc, err = srv.Patch(accountID, tReq)
	if err != nil {
		ctx.RespondError(err)
		return
	} else {
		ctx.Respond(tc)
	}
}

func Delete(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	srv := trustedcontact.Service().WithTx(ctx.Tx())

	if err = srv.Delete(accountID); err != nil {
		ctx.RespondError(err)
	} else {
		ctx.RespondWithStatus(nil, iris.StatusNoContent)
	}
}

func Get(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		return
	}

	srv := trustedcontact.Service().WithTx(ctx.Tx())

	if tc, err := srv.GetByID(accountID); err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond(tc)
	}
}
