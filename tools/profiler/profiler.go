package main

import (
	"fmt"
	"os"
	"runtime/pprof"

	"github.com/alpacahq/gobroker/utils/initializer"
)

func init() {
	initializer.Initialize()
}

func main() {
	f, err := os.Create("cpu.prof")
	if err != nil {
		panic(err)
	}

	if err = pprof.StartCPUProfile(f); err != nil {
		panic(err)
	}

	fmt.Println("profile started...")

	for i := 0; i < 1000; i++ {
		if i%10 == 0 {
			fmt.Printf("%v iterations...", i)
		}
		testFunc()
	}

	fmt.Println("done.")

	pprof.StopCPUProfile()

	f.Close()
}

// add your code to profile here
func testFunc() {
	// accountID, err := uuid.FromString("c6a0aebb-107f-460d-92b3-238bb2c25921")
	// if err != nil {
	// 	panic(err)
	// }

	// // start a transaction to lock on account so the result
	// // is consistent with what is in the DB when the API
	// // is called by a user.
	// tx := db.Begin()

	// srv := account.Service().WithTx(tx)
	// srv.SetForUpdate()

	// acct, err := srv.GetByID(accountID)

	// if err != nil {
	// 	tx.Rollback()
	// 	panic(err)
	// }

	// pSrv := position.Service().WithTx(tx)

	// positions, err := pSrv.List(accountID)
	// if err != nil {
	// 	tx.Rollback()
	// 	panic(err)
	// }

	// balances, err := acct.Balances(
	// 	tx,
	// 	tradingdate.Last(clock.Now()).MarketOpen())

	// if err != nil {
	// 	tx.Rollback()
	// 	panic(err)
	// }

	// portfolioValue := balances.Cash
	// for _, p := range positions {
	// 	portfolioValue = portfolioValue.Add(p.MarketValue)
	// }

	// tx.Commit()
}
