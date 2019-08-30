package calendar

import (
	"fmt"
	"time"
)

// DateRange list of time.Time support Unix convert
type DateRange []time.Time

type RangeFreq string

const (
	Min5 RangeFreq = "5Min"
	H1   RangeFreq = "1H"
	D1   RangeFreq = "1D"
)

// NewRange return list of timestamps for its period
func NewRange(start time.Time, end time.Time, freq RangeFreq) (DateRange, error) {
	if freq == D1 {
		return newDayRange(start, end)
	}

	freqMap := map[RangeFreq]time.Duration{
		Min5: 5 * time.Minute,
		H1:   1 * time.Hour,
	}

	duration, ok := freqMap[freq]

	if !ok {
		return nil, fmt.Errorf("freq %v is not supported", freq)
	}

	return newRange(start, end, duration)
}

// newDayRange returns list of beginning of trading time in the period.
func newDayRange(start time.Time, end time.Time) (DateRange, error) {
	marketOpen := MarketOpen(start)
	var o time.Time
	if marketOpen == nil || start.After(*marketOpen) {
		o = NextOpen(start)
	} else {
		o = *marketOpen
	}

	// Pre allocate list to optimize performance
	maxBuckets := end.Sub(o).Nanoseconds()/(24*time.Hour).Nanoseconds() + 1
	tseries := make([]time.Time, maxBuckets)

	i := 0
	for o.Before(end) {
		tseries[i] = o

		o = NextOpen(o)
		i++
	}

	return tseries[:i], nil
}

func newRange(start time.Time, end time.Time, freq time.Duration) (DateRange, error) {
	if (time.Minute*24*60)%freq != time.Minute*0 {
		return nil, fmt.Errorf("freq need to be divisible for 24h by %v", freq)
	}

	var o time.Time
	if IsMarketOpen(start) {
		o = start
	} else {
		o = NextOpen(start)
	}
	c := NextClose(o)

	// Pre allocate list to optimize performance
	tseries := make([]time.Time, end.Sub(o).Nanoseconds()/freq.Nanoseconds())

	i := 0
	for o.Before(end) {
		for o.Before(c) && o.Before(end) {
			tseries[i] = o.Truncate(freq)
			o = o.Add(freq)
			i++
		}

		o = NextOpen(o)
		c = NextClose(o)
	}
	return tseries[:i], nil
}

// Unix convert DateRange to unix timestamps
func (r *DateRange) Unix() []int64 {
	o := make([]int64, len(*r))

	for i, v := range *r {
		o[i] = v.UTC().Unix()
	}

	return o
}
