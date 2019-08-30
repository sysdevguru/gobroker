package gbreg

import (
	"encoding/json"

	"github.com/alpacahq/gobroker/service/accesskey"
	"github.com/alpacahq/gobroker/service/asset"
	"github.com/alpacahq/gobroker/service/assetcache"
	"github.com/alpacahq/gobroker/service/bar"
	"github.com/alpacahq/gobroker/service/fundamental"
	"github.com/alpacahq/gobroker/service/order"
	"github.com/alpacahq/gobroker/service/portfolio"
	"github.com/alpacahq/gobroker/service/position"
	"github.com/alpacahq/gobroker/service/profitloss"
	"github.com/alpacahq/gobroker/service/quote"
	"github.com/alpacahq/gobroker/service/registry"
	"github.com/alpacahq/gobroker/service/tradeaccount"
	"github.com/alpacahq/gopaca/env"
	"github.com/alpacahq/gopaca/rmq"
	"github.com/gofrs/uuid"
)

var (
	Services      registry.Registry
	OrderRequests = env.GetVar("ORDER_REQUESTS_QUEUE")
)

type gbRegistry struct{}

func (r *gbRegistry) Account() tradeaccount.TradeAccountService {
	return tradeaccount.Service()
}

func (r *gbRegistry) AccessKey() accesskey.AccessKeyService {
	return accesskey.Service(r.Account()).WithCache()
}

func (r *gbRegistry) Asset() asset.AssetService {
	return asset.Service()
}

func (r *gbRegistry) AssetCache() assetcache.AssetCache {
	return assetcache.GetAssetCache()
}

func (r *gbRegistry) Fundamental() fundamental.FundamentalService {
	return fundamental.Service()
}

func (r *gbRegistry) Quote() quote.QuoteService {
	return quote.Service(r.AssetCache())
}

func (r *gbRegistry) Bar() bar.BarService {
	return bar.Service(r.AssetCache())
}

func (r *gbRegistry) Position() position.PositionService {
	return position.Service(r.AssetCache())
}

func (r *gbRegistry) Order() order.OrderService {
	return order.Service(
		submitTrade,
		r.Position(),
		tradeaccount.Service(),
	)
}

func (r *gbRegistry) Portfolio() portfolio.PortfolioService {
	return portfolio.Service(
		r.AssetCache(),
		r.Account(),
		r.Position(),
	)
}

func (r *gbRegistry) ProfitLoss() profitloss.ProfitLossService {
	return profitloss.Service(
		r.AssetCache(),
	)
}

func init() {
	Services = &gbRegistry{}
}

func submitTrade(acctID uuid.UUID, msg interface{}) error {
	buf, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return rmq.Produce(OrderRequests, buf)
}
