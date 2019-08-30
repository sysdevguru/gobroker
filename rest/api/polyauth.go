package api

import (
	"net"
	"strings"
	"sync"

	"github.com/alpacahq/gobroker/gberrors"
	"github.com/alpacahq/gopaca/env"
	"github.com/kataras/iris"
)

var (
	polyNet  *net.IPNet
	polyOnce sync.Once
)

func (api *API) AuthenticatePolygon(handler func(Context)) iris.Handler {
	return api.Handler(func(ctx Context) {
		var err error

		polyOnce.Do(func() {
			if env.GetVar("POLYGON_CIDR") != "" {
				_, polyNet, err = net.ParseCIDR(env.GetVar("POLYGON_CIDR"))
			}
		})

		if err != nil {
			ctx.RespondError(gberrors.InternalServerError)
			return
		}

		key := ctx.Request().Header.Get("APCA-POLYGON-KEY")

		if !strings.EqualFold(key, env.GetVar("POLYGON_AUTH_TOKEN")) {
			ctx.RespondError(gberrors.NewUnauthorized(40110000, "invalid access key"))
			return
		}

		if polyNet != nil && !polyNet.Contains(net.ParseIP(ctx.RemoteAddr())) {
			ctx.RespondError(gberrors.NewUnauthorized(40110000, "invalid source ip"))
			return
		}

		handler(ctx)
	})
}
