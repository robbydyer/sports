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
