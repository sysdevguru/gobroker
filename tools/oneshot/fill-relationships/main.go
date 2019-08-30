package main

import (
	"fmt"

	"github.com/alpacahq/gobroker/external/plaid"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/utils/initializer"
	"github.com/alpacahq/gopaca/db"
)

func main() {
	initializer.Initialize()

	tx := db.DB().Begin()

	var rels []*models.ACHRelationship
	if err := tx.Where("status = ?", enum.RelationshipApproved).Find(&rels).Error; err != nil {
		panic(err)
	}

	for i := range rels {
		fmt.Println("running", rels[i].ID)

		plaidAuth, err := plaid.Client().GetAuth(*rels[i].PlaidToken)
		if err != nil {
			fmt.Println("error happens", err.Error())
			continue
		}

		var account map[string]interface{}
		for _, entry := range plaidAuth["accounts"].([]interface{}) {
			if acc, ok := entry.(map[string]interface{}); ok {
				if acc["account_id"] == *rels[i].PlaidAccount {
					account = acc
				}
			}
		}

		mask := account["mask"].(string)
		name := account["name"].(string)

		if err := tx.Model(&models.ACHRelationship{}).Where("id = ?", rels[i].ID).Updates(models.ACHRelationship{
			Mask:     &mask,
			Nickname: &name,
		}).Error; err != nil {
			fmt.Println("failed to update", err.Error())
		}
	}

	if err := tx.Commit().Error; err != nil {
		panic(err)
	}

	fmt.Println("done")
}
