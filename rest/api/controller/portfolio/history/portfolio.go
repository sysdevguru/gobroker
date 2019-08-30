package history

import (
	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/rest/api/controller/entities"
	"github.com/alpacahq/gobroker/rest/api/controller/parameter"
	"github.com/alpacahq/gobroker/utils/tradingdate"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/clock"
)

func Get(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(gberrors.InvalidRequestParam.WithError(err))
		return
	}
	// set isolation to repeatable read at begninning
	// so it propagates to following calls for consistency in
	// performance calculations
	tx := ctx.RepeatableTx()

	req := entities.NewPortfolioHistoriesRequest(ctx.Request())

	srv := ctx.Services().Portfolio().WithTx(tx)

	now := clock.Now().In(calendar.NY)
	today := tradingdate.Last(now)

	chartdata, err := srv.GetHistory(
		accountID,
		today, req.Timeframe,
		req.Period, &now)

	if err != nil {
		ctx.RespondError(err)
		return
	}

	ctx.Respond(chartdata)
}
