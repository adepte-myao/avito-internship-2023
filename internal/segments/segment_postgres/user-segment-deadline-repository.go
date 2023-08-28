package segment_postgres

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"avito-internship-2023/internal/pkg/common"
	"avito-internship-2023/internal/pkg/postgres"
	"avito-internship-2023/internal/segments"

	"github.com/lib/pq"
)

type UserSegmentDeadlineRepository struct {
	logger                 common.Logger
	db                     *sql.DB
	userSegmentHistoryRepo *UserSegmentHistoryRepository
}

func NewUserSegmentDeadlineRepository(logger common.Logger, db *sql.DB, userSegmentHistoryRepo *UserSegmentHistoryRepository) *UserSegmentDeadlineRepository {
	return &UserSegmentDeadlineRepository{logger: logger, db: db, userSegmentHistoryRepo: userSegmentHistoryRepo}
}

func (repo *UserSegmentDeadlineRepository) AddDeadlines(ctx context.Context, deadlines []segments.DeadlineEntry) error {
	if len(deadlines) == 0 {
		return nil
	}

	executor, err := getExecutor(ctx, repo.db)
	if err != nil {
		return err
	}

	placeHolders := make([]string, 0)
	values := make([]any, 0)
	const nArgs = 3
	for i, entry := range deadlines {
		startIndex := nArgs * i
		placeHolders = append(placeHolders, postgres.NewPlaceHolders(startIndex, nArgs))
		values = append(values, entry.UserID, entry.Slug, entry.Deadline)
	}

	sqlInsert := `INSERT INTO users_segments_to_remove (user_id, segment_slug, remove_time) VALUES ` + strings.Join(placeHolders, ",")
	_, err = executor.ExecContext(ctx, sqlInsert, values...)
	if err != nil {
		return err
	}

	return nil
}

func (repo *UserSegmentDeadlineRepository) GetAllBefore(ctx context.Context, maxTime time.Time) ([]segments.DeadlineEntry, error) {
	executor, err := getExecutor(ctx, repo.db)
	if err != nil {
		return nil, err
	}

	rows, err := executor.QueryContext(ctx,
		`SELECT user_id, segment_slug, remove_time FROM users_segments_to_remove WHERE remove_time < $1`, maxTime)
	if err != nil {
		return nil, err
	}

	deadlineEntries := make([]segments.DeadlineEntry, 0)
	for rows.Next() {
		var entry segments.DeadlineEntry
		err = rows.Scan(&entry.UserID, &entry.Slug, &entry.Deadline)
		if err != nil {
			return nil, err
		}

		deadlineEntries = append(deadlineEntries, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return deadlineEntries, nil
}

func (repo *UserSegmentDeadlineRepository) Remove(ctx context.Context, toRemove []segments.DeadlineEntry) error {
	executor, err := getExecutor(ctx, repo.db)
	if err != nil {
		return err
	}

	// toRemoveSegmentMap[slug] = [userID_1, userID_2, ...]
	toRemoveSegmentMap := make(map[string][]string)
	for _, entry := range toRemove {
		userIDs, ok := toRemoveSegmentMap[entry.Slug]
		if ok {
			userIDs = []string{entry.UserID}
		} else {
			userIDs = append(userIDs, entry.UserID)
		}

		toRemoveSegmentMap[entry.Slug] = userIDs
	}

	for slug, userIDS := range toRemoveSegmentMap {
		_, err = executor.ExecContext(ctx, `DELETE FROM users_segments_to_remove WHERE segment_slug=$1 AND user_id = ANY($2)`, slug, pq.Array(userIDS))
		if err != nil {
			return err
		}
	}

	return nil
}
