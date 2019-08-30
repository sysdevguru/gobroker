package mkts

import (
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/alpacahq/gopaca/env"
	"github.com/alpacahq/marketstore/frontend"
	"github.com/alpacahq/marketstore/frontend/client"
)

var (
	once                   sync.Once
	cli                    *client.Client
	NewQueryRequestBuilder = frontend.NewQueryRequestBuilder
)

// Client defines a singleton MarketStore JSONRPC client
func Client() *client.Client {
	host := env.GetVar("MARKETSTORE_HOST")
	if strings.HasSuffix(host, "/rpc") {
		host = host[:len(host)-4]
	}
	u, err := url.Parse(host)
	if err != nil {
		panic(fmt.Errorf("failed to load marketstore host(%v) %v", host, err))
	}

	once.Do(func() {
		cli, err = client.NewClient(u.String())
		if err != nil {
			panic(fmt.Errorf("failed to init mktscli %v", err))
		}
	})

	return cli
}
