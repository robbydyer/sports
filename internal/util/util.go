package util

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"os"
	"time"

	"github.com/robfig/cron/v3"
)

// Today is sometimes actually yesterday
func Today(t time.Time) time.Time {
	if t.Local().Hour() < 4 {
		return t.AddDate(0, 0, -1).Local()
	}

	return t.Local()
}

func TodayFunc(t time.Time) func() []time.Time {
	return func() []time.Time {
		return []time.Time{Today(t)}
	}
}

// NCAAFToday takes a single "today" time and adds thurs-sat games for the coming week.
// Sunday shows previous week's scores
func NCAAFToday(t time.Time) []time.Time {
	// Bowl season is weird
	if t.Month() == time.December || t.Month() == time.January {
		switch t.Weekday() {
		case time.Monday:
			return []time.Time{
				t,
				t.AddDate(0, 0, 1),
				t.AddDate(0, 0, 2),
				t.AddDate(0, 0, 3),
				t.AddDate(0, 0, 4),
				t.AddDate(0, 0, 5),
				t.AddDate(0, 0, 6),
			}
		case time.Tuesday:
			return []time.Time{
				t.AddDate(0, 0, -1),
				t,
				t.AddDate(0, 0, 1),
				t.AddDate(0, 0, 2),
				t.AddDate(0, 0, 3),
				t.AddDate(0, 0, 4),
				t.AddDate(0, 0, 5),
			}
		case time.Wednesday:
			return []time.Time{
				t.AddDate(0, 0, -2),
				t.AddDate(0, 0, -1),
				t,
				t.AddDate(0, 0, 1),
				t.AddDate(0, 0, 2),
				t.AddDate(0, 0, 3),
				t.AddDate(0, 0, 4),
			}
		case time.Thursday:
			return []time.Time{
				t.AddDate(0, 0, -3),
				t.AddDate(0, 0, -2),
				t.AddDate(0, 0, -1),
				t,
				t.AddDate(0, 0, 1),
				t.AddDate(0, 0, 2),
				t.AddDate(0, 0, 3),
			}
		case time.Friday:
			return []time.Time{
				t.AddDate(0, 0, -4),
				t.AddDate(0, 0, -3),
				t.AddDate(0, 0, -2),
				t.AddDate(0, 0, -1),
				t,
				t.AddDate(0, 0, 1),
				t.AddDate(0, 0, 2),
			}
		case time.Saturday:
			return []time.Time{
				t.AddDate(0, 0, -5),
				t.AddDate(0, 0, -4),
				t.AddDate(0, 0, -3),
				t.AddDate(0, 0, -2),
				t.AddDate(0, 0, -1),
				t,
				t.AddDate(0, 0, 1),
			}
		case time.Sunday:
			return []time.Time{
				t.AddDate(0, 0, -6),
				t.AddDate(0, 0, -5),
				t.AddDate(0, 0, -4),
				t.AddDate(0, 0, -3),
				t.AddDate(0, 0, -2),
				t.AddDate(0, 0, -1),
				t,
			}
		}

		return []time.Time{t}
	}

	// Do Thurs-Saturday of the coming week
	switch t.Weekday() {
	case time.Monday:
		return []time.Time{
			t.AddDate(0, 0, 3),
			t.AddDate(0, 0, 4),
			t.AddDate(0, 0, 5),
		}
	case time.Tuesday:
		return []time.Time{
			t.AddDate(0, 0, 2),
			t.AddDate(0, 0, 3),
			t.AddDate(0, 0, 4),
		}
	case time.Wednesday:
		return []time.Time{
			t.AddDate(0, 0, 1),
			t.AddDate(0, 0, 2),
			t.AddDate(0, 0, 3),
		}
	case time.Thursday:
		return []time.Time{
			t,
			t.AddDate(0, 0, 1),
			t.AddDate(0, 0, 2),
		}
	case time.Friday:
		return []time.Time{
			t,
			t.AddDate(0, 0, -1),
			t.AddDate(0, 0, 1),
		}
	case time.Saturday:
		return []time.Time{
			t,
			t.AddDate(0, 0, -1),
			t.AddDate(0, 0, -2),
		}
	case time.Sunday:
		return []time.Time{
			t.AddDate(0, 0, -1),
			t.AddDate(0, 0, -2),
			t.AddDate(0, 0, -3),
		}
	}

	return []time.Time{t}
}

// NFLToday takes a single "today" time and adds thurs, sun, mon games for the coming week.
// Monday shows previous week's scores
func NFLToday(t time.Time) []time.Time {
	// Do Thurs-Sunday of the coming week
	switch t.Weekday() {
	case time.Monday:
		return []time.Time{
			t,
			t.AddDate(0, 0, -1),
			t.AddDate(0, 0, -2),
			t.AddDate(0, 0, -4),
		}
	case time.Tuesday:
		return []time.Time{
			t.AddDate(0, 0, 2),
			t.AddDate(0, 0, 4),
			t.AddDate(0, 0, 5),
			t.AddDate(0, 0, 6),
		}
	case time.Wednesday:
		return []time.Time{
			t.AddDate(0, 0, 1),
			t.AddDate(0, 0, 3),
			t.AddDate(0, 0, 4),
			t.AddDate(0, 0, 5),
		}
	case time.Thursday:
		return []time.Time{
			t,
			t.AddDate(0, 0, 2),
			t.AddDate(0, 0, 3),
			t.AddDate(0, 0, 4),
		}
	case time.Friday:
		return []time.Time{
			t.AddDate(0, 0, -1),
			t.AddDate(0, 0, 1),
			t.AddDate(0, 0, 2),
			t.AddDate(0, 0, 3),
		}
	case time.Saturday:
		return []time.Time{
			t.AddDate(0, 0, -2),
			t,
			t.AddDate(0, 0, 1),
			t.AddDate(0, 0, 2),
		}
	case time.Sunday:
		return []time.Time{
			t.AddDate(0, 0, -3),
			t.AddDate(0, 0, -1),
			t,
			t.AddDate(0, 0, 1),
		}
	}

	return []time.Time{t}
}

func AddTodays(today time.Time, previousDays int, advanceDays int) []time.Time {
	todays := []time.Time{}

	if previousDays > 0 {
		for i := 1; i <= previousDays; i++ {
			todays = append(todays, today.AddDate(0, 0, -1*i))
		}
	}
	todays = append(todays, today)

	if advanceDays > 0 {
		for i := 1; i <= advanceDays; i++ {
			todays = append(todays, today.AddDate(0, 0, i))
		}
	}

	return todays
}

// PullPng GETs a png and returns it decoded as an image.Image
func PullPng(ctx context.Context, url string) (image.Image, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	client := http.DefaultClient

	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to pull png from %s: http status %s", url, resp.Status)
	}

	return png.Decode(resp.Body)
}

// FileExists ...
func FileExists(fileName string) (bool, error) {
	_, err := os.Stat(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func SetCrons(times []string, f func()) error {
	if len(times) < 1 {
		return nil
	}

	c := cron.New()
	for _, t := range times {
		if _, err := c.AddFunc(t, f); err != nil {
			return fmt.Errorf("failed to add cron func: %w", err)
		}
	}
	c.Start()

	return nil
}
