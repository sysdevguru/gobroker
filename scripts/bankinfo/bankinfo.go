package main

import (
	"fmt"
	"os"

	"github.com/alpacahq/gobroker/service/relationship"
	"github.com/alpacahq/gopaca/db"
	"github.com/gofrs/uuid"
)

func main() {
	// Input the account ID
	acctID := os.Args[1]

	srv := relationship.Service().WithTx(db.DB())

	acctUUID, err := uuid.FromString(acctID)
	if err != nil {
		panic(err)
	}

	rels, err := srv.List(acctUUID, nil)
	if err != nil {
		panic(err)
	}

	for _, rel := range rels {
		bInfo, berr := rel.GetBankInfo()
		if berr != nil {
			panic(err)
		}
		fmt.Println(fmt.Sprintf("Account Number: %v | Routing Number: %v | Account Owner Name: %v | Account Type: %v",
			bInfo.Account, bInfo.RoutingNumber, bInfo.AccountOwnerName, bInfo.AccountType))
	}

	return
}
