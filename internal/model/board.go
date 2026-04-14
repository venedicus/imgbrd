package model

import "time"

type Board struct {
	ID          int
	Slug        string
	Title       string
	MaxThreads  int
	NSFW        bool
	CreatedAt   time.Time
}
