package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/alpacahq/gobroker/sod/files"

	"github.com/alpacahq/gobroker/workers/sod"
	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/env"
	"github.com/alpacahq/gopaca/log"
)

var (
	dateFlag     = flag.String("date", clock.Now().Add(-24*time.Hour).Format("2006-01-02"), "Date to pull SoD files for")
	localDirFlag = flag.String("localDir", "", "local directory with sod files")
	sodFiles     = []files.SODFile{
		&files.AccountMaster{},
		&files.AmountAvailableDetailReport{},
		&files.BuyingPowerDetailReport{},
		&files.BuyingPowerSummaryReport{},
		&files.DividendReport{},
		&files.CashActivityReport{},
		&files.ElectronicCommPrefReport{},
		&files.MarginCallReport{},
		&files.ReturnedMailReport{},
		&files.MandatoryActionReport{},
		&files.SecurityMaster{},
		&files.EasyToBorrowReport{},
		&files.PositionReport{},
		&files.TradeActivityReport{},
	}
)

func init() {
	setupEnv()

	flag.Parse()

	clock.Set()
}

func main() {
	t, err := time.ParseInLocation("2006-01-02", *dateFlag, calendar.NY)
	if err != nil {
		t = clock.Now()
	}

	if *localDirFlag != "" {
		for _, file := range sodFiles {
			dirName := fmt.Sprintf(
				"%v/download/%v/%v/",
				*localDirFlag,
				t.Format("20060102"),
				file.ExtCode(),
			)

			fileInfos, err := ioutil.ReadDir(dirName)
			if err != nil {
				log.Warn("cannot read directory, skipping sod file", "directory", dirName, "file_code", file.ExtCode(), "error", err)
				continue
			}

			for _, fileInfo := range fileInfos {
				filePath := fmt.Sprintf("%v/%v", dirName, fileInfo.Name())
				buf, err := ioutil.ReadFile(filePath)
				if err != nil {
					log.Error("cannot read file, skipping sod file", "error", err, "file", filePath, "file_code", file.ExtCode())
					break
				}

				if err = files.Parse(buf, file); err != nil {
					log.Error("failed to parse sod file", "error", err, "file_code", file.ExtCode())
					break
				}

				processed, errors := file.Sync(t)

				log.Info("synced sod file", "processed", processed, "errors", errors)
			}
		}
	} else {
		sod.Work(t)
	}

}

func setupEnv() {
	env.RegisterDefault("APEX_USER", "apex_api")
	env.RegisterDefault("APEX_ENTITY", "correspondent.apca")
	env.RegisterDefault(
		"APEX_SECRET",
		"j7VRz7Z91IOi4tT39vtVEY4rXn3_R4IUFZw7ubdM72aSUZ05Vo1Dm02VUuVlkrLKxzgHGupuiPs8lnnFc1K0xA",
	)
	env.RegisterDefault("APEX_CUSTOMER_ID", "101758")
	env.RegisterDefault("APEX_ENCRYPTION_KEY", "XIQIudheJi01Dk4o")
	env.RegisterDefault("APEX_URL", "https://uat-api.apexclearing.com")
	env.RegisterDefault("APEX_WS_URL", "https://uatwebservices.apexclearing.com")
	env.RegisterDefault("APEX_FIRM_CODE", "48")
	env.RegisterDefault("APEX_SFTP_USER", "apca_uat")
	env.RegisterDefault("APEX_RSA", "id_rsa_apca_uat")
	env.RegisterDefault("APEX_BRANCH", "3AP")
	env.RegisterDefault("APEX_REP_CODE", "APA")
	env.RegisterDefault("APEX_CORRESPONDENT_CODE", "APCA")
}
