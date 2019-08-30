package main

import (
	"fmt"

	"github.com/alpacahq/gobroker/external/plaid"
	"github.com/alpacahq/gopaca/env"
)

const (
	plaidAccount = "ndDaKQxvqXhnVn8RQZ9YtxOVDDPbdZtXLJX3Kq"
	plaidToken   = "access-production-5fd00ef3-00b3-4d09-a26d-f32cc62e103b"
)

func init() {
	env.RegisterDefault("PLAID_PUBLIC_KEY", "e11605cec07e75aef2ce14c9f5b712")
	env.RegisterDefault("PLAID_SECRET", "e0346c3978d3dd1d0c3c417225bf4c")
	env.RegisterDefault("PLAID_CLIENT_ID", "59f7b8444e95b8782b00bc9b")
	env.RegisterDefault("PLAID_URL", "https://production.plaid.com")
}

func main() {
	bal, err := plaid.Client().GetBalance(plaidToken, plaidAccount)
	if err != nil {
		panic(err)
	}

	fmt.Println("Balance: ", bal.String())
}
