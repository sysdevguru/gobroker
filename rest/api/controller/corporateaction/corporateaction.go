package corporateaction

import (
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/service/corporateaction"
)

func List(ctx api.Context) {
	srv := corporateaction.Service().WithTx(ctx.Tx())

	if actions, err := srv.List(ctx.Params().Get("date")); err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond(actions)
	}
}
