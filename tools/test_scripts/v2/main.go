package main

func main() {

}

// import (
// 	"encoding/json"
// 	"flag"
// 	"fmt"
// 	"io/ioutil"
// 	"os"
// 	"strings"

// 	"github.com/alpacahq/gobroker/service/admin/account"
// 	"github.com/alpacahq/gobroker/service/admin/ops"
// 	"github.com/alpacahq/gobroker/tools/test_scripts/v2/suites"
// 	"github.com/alpacahq/gopaca/clock"
// 	"github.com/alpacahq/gopaca/db"
// 	"github.com/alpacahq/gopaca/env"
// 	"github.com/alpacahq/gopaca/log"
// )

// // TradePosting-  you will need 3 days of successful trade submission.  Maximum of 10 execution/allocation each round and when each round is sent the trades will be validated by our P&S department.  Once a PASS/FAIL is given for that round the next round may be sent over.

// // TODO: trading

// var tradeOnly = flag.Bool("tradeOnly", false, "skip atlas and sentinel portions")
// var accountLogJSON = flag.String("acctLogJSON", "", "path to json file w/ previously built AccountLog")
// var fundingLogJSON = flag.String("fundLogJSON", "", "path to json file w/ previously built FundingLog")

// func init() {
// 	setupEnv()

// 	flag.Parse()

// 	clock.Set()
// }

// func main() {
// 	// create an administrator
// 	var err error
// 	tx := db.Begin()
// 	service := account.Service().WithTx(tx)

// 	adminEmail := fmt.Sprintf("%v@admin.com", clock.Now().Unix())
// 	suites.Administrator, err = service.Create(adminEmail, "admin", "Test Admin")
// 	if err != nil {
// 		tx.Rollback()
// 		log.Fatal(err.Error())
// 	} else {
// 		tx.Commit()
// 	}

// 	if !*tradeOnly {
// 		// account opening related tasks
// 		if err = atlas(); err != nil {
// 			log.Fatal(err.Error())
// 		}

// 		// funding related tasks
// 		if err = sentinel(); err != nil {
// 			log.Fatal(err.Error())
// 		}
// 	}

// 	// trade related tasks
// 	if err = braggart(); err != nil {
// 		log.Fatal(err.Error())
// 	}
// }

// func atlas() (err error) {

// 	if accountLogJSON == nil || *accountLogJSON == "" {
// 		// create the required accounts
// 		suites.AcctLog, err = suites.CreateAccounts()
// 		if err != nil {
// 			return
// 		}
// 		// store account log to JSON file
// 		file, err := os.OpenFile(fmt.Sprintf(
// 			"acct_log_%v.json", clock.Now().Unix()),
// 			os.O_CREATE|os.O_WRONLY,
// 			os.ModeAppend)
// 		if err != nil {
// 			log.Fatal(err.Error())
// 		}
// 		buf, err := json.Marshal(suites.AcctLog)
// 		if err != nil {
// 			log.Fatal(err.Error())
// 		}
// 		if _, err = file.Write(buf); err != nil {
// 			log.Fatal(err.Error())
// 		}
// 		if err = file.Close(); err != nil {
// 			log.Fatal(err.Error())
// 		}
// 	} else {
// 		// read account log from JSON file
// 		loadAcctLog()
// 	}

// 	// // wait for them to reach their expected states
// 	if err = suites.AcctLog.Verify(); err != nil {
// 		return
// 	}

// 	// // update accounts
// 	if err = suites.AcctLog.UpdateAccounts(); err != nil {
// 		return
// 	}

// 	// // approve ACTION_REQUIRED accounts
// 	if err = suites.AcctLog.ApproveActionRequired(suites.Administrator.IDAsUUID()); err != nil {
// 		return
// 	}

// 	// // snap and appeal suspended accounts
// 	if err = suites.AcctLog.SnapAndAppealSuspended(); err != nil {
// 		return
// 	}

// 	// // snap 407 letters for finra affiliated accounts
// 	if err = suites.AcctLog.Snap407Letters(); err != nil {
// 		return
// 	}
// 	return nil
// }

// func loadAcctLog() {
// 	file, err := os.Open(*accountLogJSON)
// 	if err != nil {
// 		log.Fatal(err.Error())
// 	}
// 	buf, err := ioutil.ReadAll(file)
// 	if err != nil {
// 		log.Fatal(err.Error())
// 	}
// 	if err = json.Unmarshal(buf, &suites.AcctLog); err != nil {
// 		log.Fatal(err.Error())
// 	}
// 	if err = file.Close(); err != nil {
// 		log.Fatal(err.Error())
// 	}
// }

// func sentinel() (err error) {
// 	tx := db.Begin()
// 	srv := ops.Service().WithTx(tx)

// 	for _, acctID := range suites.AcctLog.Standard {
// 		if err = srv.Approve(acctID); err != nil && !strings.Contains(err.Error(), "account is already approved") {
// 			tx.Rollback()
// 			return
// 		}
// 	}
// 	tx.Commit()

// 	if fundingLogJSON == nil || *fundingLogJSON == "" {
// 		// create the required relationships
// 		suites.FundLog = &suites.FundingLog{}
// 		if suites.FundLog.Relationships, err = suites.CreateRelationships(suites.AcctLog.Standard[0:10]); err != nil {
// 			return
// 		}

// 		// store funding log to JSON file
// 		file, err := os.OpenFile(fmt.Sprintf(
// 			"fund_log_%v.json", clock.Now().Unix()),
// 			os.O_CREATE|os.O_WRONLY,
// 			os.ModeAppend)
// 		if err != nil {
// 			log.Fatal(err.Error())
// 		}
// 		buf, err := json.Marshal(suites.AcctLog)
// 		if err != nil {
// 			log.Fatal(err.Error())
// 		}
// 		if _, err = file.Write(buf); err != nil {
// 			log.Fatal(err.Error())
// 		}
// 		if err = file.Close(); err != nil {
// 			log.Fatal(err.Error())
// 		}
// 	} else {
// 		// read funding log from JSON file
// 		loadFundLog()
// 	}

// 	// make deposits
// 	if suites.FundLog.Deposits, err = suites.MakeDeposits(suites.AcctLog.Standard[0:10], suites.FundLog.Relationships); err != nil {
// 		return
// 	}

// 	// make withdrawals
// 	if suites.FundLog.Withdrawals, err = suites.MakeWithdrawals(suites.AcctLog.Standard[0:10], suites.FundLog.Relationships); err != nil {
// 		return
// 	}

// 	// cancel a relationship
// 	err = suites.CancelRelationship(suites.AcctLog.Standard[0], suites.FundLog.Relationships[0])
// 	if err != nil {
// 		return err
// 	}
// 	suites.FundLog.CanceledRelationship = suites.FundLog.Relationships[0]

// 	// cancel a transfer (creating a new one to do so)
// 	transferID, err := suites.CancelTransfer(suites.AcctLog.Standard[1], suites.FundLog.Relationships[1])
// 	if err != nil {
// 		return err
// 	}
// 	suites.FundLog.CanceledTransfer = *transferID

// 	// simulate ACH return
// 	err = suites.SimReturn(suites.FundLog.Withdrawals[0])
// 	if err != nil {
// 		return err
// 	}
// 	suites.FundLog.ReturnTransfer = suites.FundLog.Withdrawals[0]

// 	// simulate NOC
// 	err = suites.SimNOC(suites.FundLog.Withdrawals[1])
// 	if err != nil {
// 		return err
// 	}
// 	suites.FundLog.NOCTransfer = suites.FundLog.Withdrawals[1]

// 	// store funding log to JSON file
// 	file, err := os.OpenFile(fmt.Sprintf(
// 		"fund_log_%v.json", clock.Now().Unix()),
// 		os.O_CREATE|os.O_WRONLY,
// 		os.ModeAppend)
// 	if err != nil {
// 		log.Fatal(err.Error())
// 	}
// 	buf, err := json.Marshal(suites.AcctLog)
// 	if err != nil {
// 		log.Fatal(err.Error())
// 	}
// 	if _, err = file.Write(buf); err != nil {
// 		log.Fatal(err.Error())
// 	}
// 	return file.Close()
// }

// func loadFundLog() {
// 	file, err := os.Open(*fundingLogJSON)
// 	if err != nil {
// 		log.Fatal(err.Error())
// 	}
// 	buf, err := ioutil.ReadAll(file)
// 	if err != nil {
// 		log.Fatal(err.Error())
// 	}
// 	if err = json.Unmarshal(buf, &suites.FundLog); err != nil {
// 		log.Fatal(err.Error())
// 	}
// 	if err = file.Close(); err != nil {
// 		log.Fatal(err.Error())
// 	}
// }

// func braggart() error {
// 	loadAcctLog()
// 	if err := suites.PostBuys(suites.AcctLog.Standard[10:20]); err != nil {
// 		return err
// 	}
// 	return suites.PostSells(suites.AcctLog.Standard[10:20])
// 	// return suites.VerifyOrdersPosted(suites.AcctLog.Standard[10:20], orderIDs)
// 	// return nil
// }

// func setupEnv() {
// 	env.RegisterDefault("APEX_USER", "apex_api")
// 	env.RegisterDefault("APEX_ENTITY", "correspondent.apca")
// 	env.RegisterDefault(
// 		"APEX_SECRET",
// 		"j7VRz7Z91IOi4tT39vtVEY4rXn3_R4IUFZw7ubdM72aSUZ05Vo1Dm02VUuVlkrLKxzgHGupuiPs8lnnFc1K0xA",
// 	)
// 	env.RegisterDefault("APEX_CUSTOMER_ID", "101758")
// 	env.RegisterDefault("APEX_ENCRYPTION_KEY", "XIQIudheJi01Dk4o")
// 	env.RegisterDefault("APEX_URL", "https://uat-api.apexclearing.com")
// 	env.RegisterDefault("APEX_WS_URL", "https://uatwebservices.apexclearing.com")
// 	env.RegisterDefault("APEX_FIRM_CODE", "48")
// 	env.RegisterDefault("APEX_SFTP_USER", "apca_uat")
// 	env.RegisterDefault("APEX_RSA", "id_rsa_apca_uat")
// 	env.RegisterDefault("APEX_BRANCH", "3AP")
// 	env.RegisterDefault("APEX_REP_CODE", "APA")
// 	env.RegisterDefault("APEX_CORRESPONDENT_CODE", "APCA")
// }
