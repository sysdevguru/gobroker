package main

import (
	"flag"

	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/utils/initializer"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/service/relationship"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/log"
	"github.com/gofrs/uuid"
)

func init() {
	initializer.Initialize()

	flag.Parse()
}

func main() {
	tx := db.Begin()

	srv := relationship.Service().WithTx(tx)

	relationships := []models.ACHRelationship{}

	if err := tx.Where("status = ?", enum.RelationshipApproved).Find(&relationships).Error; err != nil {
		tx.Rollback()
		panic(err)
	}

	log.Info("canceling relationships", "count", len(relationships))

	for _, rel := range relationships {
		id, err := uuid.FromString(rel.AccountID)
		if err != nil {
			tx.Rollback()
			panic(err)
		}
		if err := srv.Cancel(id, rel.ID); err != nil {
			tx.Rollback()
			panic(err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		panic(err)
	}

	log.Info("done")
}
