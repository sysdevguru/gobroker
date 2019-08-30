package parameter

import (
	"fmt"
	"strings"
	"time"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/gofrs/uuid"
)

func GetParamAccountID(ctx api.Context) (accountID uuid.UUID, err error) {
	if ctx.Session().Permission == api.PermissionTrading {
		// API key auth is assumed (might be better to have explicit field in ctx)
		return ctx.Session().ID, nil
	}

	accountID, err = uuid.FromString(ctx.Params().Get("account_id"))

	if err != nil {
		return accountID, gberrors.InvalidRequestParam
	}

	if ctx.Session().Permission == api.PermissionAdmin {
		return accountID, nil
	}

	if !ctx.Session().Authorized(accountID) {
		// Use not found, instead unauthorized to hide account for other people.
		return accountID, gberrors.NotFound
	}

	return accountID, nil
}

func GetParamAdminID(ctx api.Context) (adminID uuid.UUID, err error) {
	if ctx.Session().Permission != api.PermissionAdmin {
		return ctx.Session().ID, fmt.Errorf("non administrator permission level")
	}

	adminID, err = uuid.FromString(ctx.Values().Get("admin_id").(string))
	if err != nil {
		return adminID, gberrors.InvalidRequestParam
	}

	if !ctx.Session().Authorized(adminID) {
		return adminID, gberrors.NotFound
	}

	return adminID, nil
}

func GetParamPaperAccountID(ctx api.Context) (uuid.UUID, error) {
	accID := ctx.Params().Get("paper_account_id")
	if accID == "" {
		return uuid.Nil, gberrors.InvalidRequestParam.WithMsg("paper_account_id is required")
	}

	u, err := uuid.FromString(accID)
	if err != nil {
		return uuid.Nil, gberrors.InvalidRequestParam.WithMsg("paper_account_id is invalid format")
	}

	return u, nil
}

func ParseTimestamp(tStr, fieldName string) (*time.Time, error) {
	t, err := time.Parse(time.RFC3339, tStr)
	if err != nil {
		t, err = time.Parse("2006-01-02", tStr)
		if err != nil {
			return nil, gberrors.InvalidRequestParam.WithMsg(
				fmt.Sprintf("%v is invalid format. please format timestamp with YYYY-MM-DD or ISO8601 like: '2006-01-02T15:04:05Z'", fieldName))
		}
	}

	return &t, nil
}

type assetGetter func(string) *models.Asset

func symbolToAssetIDs(getter assetGetter, symbols []string) []uuid.UUID {
	uuids := []uuid.UUID{}

	for _, symbol := range symbols {
		asset := getter(symbol)
		if asset == nil {
			continue
		}
		uuids = append(uuids, asset.IDAsUUID())
	}

	return uuids
}

func GetAsset(ctx api.Context) (*models.Asset, error) {
	assetKey := ctx.Params().Get("symbol")
	if assetKey == "" {
		return nil, gberrors.InvalidRequestParam.WithMsg(
			"symbol is required")
	}

	asset := ctx.Services().AssetCache().Get(assetKey)

	if asset == nil {
		return nil, gberrors.NotFound.WithMsg(
			fmt.Sprintf("asset not found for %v", assetKey))
	}

	return asset, nil
}

func GetAssetIDs(ctx api.Context) []uuid.UUID {
	symbols := ctx.URLParam("symbols")
	if symbols == "" {
		symbols = ctx.URLParam("symbol")
	}

	return symbolToAssetIDs(ctx.Services().AssetCache().Get, strings.Split(symbols, ","))
}
