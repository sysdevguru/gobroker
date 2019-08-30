package common

import (
	"time"
)

func WaitTimeout(done chan struct{}, timeout time.Duration) bool {
	select {
	case <-done:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}
