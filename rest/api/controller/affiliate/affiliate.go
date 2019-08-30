package affiliate

import (
	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/rest/api/controller/entities"
	"github.com/alpacahq/gobroker/rest/api/controller/parameter"
	"github.com/alpacahq/gobroker/service/affiliate"
	"github.com/kataras/iris"
)

func Create(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}
	aReq := entities.CreateAffiliateRequest{}
	if err := ctx.Read(&aReq); err != nil {
		ctx.RespondError(gberrors.RequestBodyLoadFailure)
		return
	}

	if err := aReq.Verify(); err != nil {
		ctx.RespondError(err)
		return
	}

	srv := affiliate.Service().WithTx(ctx.Tx())

	aff, err := srv.Create(aReq.Model(accountID))

	if err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond(aff)
	}
}

func List(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	srv := affiliate.Service().WithTx(ctx.Tx())

	if aff, err := srv.List(accountID); err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond(aff)
	}
}

func Patch(ctx api.Context) {
	var (
		aff *models.Affiliate
		err error
	)

	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	affiliateID, err := ctx.Params().GetInt("affiliate_id")
	if err != nil || affiliateID <= 0 {
		ctx.RespondError(gberrors.InvalidRequestParam.WithMsg("invalid affiliate_id").WithError(err))
		return
	}

	aReq := map[string]interface{}{}
	if err := ctx.Read(&aReq); err != nil {
		ctx.RespondError(gberrors.RequestBodyLoadFailure)
		return
	}

	srv := affiliate.Service().WithTx(ctx.Tx())

	aff, err = srv.Patch(accountID, uint(affiliateID), aReq)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	ctx.Respond(aff)
}

func Delete(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	affiliateID, err := ctx.Params().GetInt("affiliate_id")
	if err != nil || affiliateID <= 0 {
		ctx.RespondError(gberrors.InvalidRequestParam)
	}

	srv := affiliate.Service().WithTx(ctx.Tx())

	if err = srv.Delete(accountID, uint(affiliateID)); err != nil {
		ctx.RespondError(err)
	} else {
		ctx.RespondWithStatus(nil, iris.StatusNoContent)
	}
}
