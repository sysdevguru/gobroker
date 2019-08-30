package paper

import (
	"fmt"
	"net/http"

	"github.com/alpacahq/gobroker/rest/api/controller/entities"
	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
)

func (c *Client) CreateAccount(cash decimal.Decimal) (accID uuid.UUID, err error) {
	payload := map[string]interface{}{
		"cash": cash.String(),
	}

	r := struct {
		ID string `json:"id"`
	}{}

	err = c.request(http.MethodPost, "/_internal/v1/accounts", payload, &r)
	if err != nil {
		return accID, err
	}

	return uuid.FromString(r.ID)
}

func (c *Client) GetAccountForTrading(accountID uuid.UUID) (acct entities.AccountForTrading, err error) {
	err = c.request(http.MethodGet, fmt.Sprintf("/_internal/v1/accounts/%v/trade_account", accountID), nil, &acct)
	return
}
