package util

import "time"

// Today is sometimes actually yesterday
func Today() time.Time {
	if time.Now().Local().Hour() < 4 {
		return time.Now().AddDate(0, 0, -1).Local()
	}

	return time.Now().Local()
}
