package model

import "time"

type Ban struct {
	ID        int
	IP        string
	BoardID   *int
	Reason    string
	ExpiresAt *time.Time
	CreatedAt time.Time
}
