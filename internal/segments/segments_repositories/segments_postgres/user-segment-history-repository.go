package segments_postgres

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"avito-internship-2023/internal/pkg/common"
	"avito-internship-2023/internal/pkg/postgres"
	"avito-internship-2023/internal/segments/segments_core"
)

type UserSegmentHistoryRepository struct {
	logger common.Logger
	db     *sql.DB
}

func NewUserSegmentHistoryRepository(logger common.Logger, db *sql.DB) *UserSegmentHistoryRepository {
	return &UserSegmentHistoryRepository{logger: logger, db: db}
}

func (repo *UserSegmentHistoryRepository) GetAllForUser(ctx context.Context, userID string, month, year int) ([]segments_core.UserSegmentHistoryEntry, error) {
	executor, err := getExecutor(ctx, repo.db)
	if err != nil {
		return nil, err
	}

	rows, err := executor.QueryContext(ctx,
		`SELECT user_id, segment_slug, action, log_time FROM users_segments_history 
                WHERE user_id = $1 AND EXTRACT(MONTH FROM log_time)=$2 AND EXTRACT(YEAR FROM log_time)=$3`,
		userID, month, year)
	if err != nil {
		return nil, err
	}

	historyEntries := make([]segments_core.UserSegmentHistoryEntry, 0)
	for rows.Next() {
		var entry segments_core.UserSegmentHistoryEntry
		err = rows.Scan(&entry.UserID, &entry.Slug, &entry.ActionType, &entry.LogTime)
		if err != nil {
			return nil, err
		}

		historyEntries = append(historyEntries, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return historyEntries, nil
}

func (repo *UserSegmentHistoryRepository) AddEntries(ctx context.Context, entries []segments_core.UserSegmentHistoryEntry) error {
	if len(entries) == 0 {
		return nil
	}

	executor, err := getExecutor(ctx, repo.db)
	if err != nil {
		return err
	}

	for i := 0; i < len(entries); i++ {
		if entries[i].LogTime.IsZero() {
			entries[i].LogTime = time.Now()
		}
	}

	placeHolders := make([]string, 0)
	values := make([]any, 0)
	const nArgs = 4
	for i, entry := range entries {
		startIndex := nArgs * i
		placeHolders = append(placeHolders, postgres.NewPlaceHolders(startIndex, nArgs))
		values = append(values, entry.UserID, entry.Slug, entry.ActionType, entry.LogTime)
	}

	sqlInsert := `INSERT INTO users_segments_history (user_id, segment_slug, action, log_time) VALUES ` + strings.Join(placeHolders, ",")
	_, err = executor.ExecContext(ctx, sqlInsert, values...)
	if err != nil {
		return err
	}

	return nil
}
