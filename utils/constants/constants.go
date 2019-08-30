package constants

import (
	"github.com/alpacahq/gopaca/env"
	"github.com/alpacahq/gopaca/log"
	"github.com/shopspring/decimal"
)

// this package is for constants that are used in multiple
// locations across the code and don't strictly apply to
// another package (i.e. models etc.)

var (
	// InstantDepositLimit is the limit of cash/buying power
	// allowed for instant deposit transfers
	InstantDepositLimit = func() decimal.Decimal {
		limit, err := decimal.NewFromString(env.GetVar("INSTANT_DEPOSIT_LIMIT"))
		if err != nil {
			log.Error(
				"invalid constant set",
				"name", "INSTANT_DEPOSIT_LIMIT",
				"value", env.GetVar("INSTANT_DEPOSIT_LIMIT"),
				"error", err)
			return decimal.Zero
		}

		return limit
	}()
)
