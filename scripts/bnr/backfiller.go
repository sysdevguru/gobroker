package main

import (
	"bytes"
	"flag"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/alpacahq/gobroker/s3man"
	"github.com/alpacahq/gobroker/sod/files"
	"github.com/alpacahq/gobroker/utils/initializer"
	"github.com/alpacahq/gobroker/workers/backup"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/log"
)

var (
	start   = flag.String("start", clock.Now().Add(-24*time.Hour).Format("2006-01-02"), "date to start backfilling from")
	monthly = flag.Bool("monthly", false, "run the monthly backfill")
	weekly  = flag.Bool("weekly", false, "run the weekly backfill")
	daily   = flag.Bool("daily", false, "run the daily backfill")
	sod     = flag.Bool("sod", false, "run the sod file backfill")
	sync    = flag.Bool("sync", false, "run the sync (S3 -> Egnyte)")
)

func init() {
	initializer.Initialize()
	flag.Parse()
}

func main() {
	central, _ := time.LoadLocation("America/Chicago")
	t, _ := time.ParseInLocation("2006-01-02", *start, central)

	if *monthly {
		asOf := time.Date(t.Year(), t.Month(), 0, 0, 0, 0, 0, central)

		log.Info("beginning monthly backfill", "asOf", asOf)

		for {
			s := time.Now()

			if asOf.After(s) {
				break
			}

			backup.WorkMonthly(asOf)

			log.Info(
				"month backfilled",
				"date", asOf.Format("2006-01"),
				"elapsed", time.Now().Sub(s))

			asOf = asOf.AddDate(0, 1, 0)
		}
	}

	if *weekly {
		asOf := t

		if asOf.Weekday() != time.Sunday {
			if asOf.After(t) {
				asOf = asOf.AddDate(0, 0, -int(asOf.Weekday()))
			} else {
				asOf = asOf.AddDate(0, 0, int(time.Saturday)-int(asOf.Weekday()))
			}
		}

		log.Info("beginning weekly backfill", "asOf", asOf)

		for {
			s := time.Now()

			if asOf.After(s) {
				break
			}

			backup.WorkWeekly(asOf)

			log.Info(
				"week backfilled",
				"date", asOf.Format("2006-01-02"),
				"elapsed", time.Now().Sub(s),
			)

			asOf = asOf.AddDate(0, 0, 7)
		}
	}

	if *daily {
		log.Info("beginning daily backfill", "asOf", t)

		asOf := t

		for {
			s := time.Now()

			if asOf.After(s) {
				break
			}

			if calendar.IsMarketDay(asOf) {
				backup.WorkDaily(asOf)

				log.Info(
					"day backfilled",
					"date", asOf.Format("2006-01-02"),
					"elapsed", time.Now().Sub(s),
				)
			}

			asOf = asOf.AddDate(0, 0, 1)
		}
	}

	if *sod {
		log.Info("beginning sod file backfill", "asOf", t)

		asOf := t

		for {
			s := time.Now()

			if asOf.After(s) {
				break
			}

			if calendar.IsMarketDay(asOf) {
				backupSoD((&files.CashActivityReport{}).ExtCode(), "money_movements", asOf)
				backupSoD((&files.TradeActivityReport{}).ExtCode(), "trades", asOf)
				log.Info(
					"sod backfilled",
					"date", asOf.Format("2006-01-02"),
					"elapsed", time.Now().Sub(s),
				)
			}

			asOf = asOf.AddDate(0, 0, 1)
		}
	}

	if *sync {
		now := clock.Now()

		log.Info("beginning S3 -> Egnyte sync", "asOf", now)

		backup.Sync(now)
	}
}

func backupSoD(code, category string, asOf time.Time) {
	s3 := s3man.New()

	s3Path := fmt.Sprintf(
		"/apex/download/%v/%v/APCA.csv",
		asOf.Format("20060102"),
		code,
	)

	buf, err := s3.DownloadInMemory(s3Path)
	if err != nil && !strings.Contains(err.Error(), "status code: 404") {
		log.Error(
			"failed to pull sod file from S3",
			"date", asOf.Format("2006-01-02"),
			"file_code", code,
			"error", err)
		return
	}

	booksAndRecordsPath := path.Join(
		"books_and_records",
		category,
		asOf.Format("20060102"),
		code,
		"APCA.csv",
	)

	if err = s3.Upload(bytes.NewReader(buf), booksAndRecordsPath); err != nil {
		log.Error(
			"failed to upload to egnyte",
			"date", asOf.Format("2006-01-02"),
			"file_code", code,
			"error", err)
	}
}
