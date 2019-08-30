package binder

import (
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/rest/api/controller/accesskey"
	"github.com/alpacahq/gobroker/rest/api/controller/account"
	"github.com/alpacahq/gobroker/rest/api/controller/account/configurations"
	"github.com/alpacahq/gobroker/rest/api/controller/affiliate"
	"github.com/alpacahq/gobroker/rest/api/controller/agreements"
	"github.com/alpacahq/gobroker/rest/api/controller/asset"
	"github.com/alpacahq/gobroker/rest/api/controller/bar"
	"github.com/alpacahq/gobroker/rest/api/controller/calendar"
	"github.com/alpacahq/gobroker/rest/api/controller/clock"
	"github.com/alpacahq/gobroker/rest/api/controller/corporateaction"
	"github.com/alpacahq/gobroker/rest/api/controller/docrequest"
	"github.com/alpacahq/gobroker/rest/api/controller/documents"
	"github.com/alpacahq/gobroker/rest/api/controller/email"
	"github.com/alpacahq/gobroker/rest/api/controller/fundamental"
	"github.com/alpacahq/gobroker/rest/api/controller/institution"
	intraAsset "github.com/alpacahq/gobroker/rest/api/controller/intra/asset"
	"github.com/alpacahq/gobroker/rest/api/controller/order"
	"github.com/alpacahq/gobroker/rest/api/controller/owner"
	"github.com/alpacahq/gobroker/rest/api/controller/ownerdetails"
	paperAccessKeys "github.com/alpacahq/gobroker/rest/api/controller/paper/accesskey"
	paperAccount "github.com/alpacahq/gobroker/rest/api/controller/paper/account"
	"github.com/alpacahq/gobroker/rest/api/controller/paper/proxy"
	"github.com/alpacahq/gobroker/rest/api/controller/polygon"
	"github.com/alpacahq/gobroker/rest/api/controller/portfolio/history"
	"github.com/alpacahq/gobroker/rest/api/controller/position"
	"github.com/alpacahq/gobroker/rest/api/controller/profitloss"
	"github.com/alpacahq/gobroker/rest/api/controller/quote"
	"github.com/alpacahq/gobroker/rest/api/controller/relationship"
	"github.com/alpacahq/gobroker/rest/api/controller/transfer"
	"github.com/alpacahq/gobroker/rest/api/controller/trustedcontact"
	"github.com/alpacahq/gobroker/rest/api/middleware/httplogger"
	"github.com/alpacahq/gobroker/utils"
	"github.com/iris-contrib/middleware/cors"
	"github.com/kataras/iris"
)

type APIHandler interface {
	Authenticate(func(api.Context), ...bool) iris.Handler
	NoAuth(func(api.Context), ...bool) iris.Handler
	RouteNotFound(api.Context)
}

// Internal binds all of the internal brokerage API handlers
// to their respective endpoints
func Internal(api *api.API, r iris.Party) {
	//----------------------------------
	//    Broker API
	//----------------------------------
	r.Use(httplogger.New())

	// CORS
	{
		getOrigins := func() []string {
			switch {
			case utils.Prod():
				return []string{"https://app.alpaca.markets"}
			default:
				// staging/dev mode
				return []string{"*"}
			}
		}

		crs := cors.New(cors.Options{
			AllowedOrigins: getOrigins(),
			AllowedMethods: []string{
				iris.MethodGet,
				iris.MethodPost,
				iris.MethodPatch,
				iris.MethodDelete,
				iris.MethodOptions,
			},
			AllowedHeaders:     []string{"*"},
			AllowCredentials:   true,
			OptionsPassthrough: false,
		})

		r.Use(crs)
		r.AllowMethods(iris.MethodOptions) // <- important for the preflight.
	}

	// owner
	r.Get("/owner", api.AuthenticateWithAll(owner.Get))
	r.Patch("/owner", api.AuthenticateWithAll(owner.Patch, utils.StandBy()))

	// account
	r.Get("/accounts", api.AuthenticateWithAll(account.List))
	r.Get("/accounts/{account_id}", api.AuthenticateWithAll(account.Get))
	r.Patch("/accounts/{account_id}", api.AuthenticateWithAll(account.Patch, utils.StandBy()))
	r.Post("/accounts", api.NoAuth(account.Create, utils.StandBy()))

	// trade account & portfolio info
	r.Get("/accounts/{account_id}/trade_account", api.AuthenticateWithAll(account.GetForTrading))
	r.Get("/accounts/{account_id}/profitloss", api.AuthenticateWithAll(profitloss.Get))
	r.Get("/accounts/{account_id}/portfolio/history", api.AuthenticateWithAll(history.Get))
	r.Patch("/accounts/{account_id}/configurations", api.AuthenticateWithAll(configurations.Patch))

	// owner details
	r.Get("/accounts/{account_id}/details", api.AuthenticateWithAll(ownerdetails.Get))
	r.Patch("/accounts/{account_id}/details", api.AuthenticateWithAll(ownerdetails.Patch, utils.StandBy()))

	// affiliates
	r.Get("/accounts/{account_id}/affiliates", api.AuthenticateWithAll(affiliate.List))
	r.Post("/accounts/{account_id}/affiliates", api.AuthenticateWithAll(affiliate.Create))
	r.Patch("/accounts/{account_id}/affiliates/{affiliate_id}", api.AuthenticateWithAll(affiliate.Patch))
	r.Delete("/accounts/{account_id}/affiliates/{affiliate_id}", api.AuthenticateWithAll(affiliate.Delete))

	// trusted contacts
	r.Get("/accounts/{account_id}/trusted_contact", api.AuthenticateWithAll(trustedcontact.Get))
	r.Post("/accounts/{account_id}/trusted_contact", api.AuthenticateWithAll(trustedcontact.Create))
	r.Patch("/accounts/{account_id}/trusted_contact", api.AuthenticateWithAll(trustedcontact.Patch))
	r.Delete("/accounts/{account_id}/trusted_contact", api.AuthenticateWithAll(trustedcontact.Delete))

	// document requests
	r.Get("/accounts/{account_id}/doc_requests", api.AuthenticateWithAll(docrequest.List))
	r.Post("/accounts/{account_id}/doc_requests", api.AuthenticateWithAll(docrequest.Post))

	// ACH relationships
	r.Get("/accounts/{account_id}/relationships", api.AuthenticateWithAll(relationship.List))
	r.Post("/accounts/{account_id}/relationships", api.AuthenticateWithAll(relationship.Create, utils.StandBy()))
	r.Post("/accounts/{account_id}/relationships/{relationship_id}/approve", api.AuthenticateWithAll(relationship.Approve, utils.StandBy()))
	r.Post("/accounts/{account_id}/relationships/{relationship_id}/reissue", api.AuthenticateWithAll(relationship.Reissue, utils.StandBy()))
	r.Delete("/accounts/{account_id}/relationships/{relationship_id}", api.AuthenticateWithAll(relationship.Delete, utils.StandBy()))

	// ACH transfers
	r.Get("/accounts/{account_id}/transfers", api.AuthenticateWithAll(transfer.List))
	r.Post("/accounts/{account_id}/transfers", api.AuthenticateWithAll(transfer.Create, utils.StandBy()))
	r.Delete("/accounts/{account_id}/transfers/{transfer_id}", api.AuthenticateWithAll(transfer.Delete, utils.StandBy()))

	// positions
	r.Get("/accounts/{account_id}/positions", api.AuthenticateWithAll(position.List))
	r.Get("/accounts/{account_id}/positions/{symbol}", api.AuthenticateWithAll(position.Get))
	r.Get("/accounts/{account_id}/orders", api.AuthenticateWithAll(order.List))
	r.Get("/accounts/{account_id}/orders/{order_id}", api.AuthenticateWithAll(order.Get))
	r.Post("/accounts/{account_id}/orders", api.AuthenticateWithAll(order.Create, utils.StandBy()))
	r.Delete("/accounts/{account_id}/orders/{order_id}", api.AuthenticateWithAll(order.Delete, utils.StandBy()))

	// api keys
	r.Get("/access_keys", api.AuthenticateWithAll(accesskey.List))
	r.Post("/access_keys", api.AuthenticateWithAll(accesskey.Create, utils.StandBy()))
	r.Delete("/access_keys/{key_id}", api.AuthenticateWithAll(accesskey.Delete, utils.StandBy()))

	// send emails
	r.Post("/emails", api.AuthenticateWithAll(email.Create))

	// plaid institutions
	r.Get("/institutions/{institution_id}", api.AuthenticateWithAll(institution.Get))

	// data agreements
	r.Get("/accounts/{account_id}/agreements/{agreement}/preview", api.NoAuth(agreements.Get))
	r.Post("/accounts/{account_id}/agreements/{agreement}/accept", api.AuthenticateWithAll(agreements.Post, utils.StandBy()))

	// account documents
	r.Get("/accounts/{account_id}/documents", api.AuthenticateWithAll(documents.List))

	// papertrading

	// accounts
	r.Get("/accounts/{account_id}/paper_accounts", api.AuthenticateWithAll(paperAccount.List))
	r.Get("/accounts/{account_id}/paper_accounts/{paper_account_id}", api.AuthenticateWithAll(paperAccount.Get))
	r.Post("/accounts/{account_id}/paper_accounts", api.AuthenticateWithAll(paperAccount.Create, utils.StandBy()))
	r.Delete("/accounts/{account_id}/paper_accounts/{paper_account_id}", api.AuthenticateWithAll(paperAccount.Delete, utils.StandBy()))

	// access keys
	r.Get("/paper_accounts/{paper_account_id}/access_keys", api.AuthenticateWithAll(paperAccessKeys.List))
	r.Post("/paper_accounts/{paper_account_id}/access_keys", api.AuthenticateWithAll(paperAccessKeys.Create, utils.StandBy()))
	r.Delete("/paper_accounts/{paper_account_id}/access_keys/{key_id}", api.AuthenticateWithAll(paperAccessKeys.Delete, utils.StandBy()))

	// papertrading internal proxy

	// trade account
	r.Get("/paper_accounts/{paper_account_id}/trade_account", api.AuthenticateWithAll(proxy.Proxy))
	r.Get("/paper_accounts/{paper_account_id}/portfolio/history", api.AuthenticateWithAll(proxy.Proxy))
	r.Get("/paper_accounts/{paper_account_id}/profitloss", api.AuthenticateWithAll(proxy.Proxy))
	r.Patch("/paper_accounts/{paper_account_id}/configurations", api.AuthenticateWithAll(proxy.Proxy))

	// orders
	r.Get("/paper_accounts/{paper_account_id}/orders", api.AuthenticateWithAll(proxy.Proxy))
	r.Get("/paper_accounts/{paper_account_id}/orders/{order_id}", api.AuthenticateWithAll(proxy.Proxy))
	r.Post("/paper_accounts/{paper_account_id}/orders", api.AuthenticateWithAll(proxy.Proxy, utils.StandBy()))
	r.Delete("/paper_accounts/{paper_account_id}/orders/{order_id}", api.AuthenticateWithAll(proxy.Proxy, utils.StandBy()))

	// positions
	r.Get("/paper_accounts/{paper_account_id}/positions", api.AuthenticateWithAll(proxy.Proxy))
	r.Get("/paper_accounts/{paper_account_id}/positions/{symbol}", api.AuthenticateWithAll(proxy.Proxy))
}

// PaperTrader bind API endpoints for papertrader app integration
func PaperTrader(api *api.API, r iris.Party) {
	r.Get("/assets", api.NoAuth(intraAsset.List))
	r.Get("/corporate_actions", api.NoAuth(corporateaction.List))
}

// Trade binds all of the external trading API handlers
// to their respective endpoints
func Trade(api APIHandler, r iris.Party) {
	//----------------------------------
	//    Trading API
	//----------------------------------
	r.Use(httplogger.New())

	// account
	r.Get("/account", api.Authenticate(account.GetForTrading))
	r.Patch("/account/configurations", api.Authenticate(configurations.Patch))

	// positions
	r.Get("/positions", api.Authenticate(position.List))
	r.Get("/positions/{symbol}", api.Authenticate(position.Get))

	// orders
	r.Get("/orders", api.Authenticate(order.List))
	r.Get("/orders/{order_id}", api.Authenticate(order.Get))
	r.Get("/orders:by_client_order_id", api.Authenticate(order.GetByClientOrderID))
	r.Post("/orders", api.Authenticate(order.Create, utils.StandBy()))
	r.Delete("/orders/{order_id}", api.Authenticate(order.Delete, utils.StandBy()))

	// assets
	r.Get("/assets", api.Authenticate(asset.List))
	r.Get("/assets/{symbol}", api.Authenticate(asset.Get))

	// fundamentals
	r.Get("/fundamentals", api.Authenticate(fundamental.List))
	r.Get("/assets/{symbol}/fundamental", api.Authenticate(fundamental.Get))

	// quotes
	r.Get("/quotes", api.Authenticate(quote.List))
	r.Get("/assets/{symbol}/quote", api.Authenticate(quote.Get))

	// bars
	r.Get("/bars", api.Authenticate(bar.List))
	r.Get("/assets/{symbol}/bars", api.Authenticate(bar.Get))

	// market clock & calendar
	r.Get("/clock", api.Authenticate(clock.Get))
	r.Get("/calendar", api.Authenticate(calendar.Get))

	r.Any("/", api.NoAuth(api.RouteNotFound))
	r.Any("/{anypath}", api.NoAuth(api.RouteNotFound))
}

// Polygon binds all of the internal polygon API handlers
// to their respective endpoints
func Polygon(api *api.API, r iris.Party) {
	// ---------------------------------
	// 	  Polygon specific interal API
	// ---------------------------------
	r.Use(httplogger.New())

	// auth
	r.Post("/auth", api.AuthenticatePolygon(polygon.Auth))
	r.Post("/keys", api.AuthenticatePolygon(polygon.List))
}
