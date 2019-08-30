package main

import (
	"fmt"
	"time"

	"github.com/alpacahq/gobroker/utils/initializer"

	"github.com/alpacahq/gobroker/sod/backup"
	"github.com/alpacahq/gobroker/sod/files"
	"github.com/alpacahq/gobroker/utils/backfiller"

	"github.com/alpacahq/gobroker/utils/tradingdate"
	"github.com/alpacahq/gopaca/calendar"
)

func main() {
	initializer.Initialize()

	start, _ := tradingdate.New(time.Date(2018, time.April, 17, 15, 30, 0, 0, calendar.NY))
	end := tradingdate.Current().Prev()

	filler := backfiller.NewWithTradingDate(*start, end)
	central, _ := time.LoadLocation("America/Chicago")

	for filler.Next() {
		var file files.DividendReport

		if err := backup.Load(&file, filler.Value()); err != nil {
			fmt.Println("missing file", filler.Value())
			continue
		}

		// central asof, same as SoD does
		asof, _ := time.ParseInLocation("2006-01-02", filler.Value().String(), central)

		if processed, errors := file.Sync(asof); errors > 0 {
			fmt.Println("Fail", errors)
		} else {
			fmt.Println("Done", processed)
		}

		var cafile files.CashActivityReport

		if err := backup.Load(&cafile, filler.Value()); err != nil {
			fmt.Println("failed to load cash activity", err)
			continue
		}

		if processed, errors := cafile.SyncForBackfill(asof); errors > 0 {
			fmt.Println("Fail", errors)
		} else {
			fmt.Println("Done", processed)
		}
	}

	fmt.Println(start, end)
}
