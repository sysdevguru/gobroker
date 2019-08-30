package institution

import (
	"github.com/alpacahq/gobroker/external/plaid"
	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/service/relationship"
)

func Get(ctx api.Context) {
	institutionID := ctx.Params().Get("institution_id")

	if institutionID == "" {
		ctx.RespondError(gberrors.InternalServerError)
		return
	}

	srv := relationship.Service().WithTx(ctx.Tx())

	institution, err := srv.GetPlaidInstitution(institutionID)
	if err != nil {
		if apiErr, ok := err.(*plaid.APIError); ok && apiErr.CanDisplay() {
			ctx.RespondError(gberrors.InternalServerError.WithMsg(*apiErr.DisplayMessage).WithError(err))
			return
		}
		ctx.RespondError(err)
		return
	}

	ctx.Respond(institution)
}
