package segments

import (
	"errors"
	"time"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrInvalidUserStatus = errors.New("invalid user status")
)

type CreateSegmentDTO struct {
	Slug          string  `json:"slug" validate:"required"`
	PercentToFill float64 `json:"percentToFill" validate:"gte=0,lte=100"`
}

type RemoveSegmentDTO struct {
	Slug string `json:"slug" validate:"required"`
}

type AddSegmentEntry struct {
	SegmentSlug                 string    `json:"segmentSlug" validate:"required"`
	SecondsToBeInSegment        int       `json:"secondsToBeInSegment" validate:"gte=1"`
	DeadlineForStayingInSegment time.Time `json:"deadlineForStayingInSegment"`
}

type RemoveSegmentEntry struct {
	SegmentSlug string `json:"segmentSlug" validate:"required"`
}

type ChangeSegmentsForUserDTO struct {
	UserID           string               `json:"userID" validate:"required"`
	SegmentsToAdd    []AddSegmentEntry    `json:"segmentsToAdd"`
	SegmentsToRemove []RemoveSegmentEntry `json:"segmentsToRemove"`
}

type GetSegmentsForUserDTO struct {
	UserID string `json:"userID" validate:"required"`
}

type GetSegmentsForUserOutDTO struct {
	Segments []string `json:"segments"`
}

type GetSegmentsHistoryReportLinkDTO struct {
	UserID string `json:"userID" validate:"required"`
	Month  int    `json:"month" validate:"required,gte=1,lte=12"`
	Year   int    `json:"year" validate:"required"`
}

type CreateUserDTO struct {
	UserID string `json:"userID" validate:"required"`
}

type RemoveUserDTO struct {
	UserID string `json:"userID" validate:"required"`
}

type UpdateUserDTO struct {
	UserID string     `json:"userID" validate:"required"`
	Status UserStatus `json:"status"`
}

type UserActionDTO struct {
	UserID string `json:"userID" validate:"required"`
}

type UserStatus string

const (
	Active   UserStatus = "active"
	Excluded UserStatus = "excluded"
)

var possibleStatuses = []UserStatus{Active, Excluded}

func validateUserStatus(status UserStatus) error {
	for _, possibleStatus := range possibleStatuses {
		if status == possibleStatus {
			return nil
		}
	}

	return ErrInvalidUserStatus
}

type User struct {
	Id     string
	Status UserStatus
}

type Segment struct {
	Slug string
}

type DeadlineEntry struct {
	UserID   string
	Slug     string
	Deadline time.Time
}
