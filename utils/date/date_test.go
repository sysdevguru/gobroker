package date

import (
	"testing"
	"time"
)

func TestFormat(t *testing.T) {
	{
		d, _ := ParseDate("2018-01-02")
		if !(d.Day == 2 && d.Year == 2018 && d.Month == time.January) {
			t.FailNow()
		}
		if d.String() != "2018-01-02" {
			t.FailNow()
		}
	}

	{
		// now not supported
		_, err := ParseDate("2018/01/02")
		if err == nil {
			t.FailNow()
		}
	}

}
