package yahoo

import (
	"regexp"
	"time"
)

var inerval = regexp.MustCompile(`[0-9]+[a-z]+`)

func durationToAPIInterval(d time.Duration) string {
	return inerval.FindString(d.String())
}
