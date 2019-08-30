package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"time"

	"github.com/alpacahq/apex"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/env"
	"github.com/shopspring/decimal"
	"github.com/urfave/cli"
)

func init() {
	flag.Parse()
	clock.Set()
}

func callGet(c *cli.Context, uri string) error {

	var body []byte
	err := apex.Client().Call(uri, "GET", nil, &body)
	if err != nil {
		return err
	}
	fmt.Println(string(body))
	return nil
}

func main() {
	// shell to run any test methods for querying Apex. Setting env vars manually here for sandbox.
	os.Setenv("APEX_USER", "apex_api")
	os.Setenv("APEX_ENTITY", "correspondent.apca")
	os.Setenv(
		"APEX_SECRET",
		"j7VRz7Z91IOi4tT39vtVEY4rXn3_R4IUFZw7ubdM72aSUZ05Vo1Dm02VUuVlkrLKxzgHGupuiPs8lnnFc1K0xA",
	)
	os.Setenv("APEX_CUSTOMER_ID", "101758")
	os.Setenv("APEX_ENCRYPTION_KEY", "XIQIudheJi01Dk4o")
	os.Setenv("APEX_URL", "https://uat-api.apexclearing.com")
	os.Setenv("APEX_WS_URL", "https://uatwebservices.apexclearing.com")
	os.Setenv("APEX_FIRM_CODE", "48")
	os.Setenv("APEX_SFTP_USER", "apca_uat")
	os.Setenv("APEX_RSA", "id_rsa_apca_uat")
	os.Setenv("APEX_BRANCH", "3AP")
	os.Setenv("APEX_REP_CODE", "APA")
	os.Setenv("APEX_CORRESPONDENT_CODE", "APCA")

	// PROD - DO NOT UN-COMMENT UNLESS YOU KNOW WHAT YOU ARE DOING!!!
	// os.Setenv("APEX_USER", "apex_api")
	// os.Setenv("APEX_ENTITY", "correspondent.apca")
	// os.Setenv(
	// 	"APEX_SECRET",
	// 	"zfI64zXwZhzdCv4ZhM72-e_wVb4VW1_3XSd2m41b2ZeIXVCNPKUrZcE8iUtOOLOYUGZA2S-0l9yfXbdBPwwudA",
	// )
	// os.Setenv("APEX_CUSTOMER_ID", "101498")
	// os.Setenv("APEX_ENCRYPTION_KEY", "pYpwB8ymSLMLjeSE")
	// os.Setenv("APEX_URL", "https://api.apexclearing.com")
	// os.Setenv("APEX_WS_URL", "https://webservices.apexclearing.com")
	// os.Setenv("APEX_FIRM_CODE", "10")
	// os.Setenv("APEX_SFTP_USER", "apca")
	// os.Setenv("APEX_RSA", "apca")
	// os.Setenv("APEX_BRANCH", "3AP")
	// os.Setenv("APEX_REP_CODE", "APA")
	// os.Setenv("APEX_CORRESPONDENT_CODE", "APCA")

	app := cli.NewApp()
	app.Commands = []cli.Command{
		{
			Name: "Authenticate",
			Action: func(c *cli.Context) error {
				if err := apex.Client().Authenticate(); err != nil {
					return err
				}
				fmt.Println("JWT: ", apex.Client().JWT)
				return nil
			},
		},
		// https://github.com/apexclearing/api-documentation/blob/master/atlas/atlas_account_request_api.md#get-account-request-by-id
		{
			Name: "GetAccountRequest",
			Action: func(c *cli.Context) error {
				requestId := c.Args().Get(0)
				uri := fmt.Sprintf(
					"%s/atlas/api/v2/account_requests/%s",
					env.GetVar("APEX_URL"),
					requestId,
				)
				return callGet(c, uri)
			},
		},
		// https://github.com/apexclearing/api-documentation/blob/master/atlas/atlas_account_request_api.md#list-account-requests
		{
			Name: "ListAccountRequests",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "startDateTime",
				},
				&cli.StringFlag{
					Name: "endDateTime",
				},
				&cli.StringFlag{
					Name: "account",
				},
				&cli.StringFlag{
					Name: "status",
				},
				&cli.IntFlag{
					Name: "limit",
				},
			},
			Action: func(c *cli.Context) error {
				v := url.Values{}
				v.Set("correspondent", "APCA")
				if s := c.String("startDateTime"); s != "" {
					v.Set("startDateTime", s)
				}
				if s := c.String("endDateTime"); s != "" {
					v.Set("endDateTime", s)
				}
				if s := c.String("account"); s != "" {
					v.Set("account", s)
				}
				if s := c.String("status"); s != "" {
					v.Set("status", s)
				}
				if s := c.String("limit"); s != "" {
					v.Set("limit", s)
				}
				uri := fmt.Sprintf(
					"%s/atlas/api/v2/account_requests?%s",
					env.GetVar("APEX_URL"),
					v.Encode(),
				)
				return callGet(c, uri)
			},
		},
		// https://github.com/apexclearing/api-documentation/blob/master/sketch/sketch_investigations.md#sketch-investigation-api-endpoints
		{
			Name: "GetSketchInvestigation",
			Action: func(c *cli.Context) error {
				id := c.Args().Get(0)
				uri := fmt.Sprintf(
					"%s/sketch/api/v1/investigations/%s",
					env.GetVar("APEX_URL"),
					id,
				)
				return callGet(c, uri)
			},
		},
		// https://github.com/apexclearing/api-documentation/blob/master/sketch/sketch_investigations.md#update-the-state-of-an-investigation
		{
			Name: "AppealSketchInvestigation",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "appealBody",
				},
			},
			Action: func(c *cli.Context) error {
				id := c.Args().Get(0)
				uri := fmt.Sprintf(
					"%s/sketch/api/v1/investigations/%s?action=APPEALED",
					env.GetVar("APEX_URL"),
					id,
				)
				appealBodyString := c.String("appealBody")
				appealBody := map[string]interface{}{}
				if err := json.Unmarshal([]byte(appealBodyString), &appealBody); err != nil {
					return err
				}
				var body []byte
				err := apex.Client().Call(uri, "PUT", appealBody, &body)
				fmt.Println(string(body))
				return err
			},
		},
		{
			Name: "GetAccountInfo",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "account",
				},
			},
			Action: func(c *cli.Context) error {
				acct := c.String("account")

				info, err := apex.Client().AccountOwnerInfo(acct)
				if err != nil {
					return err
				}

				fmt.Printf("Account Info: %v\n", info)
				return nil
			},
		},
		{
			Name: "GetAccount",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "account",
				},
			},
			Action: func(c *cli.Context) error {
				acct := c.String("account")

				info, err := apex.Client().AccountInfo(acct)
				if err != nil {
					return err
				}

				fmt.Printf("Account Info: %v\n", info)
				for _, accountInfo := range *info {
					fmt.Printf("Account Number: %v\n", *accountInfo.AccountNumber)
					fmt.Printf("Account Title: %v\n", *accountInfo.AccountTitle)
					fmt.Printf("Account Names: %v\n", accountInfo.AccountNames)
					fmt.Printf("Account Address: %v\n", *accountInfo.AccountAddress)
					fmt.Printf("Account Type: %v\n", *accountInfo.AccountType)
					fmt.Printf("Last 4 TIN: %v\n", *accountInfo.Last4TIN)
					fmt.Printf("Office Code: %v\n", *accountInfo.OfficeCode)
					fmt.Printf("Rep Code: %v\n", *accountInfo.RepCode)
					fmt.Printf("Phone Numbers: %v\n", *accountInfo.PhoneNumbers[0].PhoneNumber)
				}
				return nil
			},
		},
		{
			Name: "GetALE",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "topic",
				},
			},
			Action: func(c *cli.Context) error {
				topic := apex.ALETopic(c.String("topic"))
				// topic := apex.SentinelAchRelationshipStatus
				// Set HighWatermark: 10420000 on Dec 05, 2018 to see recent messages
				q := apex.ALEQuery{
					HighWatermark: 0,
					Since:         clock.Now().Add(-time.Hour * 24 * 30),
				}
				aleMsgs := apex.Client().ALE(topic, q)
				fmt.Printf("%v\n", aleMsgs)
				for _, msg := range aleMsgs {
					fmt.Printf("%v\n", msg)
				}
				return nil
			},
		},
		{
			Name: "CreateRelationship",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "relationshipJSON",
				},
			},
			Action: func(c *cli.Context) error {
				ach := apex.ACHRelationship{}
				data, err := ioutil.ReadFile(c.String("relationshipJSON"))
				if err != nil {
					return err
				}
				if err = json.Unmarshal(data, &ach); err != nil {
					return err
				}
				rel, err := apex.Client().CreateRelationship(ach)
				if err != nil {
					return fmt.Errorf("failed to create new ACH relationship (%v)", err)
				}
				fmt.Println("relationship created: ", rel)
				return nil
			},
		},
		{
			Name: "AmountAvailable",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "id",
				},
			},
			Action: func(c *cli.Context) error {
				aa, err := apex.Client().AmountAvailable(c.String("id"))
				if err != nil {
					return fmt.Errorf("failed to get amount available for %v (%v)", c.String("id"), err)
				}
				fmt.Printf("amount available for %v: %v\n", c.String("id"), aa)
				return nil
			},
		},
		{
			Name: "ApproveTransfer",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "id",
				},
			},
			Action: func(c *cli.Context) error {
				err := apex.Client().SimulateTransferApproval(c.String("id"))
				if err != nil {
					return fmt.Errorf("failed to approve transfer %v - Error: %v", c.String("id"), err)
				}
				fmt.Printf("transfer %v approved!\n", c.String("id"))
				return nil
			},
		},
		{
			Name: "CancelRelationship",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "id",
				},
			},
			Action: func(c *cli.Context) error {
				rel, err := apex.Client().CancelRelationship(c.String("id"), "test")
				if err != nil {
					return fmt.Errorf("failed to cancel relationship %v (%v)", c.String("id"), err)
				}
				fmt.Println("relationship canceled: ", rel)
				return nil
			},
		},
		{
			Name: "GetRelationship",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "id",
				},
			},
			Action: func(c *cli.Context) error {
				rel, err := apex.Client().GetRelationship(c.String("id"))
				if err != nil {
					return fmt.Errorf("failed to get relationship %v (%v)", c.String("id"), err)
				}
				fmt.Println("relationship: ", rel)
				return nil
			},
		},
		{
			Name: "Transfer",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "direction",
				},
				&cli.StringFlag{
					Name: "transferID",
				},
				&cli.StringFlag{
					Name: "amount",
				},
				&cli.StringFlag{
					Name: "relationshipID",
				},
			},
			Action: func(c *cli.Context) error {
				direction := apex.TransferDirection(c.String("direction"))
				amount, err := decimal.NewFromString(c.String("amount"))
				if err != nil {
					return err
				}
				transfer := apex.ACHTransfer{
					ID:             c.String("transferID"),
					Amount:         amount,
					RelationshipID: c.String("relationshipID"),
				}
				if direction == apex.Outgoing {
					transfer.DisbursementType = "PARTIAL_BALANCE"
				}
				xfer, err := apex.Client().Transfer(direction, transfer)
				if err != nil {
					return fmt.Errorf(
						"failed to transfer %v %v using %v:%v (%v)",
						direction,
						amount,
						c.String("relationshipID"),
						c.String("transferID"),
						err,
					)
				}
				fmt.Println("transferred: ", xfer)
				return nil
			},
		},
		{
			Name: "CancelTransfer",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "id",
				},
			},
			Action: func(c *cli.Context) error {
				resp, err := apex.Client().CancelTransfer(c.String("id"), "test")
				if err != nil {
					return fmt.Errorf("failed to cancel transfer %v (%v)", c.String("id"), err)
				}
				fmt.Println("canceled: ", resp)
				return nil
			},
		},
		{
			Name: "TransferStatus",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "id",
				},
			},
			Action: func(c *cli.Context) error {
				resp, err := apex.Client().TransferStatus(c.String("id"))
				if err != nil {
					return fmt.Errorf("failed to get transfer status for %v (%v)", c.String("id"), err)
				}
				fmt.Println("transfer: ", resp)
				return nil
			},
		},
		{
			Name: "ListTransactions",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "account",
				},
			},
			Action: func(c *cli.Context) error {
				resp, err := apex.Client().ListTransactions(apex.BraggartTransactionQuery{
					Correspondent: "APCA",
				})
				if err != nil {
					return fmt.Errorf("failed to list braggart transactions for: %v", c.String("account"))
				}
				fmt.Println("transactions: ", resp, err)
				return nil
			},
		},
		{
			Name: "GetDocuments",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "id",
				},
				&cli.StringFlag{
					Name: "start",
				},
				&cli.StringFlag{
					Name: "end",
				},
				&cli.StringFlag{
					Name: "docType",
				},
			},
			Action: func(c *cli.Context) error {
				start, err := time.Parse("2006-01-02", c.String("start"))
				if err != nil {
					return err
				}
				end, err := time.Parse("2006-01-02", c.String("end"))
				if err != nil {
					return err
				}
				resp, err := apex.Client().GetDocuments(
					c.String("id"),
					start,
					end,
					apex.DocumentTypeFromString(c.String("docType")),
				)
				if err != nil {
					return fmt.Errorf("failed to retrieve documents for %v (%v)", c.String("id"), err)
				}
				fmt.Println("documents: ", resp)
				return nil
			},
		},
		{
			Name: "UploadSnap",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "file",
				},
				&cli.StringFlag{
					Name: "name",
				},
			},
			Action: func(c *cli.Context) error {
				f, err := ioutil.ReadFile(c.String("file"))
				if err != nil {
					return err
				}

				snapID, err := apex.Client().PostSnap(f, c.String("name"), apex.ID_DOCUMENT)
				if err != nil {
					return err
				}

				fmt.Printf("Posted snap: %v\n", snapID)

				return nil
			},
		},
		{
			Name: "GetSnap",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "id",
				},
			},
			Action: func(c *cli.Context) error {
				preview, err := apex.Client().GetSnap(c.String("id"))
				if err != nil {
					return err
				}

				fmt.Printf("Preview: %v\n", *preview)

				return nil
			},
		},
		{
			Name: "GetMicroDepositAmounts",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "id",
				},
			},
			Action: func(c *cli.Context) error {
				preview, err := apex.Client().SimulateMicroDepositAmount(c.String("id"))
				if err != nil {
					return err
				}

				fmt.Printf("Preview: Amount One - %v, Amount Two - %v\n", preview[0], preview[1])

				return nil
			},
		},
		{
			Name: "ApproveMicroDeposits",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "id",
				},
				&cli.StringFlag{
					Name: "amount_one",
				},
				&cli.StringFlag{
					Name: "amount_two",
				},
			},
			Action: func(c *cli.Context) error {
				amountOne, err := decimal.NewFromString(c.String("amount_one"))
				if err != nil {
					return err
				}
				amountTwo, err := decimal.NewFromString(c.String("amount_two"))
				if err != nil {
					return err
				}
				amounts := apex.MicroDepositAmounts{amountOne, amountTwo}
				preview, err := apex.Client().ApproveRelationship(c.String("id"), amounts)
				if err != nil {
					return err
				}

				if preview != nil {
					fmt.Printf("Preview: Status %v\n", preview.Status)
				}

				return nil
			},
		},
		{
			Name: "ReissueMicroDeposits",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "id",
				},
			},
			Action: func(c *cli.Context) error {
				err := apex.Client().ReissueMicroDeposits(c.String("id"))
				if err != nil {
					return err
				}

				fmt.Printf("Reissue Success\n")

				return nil
			},
		},
		{
			Name: "CloseAccount",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "apexAcctNum",
				},
			},
			Action: func(c *cli.Context) error {
				resp, err := apex.Client().CloseAccountRequest(c.String("apexAcctNum"))
				if err != nil {
					return fmt.Errorf("failed to close account %v (%v)", c.String("apexAcctNum"), err)
				}
				fmt.Println("apex account number: ", c.String("apexAcctNum"), "new id:", *resp.ID)
				return nil
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Println("ERROR: ", err)
	}
}
