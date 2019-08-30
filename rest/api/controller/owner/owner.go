package owner

import (
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gobroker/service/owner"
)

func Get(ctx api.Context) {
	srv := account.Service().WithTx(ctx.Tx())

	if acct, err := srv.GetByID(ctx.Session().ID); err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond(acct.PrimaryOwner())
	}
}

func Patch(ctx api.Context) {
	patches := map[string]interface{}{}

	if err := ctx.Read(&patches); err != nil {
		ctx.RespondError(err)
		return
	}

	srv := owner.Service().WithTx(ctx.Tx())

	if owner, err := srv.Patch(ctx.Session().ID, patches); err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond(owner)
	}
}
