package main

import (
	"flag"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gobroker/utils/initializer"
	"github.com/alpacahq/gopaca/cognito"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/log"
	"github.com/gocarina/gocsv"
)

var (
	csv         = flag.Bool("csv", false, "generate cognito import csv")
	importUsers = flag.Bool("importUsers", false, "import users from cognito (only call once CSV has been generated and loaded into cognito")
)

func init() {
	flag.Parse()
	initializer.Initialize()
}

func main() {
	if *csv {
		generateCognitoCSV()
	} else if *importUsers {
		importCognitoUsers()
	} else {
		log.Error("no valid option selected - select either -csv or -importUsers")
	}
}

func importCognitoUsers() {
	email := "email"

	users, err := cognito.ListUsers([]*string{&email})
	if err != nil {
		log.Panic("cognito list users failure", "error", err)
	}

	log.Info("cognito users retrieved", "count", len(users))

	tx := db.Begin()

	srv := account.Service().WithTx(tx)

	accounts, _, err := srv.List(account.AccountQuery{Page: 0, Per: math.MaxInt32})

	if err != nil {
		tx.Rollback()
		log.Panic("failed to query accounts for cognito migration", "error", err)
	}

	if len(users) < len(accounts) {
		log.Warn("account to user count mismatch", "accounts", len(accounts), "cognito_users", len(users))
	}

	log.Info("migrating accounts", "count", len(accounts))

	start := time.Now()

	for _, account := range accounts {
		updated := false

		for _, user := range users {
			for _, attr := range user.Attributes {
				if strings.EqualFold(*attr.Name, email) && strings.EqualFold(*attr.Value, account.PrimaryOwner().Email) {
					if _, err = srv.PatchInternal(
						account.IDAsUUID(),
						map[string]interface{}{"cognito_id": *user.Username},
					); err != nil {
						tx.Rollback()
						log.Panic(
							"failed to update account record",
							"account", account.ID,
							"cognito_id", *user.Username)
					} else {
						updated = true
					}
				}
			}
		}
		if !updated {
			log.Warn("account not found in cognito user pool", "account", account.ID)
		}
	}

	if err = tx.Commit().Error; err != nil {
		tx.Rollback()
		log.Panic("database error", "error", err)
	}

	log.Info("accounts migrated", "elapsed", time.Now().Sub(start).String())
}

func generateCognitoCSV() {
	srv := account.Service().WithTx(db.DB())

	cognitoRecords := []CognitoImport{}

	accounts, _, err := srv.List(account.AccountQuery{Page: 0, Per: math.MaxInt32})

	if err != nil {
		log.Panic("failed to query accounts for cognito migration", "error", err)
	}

	for _, acct := range accounts {
		record := CognitoImport{
			Email:             acct.PrimaryOwner().Email,
			EmailVerified:     true,
			AlpacaAccountID:   acct.ID,
			CognitoMFAEnabled: false,
			CognitoUsername:   acct.PrimaryOwner().Email,
		}

		details := acct.PrimaryOwner().Details

		if details.LegalName != nil {
			record.Name = *details.LegalName
		}

		if details.GivenName != nil {
			record.GivenName = *details.GivenName
		}

		if details.FamilyName != nil {
			record.FamilyName = *details.FamilyName
		}

		cognitoRecords = append(cognitoRecords, record)
	}

	str, err := gocsv.MarshalString(cognitoRecords)
	if err != nil {
		log.Panic("failed to marshal CSV", "error", err)
	}

	fmt.Println(str)
}

// CognitoImport is a record to import a user to Cognito. It was
// generated from the CSV headers produced from the AWS dashboard.
type CognitoImport struct {
	Name                string `csv:"name"`
	GivenName           string `csv:"given_name"`
	FamilyName          string `csv:"family_name"`
	MiddleName          string `csv:"middle_name"`
	Nickname            string `csv:"nickname"`
	PreferredUsername   string `csv:"preferred_username"`
	Profile             string `csv:"profile"`
	Picture             string `csv:"picture"`
	Website             string `csv:"website"`
	Email               string `csv:"email"`
	EmailVerified       bool   `csv:"email_verified"`
	Gender              string `csv:"gender"`
	BirthDate           string `csv:"birthdate"`
	ZoneInfo            string `csv:"zoneinfo"`
	Locale              string `csv:"locale"`
	PhoneNumber         string `csv:"phone_number"`
	PhoneNumberVerified bool   `csv:"phone_number_verified"`
	Address             string `csv:"address"`
	UpdatedAt           string `csv:"updated_at"`
	AlpacaAccountID     string `csv:"custom:account_id"`
	CognitoMFAEnabled   bool   `csv:"cognito:mfa_enabled"`
	CognitoUsername     string `csv:"cognito:username"`
}

// CSV Header
//
// name
// given_name
// family_name
// middle_name
// nickname
// preferred_username
// profile
// picture
// website
// email
// email_verified
// gender
// birthdate
// zoneinfo
// locale
// phone_number
// phone_number_verified
// address
// updated_at
// custom:account_id
// cognito:mfa_enabled
// cognito:username
