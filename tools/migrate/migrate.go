package main

import (
	"github.com/alpacahq/gobroker/migration"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/env"
	"github.com/alpacahq/gopaca/log"
)

func init() {
	env.RegisterDefault("PGDATABASE", "gobroker")
	env.RegisterDefault("PGHOST", "127.0.0.1")
	env.RegisterDefault("PGUSER", "postgres")
	env.RegisterDefault("PGPASSWORD", "alpacas")
}

func main() {
	if err := migration.Migration(db.DB()).Migrate(); err != nil {
		log.Fatal("database error", "action", "migration", "error", err)
	}
	db.DB().Close()
	log.Info("migration successful")
}
