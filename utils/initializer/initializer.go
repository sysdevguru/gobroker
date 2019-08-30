package initializer

import (
	"log"

	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/clock"
	"github.com/alpacahq/gopaca/env"
)

// Initialize gobroker's required environment variables
// to their default values.
func Initialize() {
	// Broker
	env.RegisterDefault("BROKER_MODE", "DEV")
	env.RegisterDefault("BROKER_SECRET", "fd0bxOTg7Q5qxISYKvdol0FBWnAaFgsP")
	env.RegisterDefault("ADMIN_SECRET", "savBs0F2daUjSd9syVovH6gXN4MuGQos")
	env.RegisterDefault("START_TIME", clock.Now().In(calendar.NY).Format("2006-01-02 15:04"))
	env.RegisterDefault("LOG_LEVEL", "INFO")
	env.RegisterDefault("EMAILS_ENABLED", "TRUE")
	env.RegisterDefault("INSTANT_DEPOSIT_LIMIT", "0")
	env.RegisterDefault("BACKUP_PARALLELISM", "4")
	env.RegisterDefault("COGNITO_ENABLED", "TRUE")
	env.RegisterDefault("QUEUE_ORDERS", "FALSE")
	env.RegisterDefault("ACCOUNT_WORKER_INTERVAL", "5s")
	env.RegisterDefault("ALE_WORKER_INTERVAL", "1m")
	env.RegisterDefault("ALE_TIMEOUT", "10s")
	env.RegisterDefault("FUNDING_WORKER_INTERVAL", "1m")
	env.RegisterDefault("BRAGGART_WORKER_INTERVAL", "5s")
	env.RegisterDefault("PAPER_DB", "papertrader")
	env.RegisterDefault("STANDBY_MODE", "FALSE")

	// Cognito (development pool)
	env.RegisterDefault("COGNITO_REGION", "us-east-1")
	env.RegisterDefault("COGNITO_USER_POOL_ID", "us-east-1_NXqSmW0Or")
	env.RegisterDefault("COGNITO_CLIENT_ID", "2bh095j5b1qi2j0b4u2o9kiajr")

	// Postgres
	env.RegisterDefault("PGDATABASE", "gobroker")
	env.RegisterDefault("PGHOST", "127.0.0.1")
	env.RegisterDefault("PGUSER", "postgres")
	env.RegisterDefault("PGPASSWORD", "alpacas")

	// Plaid
	env.RegisterDefault("PLAID_PUBLIC_KEY", "e11605cec07e75aef2ce14c9f5b712")
	env.RegisterDefault("PLAID_SECRET", "e0346c3978d3dd1d0c3c417225bf4c")
	env.RegisterDefault("PLAID_CLIENT_ID", "59f7b8444e95b8782b00bc9b")
	env.RegisterDefault("PLAID_URL", "https://sandbox.plaid.com")

	// Apex
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

	// Segment
	env.RegisterDefault("SEGMENT_KEY", "xmCnxB6n2DZkLushN3ZN0tvHW3CUJx6D")

	// Egnyte
	env.RegisterDefault("EGNYTE_TOKEN", "zj686zemggu88nk98npdw6zp")
	env.RegisterDefault("EGNYTE_DOMAIN", "alpacadev.egnyte.com")

	// GoTrader
	env.RegisterDefault("TRADER_SECRET", "YYcaSjJqjgRFjXdUqari85Td8AltABt6")
	env.RegisterDefault("TRADER_URL", "http://127.0.0.1:5999")

	// if rmq queues are not specified, panic
	if env.GetVar("ORDER_REQUESTS_QUEUE") == "" {
		log.Fatal("invalid environment", "variable", "ORDER_REQUESTS_QUEUE")
	}
	if env.GetVar("EXECUTIONS_QUEUE") == "" {
		log.Fatal("invalid environment", "variable", "EXECUTIONS_QUEUE")
	}
	if env.GetVar("CANCEL_REJECTIONS_QUEUE") == "" {
		log.Fatal("invalid environment", "variable", "CANCEL_REJECTIONS_QUEUE")
	}
	if env.GetVar("PT_SECRET") == "" {
		log.Fatal("invalid environment", "variable", "PT_SECRET")
	}
}
