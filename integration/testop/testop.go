package testop

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/alpacahq/gobroker/gbreg"
	"github.com/alpacahq/gobroker/migration"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gobroker/service/ownerdetails"
	"github.com/alpacahq/gobroker/utils/address"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/log"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	cli "gopkg.in/urfave/cli.v1"
	yaml "gopkg.in/yaml.v2"
)

func recreateDatabase() error {
	pgdb, err := gorm.Open("postgres", "dbname=postgres user=postgres sslmode=disable")
	if err != nil {
		return err
	}
	defer func() {
		pgdb.Close()
	}()
	pgdb.Exec("select pg_terminate_backend(pid) from pg_stat_activity where datname = 'gobroker'")
	pgdb.Exec("DROP DATABASE IF EXISTS gobroker")
	return pgdb.Exec("CREATE DATABASE gobroker").Error
}

func resetDB() error {
	if err := recreateDatabase(); err != nil {
		log.Fatal("database error", "action", "recreate", "error", err)
	}
	if err := migration.Migration(db.DB()).Migrate(); err != nil {
		log.Fatal("database error", "action", "migrate", "error", err)
	}
	if err := Migration(db.DB()).Migrate(); err != nil {
		log.Fatal("database error", "action", "migrate", "error", err)
	}
	assetsFile := "/project/testop/assets.csv"
	if err := db.DB().Exec(fmt.Sprintf("COPY assets FROM '%s' DELIMITER ',' CSV HEADER", assetsFile)).Error; err != nil {
		log.Fatal(
			"database error",
			"action", "copy",
			"from", assetsFile,
			"error", err)
	}
	fundamentalsFile := "/project/testop/fundamentals.txt"
	if err := db.DB().Exec(fmt.Sprintf("COPY fundamentals FROM '%s'", fundamentalsFile)).Error; err != nil {
		log.Fatal(
			"database error",
			"action", "copy",
			"from", fundamentalsFile,
			"error", err)
	}
	log.Info("migration successful")
	return nil
}

func createAccount() error {
	email := "integration-test1@alpaca.markets"
	patches := map[string]interface{}{}
	fileName := "/go/src/github.com/alpacahq/gobroker/tools/acctloader/case1.yml"
	file, err := os.Open(fileName)
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	yamlData, err := ioutil.ReadAll(file)
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	if err := yaml.Unmarshal(yamlData, &patches); err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	var apexAddr address.Address
	for _, val := range patches["street_address"].([]interface{}) {
		apexAddr = append(apexAddr, val.(string))
	}
	patches["street_address"] = apexAddr

	tx := db.Begin()
	acctSrv := account.Service().WithTx(tx)
	acct, err := acctSrv.Create(email, uuid.Must(uuid.NewV4()))
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	tx.Commit()
	fmt.Printf("Account: %v created\n", acct.ID)
	tx = db.Begin()
	ownerSrv := ownerdetails.Service().WithTx(tx)
	details, err := ownerSrv.Patch(acct.IDAsUUID(), patches)
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	tx.Commit()
	fmt.Printf("Owner Details: %v\n", details)
	return nil
}

func createAccessKey() error {
	account := models.Account{}
	if err := db.DB().First(&account).Error; err != nil {
		log.Error("database error", "action", "query", "error", err)
		return err
	}
	if err := db.DB().Model(&account).Related(&account.Owners, "Owners").Error; err != nil {
		log.Error("database error", "action", "query", "error", err)
		return err
	}

	tx := db.Begin()
	service := gbreg.Services.AccessKey().WithTx(tx)

	accessKey, err := service.Create(account.IDAsUUID(), enum.LiveAccount)
	if err != nil {
		log.Error("database error", "action", "create", "error", err)
		return err
	}
	apiKey := ApiKey{
		AccountID: account.ID,
		KeyID:     accessKey.ID,
		SecretKey: accessKey.Secret,
	}

	if err := tx.Create(&apiKey).Error; err != nil {
		return err
	}

	tx.Commit()

	log.Info("access key created", "key-id", accessKey.ID, "secret-key", accessKey.Secret)

	return nil
}

// Setup prepares the DB for integration testing
func Setup() error {

	if err := resetDB(); err != nil {
		return err
	}

	if err := createAccount(); err != nil {
		return err
	}

	return createAccessKey()
}
