package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/alpacahq/gobroker/sod/backup"
	"github.com/alpacahq/gobroker/sod/files"
	"github.com/alpacahq/gobroker/utils/backfiller"
	"github.com/alpacahq/gobroker/utils/initializer"
)

func main() {
	initializer.Initialize()

	frm := flag.String("frm", "2018/04/17", "--date 2017/07/17")
	to := flag.String("to", "2018/06/21", "--date 2017/07/17")

	flag.Parse()

	filler, err := backfiller.New(*frm, *to)
	if err != nil {
		panic(err)
	}

	central, _ := time.LoadLocation("America/Chicago")

	for filler.Next() {

		var file files.TradeActivityReport

		if err := backup.Load(&file, filler.Value()); err != nil {
			panic(err)
		}

		// central asof, same as SoD does
		asof, _ := time.ParseInLocation("2006-01-02", filler.Value().String(), central)

		if processed, errors := file.Sync(asof); errors > 0 {
			fmt.Println("Fail", errors)
		} else {
			fmt.Println("Done", processed)
		}
	}
}
