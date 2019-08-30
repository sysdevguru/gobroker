package main

import (
	"os"

	"github.com/alpacahq/gobroker/integration/testop"
	"github.com/alpacahq/gobroker/utils"
	"github.com/alpacahq/gopaca/log"

	cli "gopkg.in/urfave/cli.v1"
)

func main() {

	app := cli.NewApp()
	app.Name = "acctloader"
	app.Action = func(c *cli.Context) (err error) {

		if !utils.Dev() {
			log.Fatal("must run in DEV mode")
			return
		}

		if err := testop.Setup(); err != nil {
			return cli.NewExitError(err.Error(), 1)
		}
		return nil
	}

	app.Run(os.Args)
}
