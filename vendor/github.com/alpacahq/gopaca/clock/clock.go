package clock

import (
	"sync"
	"time"

	"github.com/alpacahq/gopaca/calendar"
	"github.com/alpacahq/gopaca/env"
)

type Clock struct {
	start time.Time
}

var once sync.Once
var systemClock *Clock
var startTime time.Time

func Set(startTimes ...time.Time) {
	if len(startTimes) != 0 {
		st := startTimes[0]
		startTime = time.Now().UTC()
		systemClock = &Clock{start: st}
	} else {
		clock()
	}
}

func clock() *Clock {
	once.Do(func() {
		startTime = time.Now().UTC()
		start, _ := time.ParseInLocation("2006-01-02 15:04", env.GetVar("START_TIME"), calendar.NY)
		if start.IsZero() {
			start = time.Now()
		}
		systemClock = &Clock{start: start}
	})
	return systemClock
}

func Now() time.Time {
	return clock().start.Add(time.Now().Sub(startTime))
}

func Elapsed() time.Duration {
	return time.Now().Sub(startTime)
}

func Since(t time.Time) time.Duration {
	return Now().Sub(t)
}

// Call this function to schedule the alarm. The callback will be called after the alarm is triggered.
func ScheduleAlarm(alarmTime time.Time, callback func()) (endChan chan string) {
	endChan = make(chan string)
	startAlarm := alarmTime.Sub(Now())
	go func() {
		// Setting alarm.
		time.AfterFunc(startAlarm, func() {
			callback()
			endChan <- "finished waiting"
			close(endChan)
		})
	}()
	return endChan
}
