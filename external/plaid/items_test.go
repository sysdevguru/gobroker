package plaid

import (
	"encoding/json"
	"fmt"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func (s *PlaidTestSuite) TestExchangeToken() {
	// clean request
	pc.request = func(req *fasthttp.Request, resp *fasthttp.Response) error {
		body, _ := json.Marshal(map[string]interface{}{
			"access_token": "some_access_token",
			"item_id":      "some_item_id",
		})
		resp.SetBody(body)
		resp.SetStatusCode(200)
		return nil
	}
	tokenItem, err := pc.ExchangeToken("some_public_token")
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), tokenItem)

	// >400
	pc.request = func(req *fasthttp.Request, resp *fasthttp.Response) error {
		resp.SetStatusCode(400)
		return fmt.Errorf("400 code")
	}
	tokenItem, err = pc.ExchangeToken("some_public_token")
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), tokenItem)
}

func (s *PlaidTestSuite) TestGetItem() {
	// clean request
	pc.request = func(req *fasthttp.Request, resp *fasthttp.Response) error {
		body, _ := json.Marshal(map[string]interface{}{
			"item": map[string]interface{}{
				"institution_id": "some_institution",
				"item_id":        "some_item_id",
			},
		})
		resp.SetBody(body)
		resp.SetStatusCode(200)
		return nil
	}
	item, _ := pc.GetItem("some_access_token")
	assert.NotNil(s.T(), item)

	// >400
	pc.request = func(req *fasthttp.Request, resp *fasthttp.Response) error {
		resp.SetStatusCode(400)
		return fmt.Errorf("400 code")
	}
	item, _ = pc.GetItem("some_access_token")
	assert.Nil(s.T(), item)
}
