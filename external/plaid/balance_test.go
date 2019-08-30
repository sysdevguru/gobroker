package plaid

import (
	"encoding/json"
	"fmt"

	"github.com/stretchr/testify/assert"

	"github.com/shopspring/decimal"
	"github.com/valyala/fasthttp"
)

func (s *PlaidTestSuite) TestGetBalance() {
	// clean request
	available := decimal.NewFromFloat(10000)
	id := "some_account_id"
	pc.request = func(req *fasthttp.Request, resp *fasthttp.Response) error {
		body, _ := json.Marshal(BalanceResponse{
			Accounts: []Account{
				Account{
					AccountID: id,
					Balances: Balances{
						Available: 10000,
						Current:   10000,
					},
				},
			},
		})
		resp.SetBody(body)
		resp.SetStatusCode(200)
		return nil
	}
	balance, err := pc.GetBalance("some_access_token", id)
	assert.Nil(s.T(), err)
	assert.True(s.T(), balance.Equals(available))

	// >400
	pc.request = func(req *fasthttp.Request, resp *fasthttp.Response) error {
		resp.SetStatusCode(400)
		return fmt.Errorf("400 code")
	}
	balance, err = pc.GetBalance("some_access_token", id)
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), balance)
}
