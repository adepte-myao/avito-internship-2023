package domain

import "time"

type DeadlineEntry struct {
	UserID   string
	Slug     string
	Deadline time.Time
}
