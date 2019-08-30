package proxy

import (
	"github.com/alpacahq/gobroker/external/paper"
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/rest/api/controller/parameter"
	paperService "github.com/alpacahq/gobroker/service/paper"
)

func Proxy(ctx api.Context) {
	paperAccountID, err := parameter.GetParamPaperAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	srv := paperService.Service().WithTx(ctx.Tx())

	if _, err = srv.GetByID(ctx.Session().ID, paperAccountID); err != nil {
		ctx.RespondError(err)
		return
	}

	if err = paper.NewClient().Proxy(ctx); err != nil {
		ctx.RespondError(err)
		return
	}
}
