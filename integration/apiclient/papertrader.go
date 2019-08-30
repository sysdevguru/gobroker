package apiclient

import (
	"fmt"

	"github.com/alpacahq/gobroker/rest/api/controller/entities"
)

type PTClient struct {
	*RestClient
}

func NewPTClient() *PTClient {
	c := &PTClient{
		RestClient: NewRestClient("", ""),
	}
	return c
}

func (c *PTClient) ListAssets() (res []entities.AssetMarshaller, err error) {
	url := fmt.Sprintf("%v/api/_papertrader/v1/assets", c.RestClient.baseUrl)
	_, err = c.call(url, "GET", nil, &res)
	return
}
