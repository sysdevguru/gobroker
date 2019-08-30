package main

import (
	"flag"
	"fmt"

	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/s3man"
	"github.com/alpacahq/gobroker/sod/files"
	"github.com/alpacahq/gobroker/utils/backfiller"
	"github.com/alpacahq/gobroker/utils/date"
	"github.com/alpacahq/gobroker/utils/initializer"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/log"
)

func main() {
	initializer.Initialize()

	frm := flag.String("frm", "2018/04/17", "--date 2017/07/17")
	to := flag.String("to", "2018/05/20", "--date 2017/07/17")
	flag.Parse()

	filler, err := backfiller.New(*frm, *to)
	if err != nil {
		panic(err)
	}

	m := s3man.New()

	for filler.Next() {
		tradingday := filler.Value()

		f := files.BuyingPowerSummaryReport{}
		procDate := tradingday.MarketOpen().Format("2006-01-02")

		s3Path := fmt.Sprintf("/apex/download/%v/%v/APCA.csv", tradingday.MarketOpen().Format("20060102"), f.ExtCode())

		ok, err := m.Exists(s3Path)
		if err != nil {
			log.Debug("failed to find file on s3", "file", f.ExtCode(), "date", procDate)
			continue
		}

		if !ok {
			log.Debug("file is missing", "file", f.ExtCode(), "date", procDate)
			return
		}

		buf, err := m.DownloadInMemory(s3Path)

		if err := files.Parse(buf, &f); err != nil {
			log.Warn("failed to parse start of day", "file", f.ExtCode(), "date", procDate)
			return
		}

		summaries := f.Value().Interface().([]files.SoDBuyingPowerSummary)

		for _, s := range summaries {
			if files.IsFirmAccount(s.AccountNumber) {
				continue
			}
			tx := db.DB().Begin()

			var acc models.Account
			q := tx.Where("apex_account = ?", s.AccountNumber).Find(&acc)
			if q.RecordNotFound() {
				log.Warn("account not found for cash backfill", "account", s.AccountNumber)
				tx.Rollback()
				continue
			}

			if q.Error != nil {
				log.Error("start of day database error", "file", f.ExtCode(), "error", q.Error)
				tx.Rollback()
				panic(q.Error)
			}

			var accCash models.Cash
			patch := models.Cash{AccountID: acc.ID, Date: date.DateOf(tradingday.MarketOpen())}
			if err := tx.FirstOrCreate(&accCash, patch).Error; err != nil {
				log.Error("start of day database error", "file", f.ExtCode(), "error", q.Error)
				tx.Rollback()
				panic(q.Error)
			}
			accCash.Value = s.TotalEquity.Sub(*s.PositionMarketValue)

			if err := tx.Save(&accCash).Error; err != nil {
				log.Error("start of day database error", "file", f.ExtCode(), "error", err)
				tx.Rollback()
				panic(err)
			}

			if err := tx.Commit().Error; err != nil {
				log.Error("start of day database error", "file", f.ExtCode(), "error", err)
				panic(err)
			}
		}
		log.Debug("done", "date", procDate, "accounts", len(summaries))
	}
}
