package domain

import "time"

type HistoryActionType string

var (
	Added   HistoryActionType = "added"
	Removed HistoryActionType = "removed"
)

type UserSegmentHistoryEntry struct {
	UserID     string
	Slug       string
	ActionType HistoryActionType
	LogTime    time.Time
}
