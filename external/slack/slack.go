package slack

import (
	"github.com/alpacahq/gobroker/utils"
)

// Notify sends a payload over Slack to the specified target.
// Supplied message can be string, in which case it will be
// sent in raw form, or an object where it will be marshalled
// to JSON.
func Notify(msg Message) {
	switch {
	case utils.Stg():
		msg.SendStaging()
	case utils.Prod():
		msg.SendProduction()
	}
}
