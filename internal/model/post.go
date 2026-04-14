package model

import "time"

type Post struct {
	ID          int
	ThreadID    int
	Text        string
	Image       string
	Sage        bool
	Hidden      bool
	PosterName  string
	TripHash    string
	FileHash    string
	Mime        string
	FileSize    int64
	ThumbPath   string
	CreatedAt   time.Time
	BoardPostNo int
}
