package profitloss

import (
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/rest/api/controller/parameter"
	"github.com/alpacahq/gobroker/utils/tradingdate"
	"github.com/alpacahq/gopaca/clock"
)

func Get(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	srv := ctx.Services().ProfitLoss().WithTx(ctx.Tx())

	pl, err := srv.Get(accountID, tradingdate.Current(), clock.Now())

	if err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond(pl)
	}
}
