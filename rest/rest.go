// The rest package defines gobroker's RESTful API service
package rest

import (
	"context"

	"github.com/alpacahq/gobroker/debugui"
	"github.com/alpacahq/gobroker/rest/api"
	"github.com/alpacahq/gobroker/rest/api/binder"
	"github.com/alpacahq/gobroker/service/registry"
	"github.com/alpacahq/gobroker/stream"
	"github.com/alpacahq/gobroker/utils"
	"github.com/kataras/iris"
)

var app *iris.Application

func Start(port string, services registry.Registry) error {
	return run((":" + port), services)
}

func Shutdown(ctx context.Context) error {
	if app != nil {
		return app.Shutdown(ctx)
	}
	return nil
}

func bindAPI(api *api.API, binder func(*api.API, iris.Party)) func(iris.Party) {
	return func(r iris.Party) {
		binder(api, r)
	}
}

func bindTradeAPI(api *api.API, binder func(binder.APIHandler, iris.Party)) func(iris.Party) {
	return func(r iris.Party) {
		binder(api, r)
	}
}

func run(host string, services registry.Registry) error {
	app = iris.New()

	apis := api.New(api.NewAuthenticator(), services)

	// polygon API
	app.PartyFunc("/gobroker/api/_polygon/v1", bindAPI(apis, binder.Polygon))

	// internal API
	app.PartyFunc("/gobroker/api/_internal/v1", bindAPI(apis, binder.Internal))

	// API for papertrader integration
	app.PartyFunc("/gobroker/api/_papertrader/v1", bindAPI(apis, binder.PaperTrader))

	// trading API / (v1)
	app.PartyFunc("/gobroker/api/v1", bindTradeAPI(apis, binder.Trade))

	if utils.Dev() {
		dui := &debugui.DebugUI{}
		app.PartyFunc("/gobroker/debugui", dui.Bind)
	}

	// heartbeat
	app.HandleMany("GET HEAD", "/gobroker/heartbeat", func(ctx iris.Context) {
		ctx.StatusCode(iris.StatusOK)
		ctx.JSON(struct {
			Status  string `json:"status"`
			Version string `json:"version"`
		}{
			"alive", utils.Version,
		})
	})

	// streaming
	app.Any("/stream", iris.FromStd(stream.Handler))

	return app.Run(
		iris.Addr(host),
		iris.WithConfiguration(iris.Configuration{
			// Disable it to re-fetch request body again for logging purpose.
			DisableBodyConsumptionOnUnmarshal: true,
			// Enable real IP forwarding, which is reliable when it is on private proxy.
			RemoteAddrHeaders: map[string]bool{
				"X-Forwarded-For": true,
			},
		}),
		iris.WithoutInterruptHandler,
	)
}
