package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/alpacahq/gobroker/gbreg"
	"github.com/alpacahq/gobroker/utils/initializer"
	"github.com/alpacahq/gobroker/utils/tradingdate"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/gofrs/uuid"
	"github.com/golang/glog"
)

func main() {
	initializer.Initialize()

	id := flag.String("account_id", "", "account id")
	period := flag.String("period", "1M", "M1, M3, M6, 1A, all")
	timeframe := flag.String("timeframe", "1D", "5Min, 1D")
	flag.Parse()

	now := clock.Now()
	on := tradingdate.Current()

	acctId, err := uuid.FromString(*id)
	if err != nil {
		glog.Fatalf("Invalid account id")
	}

	tx := db.RepeatableRead()

	service := gbreg.Services.Portfolio().WithTx(tx)

	ret, err := service.GetHistory(acctId, on, calendar.RangeFreq(*timeframe), *period, &now)

	if err == nil {
		fmt.Println("LastdayClose", ret.BaseValue)
		fmt.Println("Columns", "ProfitLoss", "ProfitLossPctChange", "PortfolioValue")
		for i, a := range ret.Arrays[0].([]int64) {
			t := time.Unix(int64(a/1000), 0)
			var a1, a2, a3 string

			ar1 := ret.Arrays[1].([]*float64)[i]
			if ar1 != nil {
				a1 = fmt.Sprint(*ar1)
			} else {
				a1 = "nil"
			}

			ar2 := ret.Arrays[2].([]*float64)[i]
			if ar2 != nil {
				a2 = fmt.Sprint(*ar2)
			} else {
				a2 = "nil"
			}

			ar3 := ret.Arrays[3].([]*float64)[i]
			if ar3 != nil {
				a3 = fmt.Sprint(*ar3)
			} else {
				a3 = "nil"
			}
			fmt.Println(t, a1, a2, a3)
		}
	} else {
		fmt.Println(err)
	}
}
