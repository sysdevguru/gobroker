package fundamental

import (
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/rest/api/controller/parameter"
)

func Get(ctx api.Context) {
	srv := ctx.Services().Fundamental().WithTx(ctx.Tx())

	asset, err := parameter.GetAsset(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	fundamental, err := srv.GetByID(asset.IDAsUUID())

	if err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond(fundamental)
	}
}

func List(ctx api.Context) {
	assetIDs := parameter.GetAssetIDs(ctx)

	srv := ctx.Services().Fundamental().WithTx(ctx.Tx())

	fundamentals, err := srv.GetByIDs(assetIDs)

	if err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond(fundamentals)
	}
}
