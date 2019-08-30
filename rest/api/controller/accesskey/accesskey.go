package accesskey

import (
	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/kataras/iris"
)

func List(ctx api.Context) {
	srv := ctx.Services().AccessKey().WithTx(ctx.Tx())

	accessKeys, err := srv.List(ctx.Session().ID)

	if err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond(accessKeys)
	}
}

func Create(ctx api.Context) {
	srv := ctx.Services().AccessKey().WithTx(ctx.Tx())

	if !ctx.Session().Authorized(ctx.Session().ID) {
		ctx.RespondError(gberrors.Unauthorized)
		return
	}

	accessKey, err := srv.Create(ctx.Session().ID, enum.LiveAccount)
	if err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond(accessKey)
	}
}

func Delete(ctx api.Context) {
	srv := ctx.Services().AccessKey().WithTx(ctx.Tx())

	keyID := ctx.Params().Get("key_id")

	if _, err := srv.Disable(ctx.Session().ID, keyID); err != nil {
		ctx.RespondError(err)
	} else {
		ctx.RespondWithStatus(nil, iris.StatusNoContent)
	}
}
