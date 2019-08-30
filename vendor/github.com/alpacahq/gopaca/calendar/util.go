package calendar

import "time"

// Iterator is a time iterator
type Iterator struct {
	start    time.Time
	end      time.Time
	interval time.Duration
	current  time.Time
	index    int
}

// NewIterator creates a new time iterator
func NewIterator(start time.Time, end time.Time, interval time.Duration) *Iterator {
	return &Iterator{
		start:    start,
		end:      end,
		interval: interval,
		current:  start,
		index:    0,
	}
}

// Next returns the next time in the iterator
func (i *Iterator) Next() bool {
	var next time.Time

	if i.index == 0 {
		next = i.current
	} else {
		next = i.current.Add(i.interval)
	}

	if i.end.Equal(next) || i.end.After(next) {
		i.current = next
		i.index++
		return true
	}
	return false
}

// Current returns the latest time that has yet
// to be scanned
func (i *Iterator) Current() time.Time {
	return i.current
}
