package polygon

import (
	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/rest/api/controller/entities"
	"github.com/alpacahq/gobroker/service/paper"
	"github.com/alpacahq/gobroker/service/polygon"
)

func Auth(ctx api.Context) {
	req := entities.PolyAuthRequest{}

	if err := ctx.Read(&req); err != nil {
		ctx.RespondError(gberrors.RequestBodyLoadFailure.WithError(err))
		return
	}

	if req.APIKeyID == "" {
		ctx.RespondError(gberrors.InvalidRequestParam.WithMsg("api_key_id is required"))
		return
	}

	srv := polygon.Service().WithTx(ctx.Tx())

	if acct, err := srv.VerifyKey(req.APIKeyID); err != nil {
		if gberrors.IsNotFound(err) {
			paperSvc := paper.Service().WithTx(ctx.Tx())

			if resp, err := paperSvc.VerifyKeyForPolygon(req.APIKeyID); err != nil {
				ctx.RespondError(gberrors.Unauthorized.WithError(err))
			} else {
				ctx.Respond(resp)
			}
		} else {
			ctx.RespondError(err)
		}
	} else {
		ctx.Respond(map[string]interface{}{"user_id": acct.PrimaryOwner().ID})
	}
}

func List(ctx api.Context) {
	req := entities.PolySubscriberRequest{}

	if err := ctx.Read(&req); err != nil {
		ctx.RespondError(gberrors.RequestBodyLoadFailure.WithError(err))
		return
	}

	if len(req.APIKeyIDs) == 0 {
		ctx.RespondError(gberrors.InvalidRequestParam.WithMsg("api_key_ids are required"))
		return
	}

	srv := polygon.Service().WithTx(ctx.Tx())

	if accts, err := srv.List(req.APIKeyIDs); err != nil {
		ctx.RespondError(err)
	} else {
		resp := make(map[string]interface{}, len(accts))
		for key, acct := range accts {
			addr, err := acct.PrimaryOwner().Details.FormatAddress()
			if err != nil {
				ctx.RespondError(err)
				return
			}
			resp[key] = entities.PolySubscriberEntity{
				Email:        acct.PrimaryOwner().Email,
				UserID:       acct.PrimaryOwner().ID,
				FullName:     *acct.PrimaryOwner().Details.LegalName,
				Professional: false,
				Address:      addr,
			}
		}

		// aggregate with papertrader
		paperSvc := paper.Service().WithTx(ctx.Tx())

		paperResp, err := paperSvc.ListKeysForPolygon(req.APIKeyIDs)
		if err != nil {
			ctx.RespondError(err)
			return
		}

		for key, acct := range paperResp {
			resp[key] = acct
		}

		ctx.Respond(resp)
	}
}
