package main

import (
	"flag"

	"github.com/alpacahq/gobroker/utils/initializer"
	"github.com/alpacahq/gobroker/utils/tradingdate"
	"github.com/alpacahq/gobroker/workers/snapshot"
)

func main() {
	initializer.Initialize()

	flag.Parse()
	flag.Lookup("logtostderr").Value.Set("true")

	now := tradingdate.Current()
	snapshot.ProcessSnapshot(now)
}
