package main

import (
	"os"

	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/service/ownerdetails"
	"github.com/alpacahq/gopaca/db"
	"github.com/gofrs/uuid"
)

func main() {
	// Input what the city needs to be changed to
	cityChange := os.Args[1]
	accountID := os.Args[2]

	tx := db.DB()
	srv := ownerdetails.Service().WithTx(tx)

	accountUUID, err := uuid.FromString(accountID)
	if err != nil {
		panic(err)
	}

	// Pull out owner details for the account ID
	ownerDetails, err := srv.GetPrimaryByAccountID(accountUUID)
	if err != nil {
		panic(err)
	}

	// Update address
	q := tx.Exec("UPDATE owner_details SET city = ? WHERE id = ? AND replaced_by IS NULL", cityChange, ownerDetails.ID)

	if q.Error != nil {
		panic(q.Error)
	}

	// Update status to ONBOARDING to trigger a resubmit
	q = tx.Exec("UPDATE accounts SET status = ? WHERE id = ?", enum.Onboarding, accountID)

	if q.Error != nil {
		panic(q.Error)
	}

}
