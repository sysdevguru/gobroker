package entities

import (
	"net/http"

	"github.com/alpacahq/gopaca/calendar"
)

type PortfolioHistoriesRequest struct {
	Timeframe calendar.RangeFreq `url:"timeframe"`
	Period    string             `url:"period"`
}

func NewPortfolioHistoriesRequest(r *http.Request) PortfolioHistoriesRequest {
	req := PortfolioHistoriesRequest{}
	params := r.URL.Query()

	req.Period = params.Get("period")
	if req.Period == "" {
		req.Period = "1M"
	}

	req.Timeframe = calendar.RangeFreq(params.Get("timeframe"))
	if req.Timeframe == "" {
		if req.Period == "intraday" {
			req.Timeframe = calendar.Min5
		} else {
			req.Timeframe = calendar.D1
		}
	}

	return req
}
