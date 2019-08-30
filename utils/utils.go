package utils

import (
	"strconv"

	"github.com/alpacahq/gopaca/env"
)

// Dev returns true if broker is in development mode
func Dev() bool {
	return env.GetVar("BROKER_MODE") == "DEV"
}

// Stg returns true if broker is in staging mode
func Stg() bool {
	return env.GetVar("BROKER_MODE") == "STG"
}

// Prod returns true if broker is in production mode
func Prod() bool {
	return env.GetVar("BROKER_MODE") == "PROD"
}

// StandBy returns true if broker is in standby mode
func StandBy() bool {
	standby, _ := strconv.ParseBool("STANDBY_MODE")
	return standby
}

var (
	Sha1hash string
	Version  string = "dev"
)
