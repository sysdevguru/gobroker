package agreements

import (
	"github.com/alpacahq/gobroker/external/polygon"
	"github.com/alpacahq/gobroker/gberrors"
	"github.com/gofrs/uuid"
	"github.com/kataras/iris"

	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/rest/api/controller/parameter"
	"github.com/alpacahq/gobroker/service/agreements"
)

func Get(ctx api.Context) {
	accountID, err := uuid.FromString(ctx.Params().Get("account_id"))
	// accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	t, err := parseAgreementType(ctx)
	if err != nil {
		ctx.RespondError(err)
	}

	srv := agreements.Service().WithTx(ctx.Tx())

	if data, err := srv.Get(accountID, t); err != nil {
		ctx.RespondError(err)
	} else {
		ctx.StatusCode(iris.StatusOK)
		ctx.RespondWithContent(api.MIMEApplicationPDF, data)
	}
}

func Post(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	t, err := parseAgreementType(ctx)
	if err != nil {
		ctx.RespondError(err)
	}

	srv := agreements.Service().WithTx(ctx.Tx())

	if err := srv.Accept(accountID, t); err != nil {
		ctx.RespondError(err)
	} else {
		ctx.RespondWithStatus(nil, iris.StatusNoContent)
	}
}

func parseAgreementType(ctx api.Context) (polygon.AgreementType, error) {
	switch polygon.AgreementType(ctx.Params().Get("agreement")) {
	case polygon.NYSE:
		return polygon.NYSE, nil
	case polygon.NASDAQ:
		return polygon.NASDAQ, nil
	default:
		return "", gberrors.InvalidRequestParam.WithMsg("invalid agreement type")
	}
}
