package imageboard

import (
	"fmt"
	"strings"
)

// returns true if this seems like the same file
func filenameCompare(a string, b string) bool {
	a = strings.ToLower(a)
	b = strings.ToLower(b)

	if a == b {
		return true
	}

	aFull := strings.HasPrefix(a, "/")
	bFull := strings.HasPrefix(b, "/")

	aPaths := reverseStrs(strings.Split(a, "/"))
	bPaths := reverseStrs(strings.Split(b, "/"))

	if len(aPaths) != len(bPaths) && aFull && bFull {
		return false
	}

	fmt.Println(aPaths)
	fmt.Println(bPaths)

	if len(aPaths) < 1 || len(bPaths) < 1 {
		return false
	}

	max := len(aPaths)
	if len(bPaths) > max {
		max = len(bPaths)
	}

	for x := 0; x < max; x++ {
		if len(aPaths) < x+1 || len(bPaths) < x+1 {
			return true
		}
		if aPaths[x] != bPaths[x] {
			return false
		}
	}

	return true
}

func reverseStrs(s []string) []string {
	if len(s) == 0 {
		return []string{}
	}

	ret := make([]string, len(s))
	index := 0
	for i := len(s) - 1; i >= 0; i-- {
		ret[index] = s[i]
		index++
	}

	return ret
}
