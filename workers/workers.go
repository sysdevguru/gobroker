package main

import (
	"flag"
	"fmt"
	"math/rand"
	"path"
	"sync"
	"time"

	"github.com/alpacahq/gobroker/external/segment"
	"github.com/alpacahq/gobroker/external/slack"
	"github.com/alpacahq/gobroker/gbreg"
	"github.com/alpacahq/gobroker/mailer"
	"github.com/alpacahq/gobroker/models"
	"github.com/alpacahq/gobroker/models/enum"
	"github.com/alpacahq/gobroker/sod"
	"github.com/alpacahq/gobroker/utils"
	"github.com/alpacahq/gobroker/utils/gbevents"
	"github.com/alpacahq/gobroker/utils/initializer"
	"github.com/alpacahq/gobroker/utils/signalman"
	"github.com/alpacahq/gobroker/workers/account"
	"github.com/alpacahq/gobroker/workers/ale"
	"github.com/alpacahq/gobroker/workers/backup"
	"github.com/alpacahq/gobroker/workers/braggart"
	"github.com/alpacahq/gobroker/workers/funding"
	"github.com/alpacahq/gobroker/workers/gbtrade"
	"github.com/alpacahq/gobroker/workers/gc"
	sodWorker "github.com/alpacahq/gobroker/workers/sod"
	"github.com/alpacahq/gobroker/workers/trade"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/db"
	"github.com/alpacahq/gopaca/env"
	"github.com/alpacahq/gopaca/log"
	"github.com/jinzhu/gorm"
	"github.com/robfig/cron"
	"go.uber.org/zap/zapcore"
)

var (
	cronWg      sync.WaitGroup
	c           *cron.Cron
	tradeWorker *trade.TradeWorker
)

func shutdown() error {

	// stop crons so no new ones start
	if c != nil {
		c.Stop()
	}

	// wait for existing crons to finish
	cronWg.Wait()

	// stop the RMQ related tasks explicitly
	account.Stop()
	tradeWorker.Stop()

	// sleep a second to let things cleanup
	<-time.After(time.Second)
	return nil
}

func init() {
	rand.Seed(clock.Now().UTC().UnixNano())
	// set the clock
	clock.Set()

	// register env defaults
	initializer.Initialize()

	flag.Parse()

	// log errors to slack
	log.Logger().AddCallback(
		"gb-workers_slack_errors",
		zapcore.ErrorLevel,
		func(i interface{}) {
			msg := slack.NewServerError()
			msg.SetBody(i)
			slack.Notify(msg)
		},
	)

	// set deployment level on logger
	log.Logger().SetDeploymentLevel(env.GetVar("BROKER_MODE"))

	// handler initializers
	gbevents.RegisterSignalHandler()

	signalman.RegisterFunc("workers_shutdown", shutdown)
	signalman.Start()
}

func main() {
	// trade worker - no need for cron, it just runs in the background,
	// pulling from RMQ and updating things as necessary
	tradeWorker = gbtrade.NewTradeWorker()

	if utils.StandBy() {
		log.Info("starting in standby mode - no crons will be run")
		signalman.Wait()
		return
	}

	central, _ := time.LoadLocation("America/Chicago")
	c = cron.NewWithLocation(central)

	// account worker
	log.Info(
		"starting account worker",
		"interval",
		env.GetVar("ACCOUNT_WORKER_INTERVAL"))

	c.AddFunc(fmt.Sprintf("@every %v", env.GetVar("ACCOUNT_WORKER_INTERVAL")), func() {
		cronWg.Add(1)
		defer cronWg.Done()
		account.Work()
	})

	// funding worker
	log.Info(
		"starting funding worker",
		"interval",
		env.GetVar("FUNDING_WORKER_INTERVAL"))

	c.AddFunc(fmt.Sprintf("@every %v", env.GetVar("FUNDING_WORKER_INTERVAL")), func() {
		cronWg.Add(1)
		defer cronWg.Done()
		funding.Work()
	})

	// braggart worker
	log.Info(
		"starting braggart worker",
		"interval",
		env.GetVar("BRAGGART_WORKER_INTERVAL"))

	c.AddFunc(fmt.Sprintf("@every %v", env.GetVar("BRAGGART_WORKER_INTERVAL")), func() {
		cronWg.Add(1)
		defer cronWg.Done()
		braggart.Work()
	})

	// sod sync - tue-sat @ 6 AM central
	c.AddFunc("0 0 6 * * TUE-SAT", func() {
		cronWg.Add(1)
		defer cronWg.Done()

		now := clock.Now().In(central)
		log.Info("sod sync", "time", now)

		sodWorker.Work(clock.Now().In(central).Add(-24 * time.Hour))
	})

	// daily backup (accounts + order tickets + trade confirms) - every weekday just before midnight central
	c.AddFunc("0 59 23 * * MON-FRI", func() {
		cronWg.Add(1)
		defer cronWg.Done()

		asOf := clock.Now().In(central)

		if calendar.IsMarketDay(asOf) {
			log.Info("accounts, order tickets and trade confirms daily backup", "time", asOf)
			backup.WorkDaily(asOf)
		}
	})

	//Daily Segment Update
	c.AddFunc("@hourly", func() {
		cronWg.Add(1)
		defer cronWg.Done()

		log.Info("starting to send information to segment")

		// Segment status update will query all accounts and pass off to function that sends info to Segment
		accounts := []models.Account{}

		q := db.DB()

		q = q.
			Where("status != ?", enum.Onboarding).
			Preload("Owners").
			Preload("Owners.Details", "replaced_by IS NULL")

		if err := q.Find(&accounts).Error; err != nil && err != gorm.ErrRecordNotFound {
			log.Error("failed to find accounts", "error", err)
		}

		total := 0
		// Goes through all accounts and sends Segment information
		for _, acct := range accounts {
			err := segment.Identify(acct)
			if err != nil {
				if err != fmt.Errorf("account owner is nil") {
					log.Error("failed to append information to segment", "account", acct.ID, "error", err)
				}
			} else {
				total += 1
			}
		}

		log.Info("sent information to segment", "accounts", total)
	})

	// weekly backup (trade confirmations) - every sunday just before midnight central
	c.AddFunc("0 59 23 * * SUN", func() {
		cronWg.Add(1)
		defer cronWg.Done()

		now := clock.Now().In(central)

		log.Info("trade confirmations weekly backup", "time", now)
		backup.WorkWeekly(now)
	})

	// monthly backup (monthly statements) - second saturday of the month just before midnight central
	c.AddFunc("0 59 23 * * SAT", func() {
		cronWg.Add(1)
		defer cronWg.Done()

		now := clock.Now().In(central)

		// ensure it is the second saturday of the month
		if now.Day() > 7 && now.Day() < 15 {
			log.Info("egnyte monthly backup", "time", now)
			backup.WorkMonthly(now)
		}
	})

	// sync the S3 backups to egnyte monthly - every sunday @ noon
	c.AddFunc("0 0 12 1 * SUN", func() {
		cronWg.Add(1)
		defer cronWg.Done()

		now := clock.Now().In(central)

		log.Info("S3 backups to egnyte monthly sync", "time", now)
		backup.Sync(now)
	})

	// monthly settlement file (15th of every month midnight central)
	c.AddFunc("0 0 0 15 * *", func() {
		if utils.Prod() {
			cronWg.Add(1)
			defer cronWg.Done()

			y, m, _ := clock.Now().Add(-30 * calendar.Day).In(central).Date()

			fileName := fmt.Sprintf("/download/%d%02dBilling_9443AP.zip", y, m)

			sp := sod.SoDProcessor{}
			sp.InitClient()
			defer sp.Close()

			log.Info("downloading settlement file", "file", fileName)

			if err := sp.InitClient(); err != nil {
				log.Error("failed to init sftp client for monthly settlement email", "error", err)
				return
			}

			buf, err := sp.DownloadFile(fileName)
			if err != nil {
				log.Error("failed to download monthly settlement file", "error", err)
				return
			}

			mailer.SendMonthlySettlement(fmt.Sprintf("%02d/%d", m, y), path.Base(fileName), buf)
		}
	})

	// ALE worker
	c.AddFunc(fmt.Sprintf("@every %v", env.GetVar("ALE_WORKER_INTERVAL")), func() {
		cronWg.Add(1)
		defer cronWg.Done()
		ale.Work()
	})

	// Call GC worker every hour
	c.AddFunc("0 0 * * * *", func() {
		gc.Work()
	})

	authSync := func() {
		log.Info("auth cache syncing")

		start := time.Now()

		if err := gbreg.Services.AccessKey().WithTx(db.DB()).Sync(true); err != nil {
			log.Error("failed to sync auth cache", "error", err)
		}

		log.Info("auth cache synced", "elapsed", time.Now().Sub(start))
	}

	// Sync the auth cache every 30 minutes
	c.AddFunc("@every 30m", authSync)

	// run it immediately
	authSync()

	// queue the crons
	c.Start()

	// start gbevent listeners
	go func() { gbevents.RunForever() }()

	log.Info(
		"workers are live",
		"mode", env.GetVar("BROKER_MODE"),
		"clock", clock.Now())

	signalman.Wait()
}
