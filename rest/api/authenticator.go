package api

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gopaca/cognito"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/env"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gofrs/uuid"
	jwtmiddleware "github.com/iris-contrib/middleware/jwt"
	"github.com/kataras/iris"
)

type Authenticator interface {
	Authenticate(Context) error
	AuthenticateAdmin(Context) error
}

type authenticator struct {
	Authenticator
}

func NewAuthenticator() Authenticator {
	return &authenticator{}
}

func (a *authenticator) Authenticate(ctx Context) error {
	if ctx.Request().Header.Get("Authorization") != "" {
		return a.authenticateWithCognito(ctx)
	}

	return a.authenticateByAccessKey(ctx)
}

func (a *authenticator) AuthenticateAdmin(ctx Context) error {
	adminID, err := uuid.FromString(ctx.Params().Get("admin_id"))
	if err != nil {
		return gberrors.Unauthorized.WithMsg("invalid admin_id")
	}

	if err = evaluateToken(ctx, adminID, env.GetVar("ADMIN_SECRET")); err != nil {
		return gberrors.Unauthorized.WithMsg(err.Error())
	}

	ctx.Authorize(adminID, PermissionAdmin)

	ctx.Values().Set("admin_id", adminID.String())

	return nil
}

var matcher = regexp.MustCompile("Bearer (.*)")

// Cognito based authentication
func (a *authenticator) authenticateWithCognito(ctx Context) (err error) {
	header := ctx.Request().Header.Get("Authorization")

	match := matcher.FindStringSubmatch(header)
	if len(match) < 2 {
		return gberrors.InvalidRequestParam.WithMsg("invalid authorization header value format")
	}

	tokenString := match[1]

	var (
		token     *jwt.Token
		accountID uuid.UUID
	)

	// in testing mode, token is account ID, so we can avoid
	// cognito during development & testing
	if !cognito.Enabled() {
		accountID = uuid.FromStringOrNil(tokenString)
		goto Authorize
	}

	token, err = cognito.ParseAndVerify(tokenString)
	if err != nil {
		return err
	}

	accountID, err = handleCognitoJWT(token)
	if err != nil {
		return err
	}

Authorize:
	ctx.Authorize(accountID, PermissionAll)

	// Assign account_id for tracking purpose
	ctx.Values().Set("account_id", accountID.String())

	return nil
}

// Access Key based authentication
func (a *authenticator) authenticateByAccessKey(ctx Context) error {
	keyID := ctx.Request().Header.Get("APCA-API-KEY-ID")
	secretKey := ctx.Request().Header.Get("APCA-API-SECRET-KEY")

	// don't need to grab the context's connection, since it
	// should be used by only the required services
	srv := ctx.Services().AccessKey().WithTx(db.DB())

	key, err := srv.Verify(keyID, secretKey)
	if err != nil {
		return gberrors.Unauthorized.WithMsg(fmt.Sprintf("access key verification failed : %v", err))
	}

	ctx.Authorize(key.AccountID, PermissionTrading)

	ctx.Values().Set("account_id", key.AccountID.String())

	return nil
}

func handleCognitoJWT(token *jwt.Token) (uuid.UUID, error) {
	claims, ok := token.Claims.(jwt.MapClaims)

	if !ok {
		return uuid.Nil, gberrors.InternalServerError
	}

	cognitoID := uuid.FromStringOrNil(claims["sub"].(string))
	if cognitoID == uuid.Nil {
		return uuid.Nil, gberrors.Unauthorized
	}

	svc := account.Service().WithTx(db.DB())

	acct, err := svc.GetByCognitoID(cognitoID)
	if err != nil {
		return uuid.Nil, gberrors.Unauthorized
	}

	return acct.IDAsUUID(), nil
}

func evaluateToken(ctx iris.Context, id uuid.UUID, secret string) error {
	token, err := extractToken(ctx, secret)
	if err != nil {
		return err
	}

	claims := token.Claims.(jwt.MapClaims)
	sub := claims["sub"].(map[string]interface{})

	userID, err := uuid.FromString(sub["id"].(string))
	if err != nil {
		return err
	}

	if !token.Valid || claims["iss"] != "alpaca" || userID != id {
		return errors.New("token invalid")
	}

	return nil
}

func extractToken(ctx iris.Context, secret string) (*jwt.Token, error) {
	tokenString, err := jwtMiddleware.Config.Extractor(ctx)
	if err != nil {
		return nil, err
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	return token, nil
}

var jwtMiddleware = jwtmiddleware.New(jwtmiddleware.Config{
	ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
		return []byte(env.GetVar("BROKER_SECRET")), nil
	},
	SigningMethod: jwt.SigningMethodHS256,
	ErrorHandler: func(ctx iris.Context, err string) {
		ctx.StatusCode(iris.StatusUnauthorized)
	},
})
