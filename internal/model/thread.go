package model

import "time"

type Thread struct {
	ID            int
	BoardID       int
	Title         string
	BumpedAt      time.Time
	Pinned        bool
	Archived      bool
	CreatedAt     time.Time
	BoardThreadNo int
}
