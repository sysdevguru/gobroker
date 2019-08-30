package asset

import (
	"fmt"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/rest/api"
)

func Get(ctx api.Context) {
	assetKey := ctx.Params().Get("symbol")
	if assetKey == "" {
		ctx.RespondError(gberrors.InvalidRequestParam.WithMsg(
			"symbol is required"))
		return
	}

	asset := ctx.Services().AssetCache().Get(assetKey)

	if asset == nil {
		ctx.RespondError(gberrors.NotFound.WithMsg(fmt.Sprintf("asset not found for %v", assetKey)))
		return
	}

	ctx.Respond(asset)
}

func List(ctx api.Context) {
	var (
		status *enum.AssetStatus
		class  *enum.AssetClass
	)

	if q := ctx.URLParam("status"); q != "" {
		s := enum.AssetStatus(q)
		status = &s
	}

	if q := ctx.URLParam("asset_class"); q != "" {
		s := enum.AssetClass(q)
		class = &s
	}

	srv := ctx.Services().Asset().WithTx(ctx.Tx())

	assets, err := srv.List(class, status)

	if err != nil {
		ctx.RespondError(err)
		return
	}

	ctx.Respond(assets)
}
