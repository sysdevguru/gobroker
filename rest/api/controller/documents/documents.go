package documents

import (
	"github.com/alpacahq/apex"
	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/rest/api/controller/entities"
	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gobroker/utils"
)

func List(ctx api.Context) {
	q := entities.DocumentQuery{}

	if err := q.Parse(ctx.Request()); err != nil {
		ctx.RespondError(err)
		return
	}

	srv := account.Service().WithTx(ctx.Tx())

	acct, err := srv.GetByID(ctx.Session().ID)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	if acct.ApexAccount == nil || utils.Dev() {
		ctx.Respond([]interface{}{})
		return
	}

	docs, err := apex.Client().
		GetDocuments(
			*acct.ApexAccount,
			q.Start, q.End,
			apex.DocumentTypeFromString(q.Type))

	if err != nil {
		ctx.RespondError(gberrors.InternalServerError.WithError(err))
	} else {
		ctx.Respond(docs)
	}
}
