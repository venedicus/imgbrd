package util

import "regexp"

var refRe = regexp.MustCompile(`>>(\d+)`)

// PostRefMarkers turns >>N into markdown links to #pN (same thread).
func PostRefMarkers(text string) string {
	return refRe.ReplaceAllString(text, "[>>$1](#p$1)")
}
