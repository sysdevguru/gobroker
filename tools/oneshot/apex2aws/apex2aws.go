package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/alpacahq/gobroker/sod/backup"
	"github.com/alpacahq/gobroker/sod/files"
	"github.com/alpacahq/gobroker/utils/initializer"
	"github.com/alpacahq/gobroker/utils/tradingdate"
	"github.com/alpacahq/gopaca/calendar"
)

var Files = []files.SODFile{
	&files.AccountMaster{},
	&files.AmountAvailableDetailReport{},
	&files.BuyingPowerDetailReport{},
	&files.BuyingPowerSummaryReport{},
	&files.CashActivityReport{},
	&files.DividendReport{},
	&files.ElectronicCommPrefReport{},
	&files.EasyToBorrowReport{},
	&files.MarginCallReport{},
	&files.ReturnedMailReport{},
	&files.MandatoryActionReport{},
	&files.SecurityMaster{},
	&files.PositionReport{},
	&files.SecurityOverrideReport{},
	// &files.StockActivityReport{},  never produced by apex
	&files.TradeActivityReport{},
	&files.TradesMovedToErrorReport{},
	&files.VoluntaryActionReport{},
}

func main() {
	initializer.Initialize()

	frm := flag.String("frm", "", "--date 2017/07/17")
	to := flag.String("to", "", "--date 2017/07/17")
	flag.Parse()

	if *frm == "" {
		fmt.Println("frm is needed")
		return
	}
	if *to == "" {
		fmt.Println("to is needed")
		return
	}

	frmTime, err := time.ParseInLocation("2006/01/02", *frm, calendar.NY)
	if err != nil {
		fmt.Println(err)
		return
	}
	frmDate, err := tradingdate.New(frmTime)
	if err != nil {
		fmt.Println(err)
		return
	}
	toTime, err := time.ParseInLocation("2006/01/02", *to, calendar.NY)
	if err != nil {
		fmt.Println(err)
		return
	}
	toDate, err := tradingdate.New(toTime)
	if err != nil {
		fmt.Println(err)
		return
	}

	backup.Backup(*frmDate, *toDate)
}
