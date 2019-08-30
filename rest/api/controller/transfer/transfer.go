package transfer

import (
	"github.com/alpacahq/apex"
	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/rest/api/controller/entities"
	"github.com/alpacahq/gobroker/rest/api/controller/parameter"
	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gobroker/service/transfer"
	"github.com/alpacahq/gobroker/utils"
	"github.com/gofrs/uuid"
	"github.com/kataras/iris"
	"github.com/shopspring/decimal"
)

func List(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	var direction *apex.TransferDirection
	if q := ctx.URLParam("direction"); q != "" {
		dir := apex.TransferDirection(q)
		direction = &dir
	}

	srv := transfer.Service().WithTx(ctx.Tx())

	if xfers, err := srv.List(accountID, direction, nil, nil); err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond(xfers)
	}
}

func Create(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	xReq := entities.TransferRequest{}
	if err := ctx.Read(&xReq); err != nil {
		ctx.RespondError(gberrors.RequestBodyLoadFailure.WithError(err))
		return
	}

	if err := xReq.Verify(); err != nil {
		ctx.RespondError(err)
		return
	}

	srv := transfer.Service().WithTx(ctx.Tx())

	xfer, err := srv.Create(
		accountID,
		xReq.RelationshipID,
		xReq.Direction,
		xReq.Amount,
	)

	if err != nil {
		ctx.RespondError(err)
		return
	}

	// in DEV mode, there are no SoD files to parse to update
	// the account's cash balances, so we update them here
	// instantly for transfers
	if utils.Dev() {
		aSrv := account.Service().WithTx(ctx.Tx())

		acc, err := aSrv.GetByID(accountID)
		if err != nil {
			ctx.RespondError(err)
			return
		}

		var tradable, withdrawable decimal.Decimal
		if xfer.Direction == apex.Incoming {
			tradable = acc.Cash.Add(xfer.Amount)
			withdrawable = acc.CashWithdrawable.Add(xfer.Amount)
		} else {
			tradable = acc.Cash.Sub(xfer.Amount)
			withdrawable = acc.CashWithdrawable.Sub(xfer.Amount)
		}

		_, err = aSrv.Patch(
			accountID,
			map[string]interface{}{
				"cash":              tradable,
				"cash_withdrawable": withdrawable,
			})
		if err != nil {
			ctx.RespondError(err)
			return
		}

		xfer.Status = enum.TransferComplete
		procAt := xfer.CreatedAt.Format("2006-01-02")
		xfer.BatchProcessedAt = &procAt
		if xfer, err = srv.Update(xfer); err != nil {
			ctx.RespondError(err)
			return
		}
	}

	ctx.Respond(xfer)
}

// not currently working
func Delete(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	xferID, err := uuid.FromString(ctx.Params().Get("transfer_id"))
	if err != nil {
		ctx.RespondError(err)
		return
	}

	srv := transfer.Service().WithTx(ctx.Tx())

	if err := srv.Cancel(accountID, xferID.String()); err != nil {
		ctx.RespondError(err)
	} else {
		ctx.RespondWithStatus(nil, iris.StatusNoContent)
	}
}
