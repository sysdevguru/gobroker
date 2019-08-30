package position

import (
	"fmt"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/rest/api/controller/parameter"
)

func List(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	srv := ctx.Services().Position().WithTx(ctx.Tx())

	positions, err := srv.List(accountID)

	if err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond(positions)
	}
}

func Get(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	assetKey := ctx.Params().Get("symbol")
	if assetKey == "" {
		ctx.RespondError(gberrors.InvalidRequestParam.WithMsg("symbol is required"))
		return
	}

	asset := ctx.Services().AssetCache().Get(assetKey)
	if asset == nil {
		ctx.RespondError(gberrors.InvalidRequestParam.WithMsg(
			fmt.Sprintf("could not find asset for \"%s\"", assetKey)))
		return
	}

	srv := ctx.Services().Position().WithTx(ctx.Tx())

	positions, err := srv.GetByAssetID(accountID, asset.IDAsUUID())

	if err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond(positions)
	}
}
