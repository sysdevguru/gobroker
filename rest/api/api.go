package api

import (
	"sync"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gobroker/service/registry"
	"github.com/alpacahq/gopaca/log"
	"github.com/kataras/iris"
)

// API containers the authentication and services for
// the broker API
type API struct {
	authenticator Authenticator
	pool          *sync.Pool
	services      registry.Registry
}

// New intializes the API
func New(authenticator Authenticator, services registry.Registry) *API {
	var contextPool = sync.Pool{New: func() interface{} {
		return &context{}
	}}

	return &API{
		authenticator: authenticator,
		pool:          &contextPool,
		services:      services,
	}
}

func (api *API) acquire(original iris.Context) Context {
	ctx := api.pool.Get().(*context)
	ctx.session = nil
	ctx.tx = nil
	ctx.txClosed.Store(true)
	ctx.Context = original
	ctx.services = api.services
	return ctx
}

func (api *API) release(ctx Context) {
	api.pool.Put(ctx)
}

func (api *API) Handler(h func(Context)) iris.Handler {
	return func(original iris.Context) {
		ctx := api.acquire(original)

		// rollback on panic, and propagate up
		defer func() {
			if r := recover(); r != nil {
				ctx.Rollback()
				log.Panic("http request panic", "error", r)
			}
		}()

		h(ctx)

		api.release(ctx)
	}
}

func (api *API) NoAuth(handler func(Context), standBy ...bool) iris.Handler {
	if len(standBy) > 0 && standBy[0] {
		return api.Handler(func(ctx Context) {
			ctx.RespondError(gberrors.Forbidden.WithMsg("stand by mode"))
		})
	}

	return api.Handler(handler)
}

func (api *API) Authenticate(handler func(Context), standBy ...bool) iris.Handler {
	if len(standBy) > 0 && standBy[0] {
		return api.Handler(func(ctx Context) {
			ctx.RespondError(gberrors.Forbidden.WithMsg("stand by mode"))
		})
	}

	return api.Handler(func(ctx Context) {
		if err := api.authenticator.Authenticate(ctx); err != nil {
			ctx.RespondError(gberrors.Unauthorized.WithMsg(err.Error()))
			return
		}
		handler(ctx)
	})
}

func (api *API) AuthenticateAdmin(handler func(Context)) iris.Handler {
	return api.Handler(func(ctx Context) {
		if err := api.authenticator.AuthenticateAdmin(ctx); err != nil {
			ctx.RespondError(gberrors.Unauthorized.WithMsg(err.Error()))
			return
		}
		handler(ctx)
	})
}

func (api *API) AuthenticateWithAll(handler func(Context), standBy ...bool) iris.Handler {
	if len(standBy) > 0 && standBy[0] {
		return api.Handler(func(ctx Context) {
			ctx.RespondError(gberrors.Forbidden.WithMsg("stand by mode"))
		})
	}

	return api.Handler(func(ctx Context) {
		if err := api.authenticator.Authenticate(ctx); err != nil {
			ctx.RespondError(gberrors.Unauthorized.WithMsg(err.Error()))
			return
		}
		if ctx.Session().Permission != PermissionAll {
			ctx.RespondError(gberrors.Unauthorized)
			return
		}
		handler(ctx)
	})
}

func (api *API) RouteNotFound(ctx Context) {
	ctx.RespondError(gberrors.NotFound.WithMsg("endpoint not found"))
}

// Authenticator returns the API's authenticator
func (api *API) Authenticator() Authenticator {
	return api.authenticator
}
