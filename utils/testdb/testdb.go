package testdb

import (
	"fmt"

	"github.com/alpacahq/gobroker/migration"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/env"
	"github.com/jinzhu/gorm"
)

func TearDown() {
	db.DB().Close()
	if err := dropTestDB(); err != nil {
		panic(err)
	}
}

func SetUp() {
	env.RegisterDefault("PGDATABASE", "gobroker_test")
	env.RegisterDefault("PGHOST", "127.0.0.1")
	env.RegisterDefault("PGUSER", "postgres")
	env.RegisterDefault("PGPASSWORD", "alpacas")
	env.RegisterDefault("LOG_DB", "true")

	if err := createTestDB(); err != nil {
		panic(err)
	}

	if err := migration.Migration(db.DB()).Migrate(); err != nil {
		panic(err)
	}
}

func createTestDB() error {
	params := fmt.Sprintf(
		"host=%v user=%v dbname=postgres password=%v sslmode=disable",
		env.GetVar("PGHOST"),
		env.GetVar("PGUSER"),
		env.GetVar("PGPASSWORD"),
	)

	pgdb, err := gorm.Open("postgres", params)
	if err != nil {
		return err
	}
	defer func() {
		pgdb.Close()
	}()
	pgdb.Exec("DROP DATABASE IF EXISTS gobroker_test")
	return pgdb.Exec("CREATE DATABASE gobroker_test").Error
}

func dropTestDB() error {
	params := fmt.Sprintf(
		"host=%v user=%v dbname=postgres password=%v sslmode=disable",
		env.GetVar("PGHOST"),
		env.GetVar("PGUSER"),
		env.GetVar("PGPASSWORD"),
	)

	pgdb, err := gorm.Open("postgres", params)
	if err != nil {
		return err
	}
	defer func() {
		pgdb.Close()
	}()
	return pgdb.Exec("DROP DATABASE IF EXISTS gobroker_test").Error
}
