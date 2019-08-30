package plaid

import (
	"github.com/alpacahq/gopaca/env"
)

func (pc *PlaidClient) ListInstitutions() (map[string]interface{}, error) {
	resp := map[string]interface{}{}
	err := pc.Request(
		"POST",
		"/institutions/get",
		map[string]interface{}{
			"client_id": env.GetVar("PLAID_CLIENT_ID"),
			"secret":    env.GetVar("PLAID_SECRET"),
			"count":     200,
			"offset":    0,
		},
		&resp,
	)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (pc *PlaidClient) GetInstitution(id string) (map[string]interface{}, error) {
	resp := map[string]interface{}{}
	err := pc.Request(
		"POST",
		"/institutions/get_by_id",
		map[string]interface{}{
			"public_key":     env.GetVar("PLAID_PUBLIC_KEY"),
			"institution_id": id,
			"options": map[string]interface{}{
				"include_display_data": true,
			},
		},
		&resp,
	)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
