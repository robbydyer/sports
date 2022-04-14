package yahoo

import (
	"fmt"
	"regexp"
	"time"
)

var interval = regexp.MustCompile(`[0-9]+[a-z]+`)

func durationToAPIInterval(d time.Duration) string {
	if d.Minutes() >= 90.0 && d.Minutes() < 120.0 {
		return fmt.Sprintf("%.0fm", d.Minutes())
	}
	return interval.FindString(d.String())
}

func tradingLocation() (*time.Location, error) {
	return time.LoadLocation("America/New_York")
}

func tradingEnd() (time.Time, error) {
	t := time.Now()
	loc, err := tradingLocation()
	if err != nil {
		return t, err
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 16, 30, 0, 0, loc), nil
}

func tradingBegin() (time.Time, error) {
	t := time.Now()
	loc, err := tradingLocation()
	if err != nil {
		return t, err
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 8, 0o0, 0, 0, loc), nil
}
