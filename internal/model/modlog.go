package model

import "time"

type ModLogEntry struct {
	ID        int
	Action    string
	Detail    string
	CreatedAt time.Time
}
