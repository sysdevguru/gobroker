package plaid

import (
	"errors"
	"fmt"

	"github.com/alpacahq/gopaca/env"
)

type Exchange struct {
	Token string `json:"plaid_token"`
	Item  string `json:"plaid_item"`
}

func (pc *PlaidClient) ExchangeToken(publicToken string) (*Exchange, error) {
	tokenItem := map[string]interface{}{}
	err := pc.Request(
		"POST",
		"/item/public_token/exchange",
		map[string]interface{}{
			"client_id":    env.GetVar("PLAID_CLIENT_ID"),
			"secret":       env.GetVar("PLAID_SECRET"),
			"public_token": publicToken,
		},
		&tokenItem,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate plaid access_token - %v", err)
	}
	return &Exchange{
		Token: tokenItem["access_token"].(string),
		Item:  tokenItem["item_id"].(string),
	}, nil
}

func (pc *PlaidClient) GetItem(accessToken string) (map[string]interface{}, error) {
	resp := map[string]interface{}{}
	err := pc.Request(
		"POST",
		"/item/get",
		map[string]interface{}{
			"client_id":    env.GetVar("PLAID_CLIENT_ID"),
			"secret":       env.GetVar("PLAID_SECRET"),
			"access_token": accessToken,
		},
		&resp,
	)
	if err != nil {
		return nil, err
	}
	return resp["item"].(map[string]interface{}), nil
}

func (pc *PlaidClient) GetAuth(accessToken string) (map[string]interface{}, error) {
	resp := map[string]interface{}{}
	err := pc.Request(
		"POST",
		"/auth/get",
		map[string]interface{}{
			"client_id":    env.GetVar("PLAID_CLIENT_ID"),
			"secret":       env.GetVar("PLAID_SECRET"),
			"access_token": accessToken,
		},
		&resp,
	)
	if err != nil {
		return nil, err
	}
	accounts := resp["accounts"].([]interface{})
	if len(accounts) == 0 {
		return nil, errors.New("no accounts in auth")
	}
	return resp, nil
}
