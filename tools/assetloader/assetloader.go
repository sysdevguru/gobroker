package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/alpacahq/gobroker/sod"
	"github.com/alpacahq/gobroker/utils/gbevents"
	"github.com/alpacahq/gobroker/utils/initializer"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/env"
)

var asOf = flag.String("asof", clock.Now().Add(-24*time.Hour).Format("2006-01-02"), "date to pull sod for")

func init() {
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

	flag.Parse()

	clock.Set()
}

func main() {
	initializer.Initialize()

	central, _ := time.LoadLocation("America/Chicago")
	t, _ := time.ParseInLocation("2006-01-02", *asOf, central)

	fmt.Println("pulling assets & fundamentals...")
	if err := (&sod.SoDProcessor{}).Pull(t, true, 0); err != nil {
		panic(err)
	}

	gbevents.TriggerEvent(&gbevents.Event{Name: gbevents.EventAssetRefreshed})

	fmt.Println("pull complete")
}
