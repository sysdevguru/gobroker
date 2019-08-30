package main

import (
	"flag"
	"fmt"

	"github.com/alpacahq/gobroker/utils/initializer"
	"github.com/alpacahq/gobroker/utils/tradingdate"
	"github.com/alpacahq/gobroker/workers/snapshot"
)

func main() {
	initializer.Initialize()

	flag.Parse()
	flag.Lookup("logtostderr").Value.Set("true")

	now := tradingdate.Current()
	d := tradingdate.Current().DaysAgo(90)

	for {
		if d.Equals(now) {
			break
		}
		snapshot.ProcessSnapshot(d)
		fmt.Println("Done", d.String())
		d = d.Next()
	}

}
