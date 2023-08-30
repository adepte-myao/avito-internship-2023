package ports

import (
	"context"
	"time"

	"avito-internship-2023/internal/segments/core/domain"
)

type TransactionHelper interface {
	BeginTransaction(ctx context.Context) (context.Context, error)
	Rollback(ctx context.Context) error
	Commit(ctx context.Context) error
}

type UserProvider interface {
	TransactionHelper
	Create(ctx context.Context, user domain.User) error
	GetAll(ctx context.Context) ([]domain.User, error)
	GetRandom(ctx context.Context) (domain.User, error)
	Exists(ctx context.Context, userID string) (bool, error)
	Remove(ctx context.Context, userID string) error
	Update(ctx context.Context, user domain.User) error
}

type SegmentsProvider interface {
	TransactionHelper
	GetAllAsMap(ctx context.Context) (map[string]domain.Segment, error)
	GetForUser(ctx context.Context, userID string) ([]string, error)
	Create(ctx context.Context, segment domain.Segment) error
	Remove(ctx context.Context, slug string) error
	AddUsersToSegments(ctx context.Context, userIDs, slugs []string) error
	RemoveSegmentsForUser(ctx context.Context, userID string, slugsToRemove []string) error
}

type UserSegmentHistoryProvider interface {
	GetAllForUser(ctx context.Context, userID string, month, year int) ([]domain.UserSegmentHistoryEntry, error)
}

type DeadlineProvider interface {
	GetAllBefore(ctx context.Context, maxTime time.Time) ([]domain.DeadlineEntry, error)
	Remove(ctx context.Context, toRemove []domain.DeadlineEntry) error
}
