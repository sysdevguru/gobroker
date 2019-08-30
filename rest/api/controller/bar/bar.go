package bar

import (
	"strconv"
	"time"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/rest/api/controller/parameter"
)

func List(ctx api.Context) {
	assetIDs := parameter.GetAssetIDs(ctx)

	var (
		err        error
		start, end *time.Time
		limit      *int
	)

	timeframe := "1D"
	params := ctx.URLParams()

	if s, ok := params["timeframe"]; ok {
		// Need to have validation here
		timeframe = s
	}

	if s, ok := params["start_dt"]; ok {
		start, err = parameter.ParseTimestamp(s, "start_dt")
		if err != nil {
			ctx.RespondError(err)
			return
		}
	}

	if s, ok := params["end_dt"]; ok {
		end, err = parameter.ParseTimestamp(s, "end_dt")
		if err != nil {
			ctx.RespondError(err)
			return
		}
	}

	if s, ok := params["limit"]; ok {
		parsed, err := parseLimit(s)
		if err != nil {
			ctx.RespondError(err)
			return
		}
		limit = &parsed
	}

	srv := ctx.Services().Bar().WithTx(ctx.Tx())

	assetBarsList, err := srv.GetByIDs(assetIDs, timeframe, start, end, limit)

	if err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond(assetBarsList)
	}
}

func Get(ctx api.Context) {
	asset, err := parameter.GetAsset(ctx)
	if err != nil {
		ctx.RespondError(err)
		return
	}

	var start, end *time.Time
	var limit *int
	timeframe := "1D"
	params := ctx.URLParams()

	if s, ok := params["timeframe"]; ok {
		// Need to have validation here
		timeframe = s
	}

	if s, ok := params["start_dt"]; ok {
		start, err = parameter.ParseTimestamp(s, "start_dt")
		if err != nil {
			ctx.RespondError(err)
			return
		}
	}

	if s, ok := params["end_dt"]; ok {
		end, err = parameter.ParseTimestamp(s, "end_dt")
		if err != nil {
			ctx.RespondError(err)
			return
		}
	}

	if s, ok := params["limit"]; ok {
		parsed, err := parseLimit(s)
		if err != nil {
			ctx.RespondError(err)
			return
		}
		limit = &parsed
	}

	service := ctx.Services().Bar().WithTx(ctx.Tx())

	assetBars, err := service.GetByID(asset.IDAsUUID(), timeframe, start, end, limit)

	if err != nil {
		ctx.RespondError(err)
	} else {
		ctx.Respond(assetBars)
	}
}

func parseLimit(value string) (int, error) {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return parsed, gberrors.InvalidRequestParam.WithMsg("limit is invalid format")
	}
	if parsed <= 0 {
		return parsed, gberrors.InvalidRequestParam.WithMsg("limit need to be more than zero")
	}
	return parsed, nil
}
