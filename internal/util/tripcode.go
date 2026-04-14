package util

import (
	"crypto/sha1"
	"encoding/hex"
)

// TripFromPassword returns display suffix (e.g. "!a1b2c3d4") and full hash for storage.
func TripFromPassword(password string) (displaySuffix, fullHash string) {
	if password == "" {
		return "", ""
	}
	h := sha1.Sum([]byte("imgbrd.trip.v1#" + password))
	full := hex.EncodeToString(h[:])
	return "!" + full[:8], full
}
