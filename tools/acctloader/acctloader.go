package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/alpacahq/gobroker/service/account"
	"github.com/alpacahq/gobroker/service/ownerdetails"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/env"
	"github.com/gofrs/uuid"
	"gopkg.in/urfave/cli.v1"
	"gopkg.in/yaml.v2"
)

func main() {
	env.RegisterDefault("BROKER_SECRET", "fd0bxOTg7Q5qxISYKvdol0FBWnAaFgsP")

	app := cli.NewApp()
	app.Name = "acctloader"
	app.Usage = "Create a single account & load account_details from yaml file"
	app.ArgsUsage = "<yaml_file>"
	app.Flags = []cli.Flag{
		&cli.StringFlag{Name: "id,i"},
		&cli.StringFlag{Name: "data,d"},
		&cli.StringFlag{Name: "email,e"},
	}
	app.Action = func(c *cli.Context) (err error) {
		flag.Lookup("logtostderr").Value.Set("true")
		numArgs := 1
		yamlData := c.String("data")
		if yamlData != "" {
			numArgs = 0
		}
		if len(c.Args()) < numArgs {
			cli.ShowAppHelpAndExit(c, 0)
			return nil
		}
		if yamlData == "" {
			fileName := c.Args().Get(0)
			file, err := os.Open(fileName)
			if err != nil {
				return cli.NewExitError(err.Error(), 1)
			}
			data, err := ioutil.ReadAll(file)
			if err != nil {
				return cli.NewExitError(err.Error(), 1)
			}
			yamlData = string(data)
		}
		patches := map[string]interface{}{}
		if err := yaml.Unmarshal([]byte(yamlData), &patches); err != nil {
			return cli.NewExitError(err.Error(), 1)
		}
		email := c.String("email")
		var id uuid.UUID
		if email != "" {
			tx := db.Serializable()
			srv := account.Service().WithTx(tx)

			acct, err := srv.Create(email, uuid.Must(uuid.NewV4()))
			if err != nil {
				tx.Rollback()
				return cli.NewExitError(err.Error(), 1)
			}
			tx.Commit()
			fmt.Printf("Account: %v created\n", acct.ID)
			id, err = uuid.FromString(acct.ID)
		} else {
			id, err = uuid.FromString(c.String("id"))
		}
		if err != nil {
			return cli.NewExitError(err.Error(), 1)
		}
		tx := db.Serializable()
		detailSrv := ownerdetails.Service().WithTx(tx)

		if _, err = detailSrv.Patch(id, patches); err != nil {
			return cli.NewExitError(err.Error(), 1)
		}
		fmt.Printf("%v\n", patches)

		return nil
	}

	app.Run(os.Args)
}
