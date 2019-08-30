package main

import (
	"os"
	"strconv"

	"github.com/alpacahq/gobroker/service/ownerdetails"
	"github.com/alpacahq/gopaca/db"
	"github.com/gofrs/uuid"
)

func main() {
	//supply account id
	accountID := os.Args[1]

	//supply date
	date := os.Args[2]

	//Admin Name
	adminNumber, err := strconv.Atoi(os.Args[3])
	if err != nil {
		panic(err)
	}
	var adminName string

	if adminNumber == 1 {
		adminName = "Christine Jue"
	} else {
		adminName = "Julie Laughlin"
	}

	//grab owner details obj
	tx := db.DB()
	srv := ownerdetails.Service().WithTx(tx)

	accountUUID, err := uuid.FromString(accountID)
	if err != nil {
		panic(err)
	}

	ownerDetails, err := srv.GetPrimaryByAccountID(accountUUID)
	if err != nil {
		panic(err)
	}

	if ownerDetails.ApprovedBy == nil {
		//assign approved by to be the same as assigned admin
		q := tx.Exec(`UPDATE owner_details SET approved_by = ? WHERE id = ? AND replaced_by IS NULL`, adminName, ownerDetails.ID)

		if q.Error != nil {
			panic(q.Error)
		}
	}

	if ownerDetails.ApprovedAt == nil {
		//assign approved at to be the value passed
		q := tx.Exec(`UPDATE owner_details SET approved_at = ? WHERE id = ? AND replaced_by IS NULL`, date, ownerDetails.ID)

		if q.Error != nil {
			panic(q.Error)
		}
	}
}
