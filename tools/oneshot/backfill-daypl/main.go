package main

import (
	"flag"

	"github.com/alpacahq/gobroker/workers/snapshot"

	"github.com/alpacahq/gobroker/utils/backfiller"
	"github.com/alpacahq/gobroker/utils/initializer"
)

func main() {
	initializer.Initialize()

	frm := flag.String("frm", "2018/04/17", "--date 2017/07/17")
	to := flag.String("to", "2018/04/20", "--date 2017/07/17")
	flag.Parse()

	filler, err := backfiller.New(*frm, *to)
	if err != nil {
		panic(err)
	}

	for filler.Next() {
		snapshot.ProcessSnapshot(filler.Value())
	}
}
