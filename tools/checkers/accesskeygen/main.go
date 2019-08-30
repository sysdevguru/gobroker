package main

import (
	"flag"
	"fmt"

	"github.com/alpacahq/gobroker/gbreg"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/utils/initializer"
	"github.com/alpacahq/gopaca/db"
)

func main() {
	initializer.Initialize()

	accountID := flag.String("account_id", "", "account id")

	flag.Parse()
	flag.Lookup("logtostderr").Value.Set("true")

	tx := db.Begin()

	var acc models.Account

	if err := tx.Where("id = ?", accountID).Preload("Owners").Find(&acc).Error; err != nil {
		panic(err)
	}

	srv := gbreg.Services.AccessKey().WithTx(tx)

	akey, err := srv.Create(acc.IDAsUUID(), enum.LiveAccount)
	if err != nil {
		panic(err)
	}

	tx.Commit()

	fmt.Println("Generation Success")
	fmt.Println("APCA-API-KEY-ID", akey.ID)
	fmt.Println("APCA-API-SECRET-KEY", akey.Secret)
}
