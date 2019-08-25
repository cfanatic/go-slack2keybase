// Package bridge forwards chat messages from Slack to Keybase.
package bridge

import (
	"fmt"
	"math"
	"strconv"
	utime "time"
)

const (
	// Conversion layout format from Local time to Unix time
	TIME_LAYOUT = "2006-01-02 15:04:05.999999999 -0700 MST"
)

type Timestamp struct {
	stamp utime.Time
}

type duration = utime.Duration

// NewTimestamp initializes the internal Time attribute and returns an object of type Timestamp.
// It takes the time of type string as input either given in Unix or Local time format.
func NewTimestamp(time string) Timestamp {
	t := Timestamp{}
	t.Set(time)
	return t
}

// String implements the Stringer interface which is used to define the native print format for the object.
// It returns the time given in Local time format as this ideal for humans to read.
func (t Timestamp) String() string {
	return t.Local()
}

// Unix returns the time in Unix time format without nanosecond precision, e.g. "1563305596".
// Optional input argument is a time which can be used to initialize the internal Time attribute first.
func (t *Timestamp) Unix(time ...string) string {
	if time != nil {
		t.Set(time[0])
	}
	return strconv.FormatInt(t.stamp.Unix(), 10)
}

// UnixNano returns the time in Unix time format with nanosecond precision, e.g. "1563305596.004500".
// Optional input argument is a time which can be used to initialize the internal Time attribute first.
func (t *Timestamp) UnixNano(time ...string) string {
	if time != nil {
		t.Set(time[0])
	}
	temp := float64(t.stamp.UnixNano()) / float64(1e9)
	return fmt.Sprintf("%f", temp)
}

// Local returns the time in Local time format, e.g. "2019-07-16 21:33:16.0045 +0200 CEST".
// Optional input argument is a time which can be used to initialize the internal Time attribute first.
func (t *Timestamp) Local(time ...string) string {
	if time != nil {
		t.Set(time[0])
	}
	return t.stamp.String()
}

// Add adds an user specific duration to the internal Time attribute.
// Input argument is a delta time.
func (t *Timestamp) Add(d duration) {
	t.stamp = t.stamp.Add(d)
}

// Set initializes the internal Time attribute using an Unix time with nanosecond precision or Local time as input.
// Any other given time format as input triggers a panic.
func (t *Timestamp) Set(time string) {
	if num, err := strconv.ParseFloat(time, 64); err == nil {
		frac := fmt.Sprintf("%.6f", math.Mod(num, 1))
		nsec, _ := strconv.ParseFloat(frac, 64)
		t.stamp = utime.Unix(int64(num), int64(nsec*1e9))
	} else if num, err := utime.Parse(TIME_LAYOUT, time); err == nil {
		t.stamp = num
	} else {
		panic(fmt.Sprintf("ERROR: Invalid timestamp %s\n", time))
	}
}
