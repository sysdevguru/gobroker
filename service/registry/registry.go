package registry

import (
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
	"github.com/alpacahq/gobroker/service/tradeaccount"
)

type Registry interface {
	Account() tradeaccount.TradeAccountService
	AccessKey() accesskey.AccessKeyService
	Asset() asset.AssetService
	AssetCache() assetcache.AssetCache
	Fundamental() fundamental.FundamentalService
	Quote() quote.QuoteService
	Bar() bar.BarService
	Position() position.PositionService
	Order() order.OrderService
	Portfolio() portfolio.PortfolioService
	ProfitLoss() profitloss.ProfitLossService
}
