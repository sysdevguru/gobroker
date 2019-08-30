package accesskey

import (
	"net/http"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/rest/api/controller/parameter"
	"github.com/alpacahq/gobroker/service/paper"
)

func List(ctx api.Context) {
	paperAccountID, err := parameter.GetParamPaperAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	svc := paper.Service().WithTx(ctx.Tx())

	keys, err := svc.GetAccessKeys(ctx.Session().ID, paperAccountID)
	if err != nil {
		ctx.RespondError(err)
		return
	}
	ctx.Respond(keys)
}

func Create(ctx api.Context) {
	paperAccountID, err := parameter.GetParamPaperAccountID(ctx)

	if err != nil {
		ctx.RespondError(err)
		return
	}

	svc := paper.Service().WithTx(ctx.Tx())

	accessKey, err := svc.CreateAccessKey(ctx.Session().ID, paperAccountID)

	if err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond(accessKey)
	}
}

func Delete(ctx api.Context) {
	paperAccountID, err := parameter.GetParamPaperAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	keyID := ctx.Params().Get("key_id")
	if keyID == "" {
		ctx.RespondError(gberrors.InvalidRequestParam.WithMsg("key_id is required"))
		return
	}

	svc := paper.Service().WithTx(ctx.Tx())

	err = svc.DeleteAccessKey(ctx.Session().ID, paperAccountID, keyID)
	if err != nil {
		ctx.RespondError(err)
	} else {
		ctx.RespondWithStatus(nil, http.StatusNoContent)
	}
}
