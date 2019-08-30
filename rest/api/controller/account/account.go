package account

import (
	"regexp"
	"strings"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/rest/api/controller/entities"
	"github.com/alpacahq/gobroker/rest/api/controller/parameter"
	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gobroker/service/paper"
	"github.com/alpacahq/gobroker/utils/tradingdate"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/cognito"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
)

// CreateAccountRequest is used to pass in the email
// when testing and not using Cognito for auth
type CreateAccountRequest struct {
	Email string `json:"email"`
}

// verify confirms that all of the required data is supplied
// in the CreateAccountRequest object
func (r *CreateAccountRequest) verify() error {
	if r.Email == "" {
		return gberrors.InvalidRequestParam.WithMsg("email is required")
	}
	return nil
}

func parseJWT(ctx api.Context) (uuid.UUID, string, error) {
	header := ctx.Request().Header.Get("Authorization")

	match := regexp.MustCompile("Bearer (.*)").FindStringSubmatch(header)
	if len(match) < 2 {
		return uuid.Nil, "", gberrors.InvalidRequestParam.WithMsg("invalid authorization header value format")
	}

	tokenString := match[1]

	if !cognito.Enabled() {
		return uuid.Must(uuid.NewV4()), "", nil
	}

	token, err := cognito.ParseAndVerify(tokenString)
	if err != nil {
		return uuid.Nil, "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)

	if !ok {
		return uuid.Nil, "", gberrors.InternalServerError
	}

	cognitoID := uuid.FromStringOrNil(claims["sub"].(string))
	if cognitoID == uuid.Nil {
		return uuid.Nil, "", gberrors.Unauthorized
	}

	email := strings.ToLower(claims["email"].(string))
	if email == "" {
		return uuid.Nil, "", gberrors.Unauthorized
	}

	return cognitoID, email, nil
}

func Create(ctx api.Context) {
	var (
		cognitoID uuid.UUID
		email     string
		err       error
	)

	// in dev mode, we don't have a valid JWT to parse
	// and get the email, so it must be posted in the body
	if !cognito.Enabled() {
		aReq := CreateAccountRequest{}
		if err := ctx.Read(&aReq); err != nil {
			ctx.RespondError(gberrors.RequestBodyLoadFailure)
			return
		}

		if err := aReq.verify(); err != nil {
			ctx.RespondError(err)
			return
		}

		email = aReq.Email

		cognitoID, _, err = parseJWT(ctx)
	} else {
		cognitoID, email, err = parseJWT(ctx)
	}

	if err != nil {
		ctx.RespondError(err)
		return
	}

	srv := account.Service().WithTx(ctx.Tx())

	acc, err := srv.Create(email, cognitoID)

	if err != nil {
		ctx.RespondError(err)
		return
	}

	// create the default paper account as well
	paperSvc := paper.Service().WithTx(ctx.Tx())

	// $100k by default
	if _, err = paperSvc.Create(acc.IDAsUUID(), decimal.New(100000, 0)); err != nil {
		ctx.RespondError(err)
		return
	}

	ctx.Respond(acc.ForJSON())
}

func Patch(ctx api.Context) {
	var (
		acct *models.Account
		err  error
	)

	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	aReq := map[string]interface{}{}
	if err := ctx.Read(&aReq); err != nil {
		ctx.RespondError(gberrors.RequestBodyLoadFailure)
		return
	}

	srv := account.Service().WithTx(ctx.Tx())
	srv.SetForUpdate()

	acct, err = srv.Patch(accountID, aReq)
	if err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond(acct.ForJSON())
	}
}

func List(ctx api.Context) {
	accountID := ctx.Session().ID

	srv := account.Service().WithTx(ctx.Tx())

	// Right now, an owner can have only one account, so using GetByID instead of List
	acct, err := srv.GetByID(accountID)

	if err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond([]interface{}{acct.ForJSON()})
	}
}

func Get(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	srv := account.Service().WithTx(ctx.Tx())

	acct, err := srv.GetByID(accountID)

	if err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond(acct.ForJSON())
	}
}

func GetForTrading(ctx api.Context) {
	accountID, err := parameter.GetParamAccountID(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	// using multipe table, so use repetable tx.
	tx := ctx.RepeatableTx()

	srv := ctx.Services().Account().WithTx(tx)
	acct, err := srv.GetByID(accountID)

	if err != nil {
		ctx.RespondError(err)
		return
	}

	pSrv := ctx.Services().Position().WithTx(tx)

	positions, err := pSrv.List(accountID)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	// should be replaced w/ tradeaccount service
	balances, err := srv.GetBalancesByAccount(acct, tradingdate.Last(clock.Now()).MarketOpen())
	if err != nil {
		ctx.RespondError(gberrors.InternalServerError.WithError(err))
		return
	}

	portfolioValue := balances.Cash
	for _, p := range positions {
		portfolioValue = portfolioValue.Add(p.MarketValue)
	}

	o := entities.AccountForTrading{
		ID:                   acct.ID,
		Status:               string(acct.Status),
		Currency:             acct.Currency,
		BuyingPower:          balances.BuyingPower,
		Cash:                 balances.Cash,
		CashWithdrawable:     balances.CashWithdrawable,
		PortfolioValue:       portfolioValue,
		PatternDayTrader:     acct.PatternDayTrader,
		TradingBlocked:       acct.TradingBlocked,
		TransfersBlocked:     acct.TransfersBlocked,
		AccountBlocked:       acct.AccountBlocked,
		CreatedAt:            acct.CreatedAt,
		TradeSuspendedByUser: acct.TradeSuspendedByUser,
	}

	ctx.Respond(o)
}
