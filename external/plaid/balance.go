package plaid

import (
	"strings"

	"github.com/alpacahq/gopaca/env"
	"github.com/shopspring/decimal"
)

type BalanceResponse struct {
	Accounts  []Account   `json:"accounts"`
	Item      interface{} `json:"item"`
	RequestID string      `json:"request_id"`
}

type Account struct {
	AccountID    string   `json:"account_id"`
	Balances     Balances `json:"balances"`
	Mask         string   `json:"mask"`
	Name         string   `json:"name"`
	OfficialName string   `json:"official_name"`
	Subtype      string   `json:"subtype"`
	Type         string   `json:"type"`
}

type Balances struct {
	Available float64     `json:"available"`
	Current   float64     `json:"current"`
	Limit     interface{} `json:"limit"`
}

func (pc *PlaidClient) GetBalance(accessToken, accountID string) (*decimal.Decimal, error) {
	resp := BalanceResponse{}

	err := pc.Request(
		"POST",
		"/accounts/balance/get",
		map[string]interface{}{
			"client_id":    env.GetVar("PLAID_CLIENT_ID"),
			"secret":       env.GetVar("PLAID_SECRET"),
			"access_token": accessToken,
			"options": map[string]interface{}{
				"account_ids": []string{accountID},
			},
		},
		&resp)
	if err != nil {
		return nil, err
	}

	for _, account := range resp.Accounts {
		if strings.EqualFold(account.AccountID, accountID) {
			balance := decimal.NewFromFloat(account.Balances.Available)
			return &balance, nil
		}
	}
	return nil, err
}
