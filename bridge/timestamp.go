package bridge

import (
	"fmt"
	"math"
	"strconv"
	utime "time"
)

const (
	TIME_LAYOUT = "2006-01-02 15:04:05.999999999 -0700 MST"
)

type Timestamp struct {
	stamp utime.Time
}

type Duration = utime.Duration

func NewTimestamp(time string) Timestamp {
	t := Timestamp{}
	t.Set(time)
	return t
}

func (t Timestamp) String() string {
	return t.Local()
}

func (t *Timestamp) Unix(time ...string) string {
	if time != nil {
		t.Set(time[0])
	}
	return strconv.FormatInt(t.stamp.Unix(), 10)
}

func (t *Timestamp) UnixNano(time ...string) string {
	if time != nil {
		t.Set(time[0])
	}
	temp := float64(t.stamp.UnixNano()) / float64(1e9)
	return fmt.Sprintf("%f", temp)
}

func (t *Timestamp) Local(time ...string) string {
	if time != nil {
		t.Set(time[0])
	}
	return t.stamp.String()
}

func (t *Timestamp) Add(d Duration) {
	t.stamp = t.stamp.Add(d)
}

func (t *Timestamp) Set(time string) {
	if num, err := strconv.ParseFloat(time, 64); err == nil {
		frac := fmt.Sprintf("%.6f", math.Mod(num, 1))
		nsec, _ := strconv.ParseFloat(frac, 64)
		t.stamp = utime.Unix(int64(num), int64(nsec*1e9))
	}
	if num, err := utime.Parse(TIME_LAYOUT, time); err == nil {
		t.stamp = num
	}
}
