package sod

import (
	"time"

	"github.com/alpacahq/gobroker/sod"
	"github.com/alpacahq/gobroker/sod/backup"
	"github.com/alpacahq/gobroker/utils/gbevents"
	"github.com/alpacahq/gobroker/utils/tradingdate"
	"github.com/alpacahq/gobroker/workers/snapshot"
	"github.com/alpacahq/gopaca/log"
)

func Work(asOf time.Time) {
	if err := (&sod.SoDProcessor{}).Pull(asOf, false, 5); err != nil {
		log.Error("failed to pull sod files", "asOf", asOf, "error", err)
		return
	}

	// After SoD process is done, then notify all the servers.
	gbevents.TriggerEvent(&gbevents.Event{Name: gbevents.EventAssetRefreshed})

	lastTradeDay, err := tradingdate.New(asOf)
	if err != nil {
		log.Error("failed to get trading date", "asOf", asOf, "error", err)
		return
	}

	snapshot.ProcessSnapshot(*lastTradeDay)

	// Take backup files we are not using yet.
	backup.Backup(*lastTradeDay, *lastTradeDay)
}
