package quote

import (
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/rest/api/controller/parameter"
)

func Get(ctx api.Context) {
	asset, err := parameter.GetAsset(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	srv := ctx.Services().Quote().WithTx(ctx.Tx())

	quote, err := srv.GetByID(asset.IDAsUUID())

	if err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond(quote)
	}
}

func List(ctx api.Context) {
	srv := ctx.Services().Quote().WithTx(ctx.Tx())

	assetIDs := parameter.GetAssetIDs(ctx)

	quotes, err := srv.GetByIDs(assetIDs)

	if err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond(quotes)
	}
}
