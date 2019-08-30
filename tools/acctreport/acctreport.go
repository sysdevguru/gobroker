package main

import (
	"math"
	"os"

	"github.com/gocarina/gocsv"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gobroker/utils/initializer"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/encryption"
	"github.com/alpacahq/gopaca/env"
)

func main() {
	initializer.Initialize()

	accounts := []models.Account{}

	srv := account.Service().WithTx(db.DB())

	accounts, _, err := srv.List(account.AccountQuery{Per: math.MaxInt32})
	if err != nil {
		panic(err)
	}

	records := make([]AccountReportRecord, len(accounts))

	for i, acct := range accounts {
		var (
			err error
			ssn []byte
		)

		if acct.PrimaryOwner().Details.HashSSN != nil {
			ssn, err = encryption.DecryptWithkey(*acct.PrimaryOwner().Details.HashSSN, []byte(env.GetVar("BROKER_SECRET")))
			if err != nil {
				panic(err)
			}
		}

		records[i] = AccountReportRecord{
			AccountID:     acct.ID,
			ApexAccountID: acct.ApexAccount,
			FirstName:     acct.PrimaryOwner().Details.GivenName,
			MiddleName:    acct.PrimaryOwner().Details.AdditionalName,
			LastName:      acct.PrimaryOwner().Details.FamilyName,
			TaxID:         string(ssn),
			DateOfBirth:   acct.PrimaryOwner().Details.DateOfBirthString(),
			City:          acct.PrimaryOwner().Details.City,
			State:         acct.PrimaryOwner().Details.State,
			ZipCode:       acct.PrimaryOwner().Details.PostalCode,
		}
	}

	if err := gocsv.MarshalFile(records, os.Stdout); err != nil {
		panic(err)
	}
}

type AccountReportRecord struct {
	AccountID     string  `csv:"account_id"`
	ApexAccountID *string `csv:"apex_account_id"`
	FirstName     *string `csv:"first_name"`
	MiddleName    *string `csv:"middle_name"`
	LastName      *string `csv:"last_name"`
	TaxID         string  `csv:"tax_id"`
	DateOfBirth   *string `csv:"date_of_birth"`
	City          *string `csv:"city"`
	State         *string `csv:"state"`
	ZipCode       *string `csv:"zip_code"`
}
