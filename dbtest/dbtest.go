package dbtest

import (
	"fmt"
	"os"

	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/env"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/suite"
)

type Suite struct {
	suite.Suite
	DatabaseID *uuid.UUID
}

func (s *Suite) SetDatabaseID(id uuid.UUID) {
	if s.DatabaseID != nil {
		s.FailNowf("testing database ID already set", "database_id: %s", s.DatabaseID.String())
	}

	s.DatabaseID = &id
}

func (s *Suite) SetupDB() {
	s.SetDatabaseID(setup())
}

func (s *Suite) TeardownDB() {
	teardown(*s.DatabaseID)
}

func teardown(id uuid.UUID) {
	db.DB().Close()
	if err := dropTestDB(id); err != nil {
		panic(err)
	}
}

func setup() (id uuid.UUID) {
	env.RegisterDefault("PGHOST", "127.0.0.1")
	env.RegisterDefault("PGUSER", "postgres")
	env.RegisterDefault("PGPASSWORD", "alpacas")
	env.RegisterDefault("LOG_DB", "true")

	id = uuid.Must(uuid.NewV4())
	database := fmt.Sprintf("gbtest_%s", id.String())

	if err := createTestDB(id); err != nil {
		panic(err)
	}

	os.Setenv("PGDATABASE", database)

	return
}

func createTestDB(id uuid.UUID) error {
	params := fmt.Sprintf(
		"host=%v user=%v password=%v dbname=postgres sslmode=disable",
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

	pgdb.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "gbtest_%s"`, id.String()))

	return pgdb.Exec(fmt.Sprintf(`CREATE DATABASE "gbtest_%s" WITH TEMPLATE gbtest`, id.String())).Error
}

func dropTestDB(id uuid.UUID) error {
	params := fmt.Sprintf(
		"host=%v user=%v password=%v dbname=postgres sslmode=disable",
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

	return pgdb.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "gbtest_%s"`, id.String())).Error
}
