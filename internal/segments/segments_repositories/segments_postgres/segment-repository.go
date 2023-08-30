package segments_postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"avito-internship-2023/internal/pkg/common"
	"avito-internship-2023/internal/pkg/postgres"
	"avito-internship-2023/internal/segments/segments_core/segments_domain"

	"github.com/lib/pq"
)

type SegmentRepository struct {
	logger                 common.Logger
	db                     *sql.DB
	userSegmentHistoryRepo *UserSegmentHistoryRepository
}

func NewSegmentRepository(logger common.Logger, db *sql.DB, userSegmentHistoryRepo *UserSegmentHistoryRepository) *SegmentRepository {
	return &SegmentRepository{logger: logger, db: db, userSegmentHistoryRepo: userSegmentHistoryRepo}
}

func (repo *SegmentRepository) BeginTransaction(ctx context.Context) (context.Context, error) {
	tx, err := repo.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}

	return postgres.SetTx(ctx, tx), nil
}

func (repo *SegmentRepository) Rollback(ctx context.Context) error {
	tx, err := postgres.GetTx(ctx)
	if err != nil {
		return err
	}

	if err = tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		return err
	}
	return nil
}

func (repo *SegmentRepository) Commit(ctx context.Context) error {
	tx, err := postgres.GetTx(ctx)
	if err != nil {
		return err
	}

	if err = tx.Commit(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		return err
	}
	return nil
}

func (repo *SegmentRepository) GetAllAsMap(ctx context.Context) (map[string]segments_domain.Segment, error) {
	executor, err := postgres.GetExecutor(ctx, repo.db)
	if err != nil {
		return nil, err
	}

	rows, err := executor.QueryContext(ctx, `SELECT slug FROM segments`)
	if err != nil {
		return nil, err
	}

	segmentsMap := make(map[string]segments_domain.Segment)
	for rows.Next() {
		var segment segments_domain.Segment
		err = rows.Scan(&segment.Slug)
		if err != nil {
			return nil, err
		}

		segmentsMap[segment.Slug] = segment
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return segmentsMap, nil
}

func (repo *SegmentRepository) GetForUser(ctx context.Context, userID string) ([]string, error) {
	executor, err := postgres.GetExecutor(ctx, repo.db)
	if err != nil {
		return nil, err
	}

	rows, err := executor.QueryContext(ctx, `SELECT segment_slug FROM users_segments WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}

	slugs := make([]string, 0)
	for rows.Next() {
		var slug string
		err = rows.Scan(&slug)
		if err != nil {
			return nil, err
		}

		slugs = append(slugs, slug)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return slugs, nil
}

func (repo *SegmentRepository) Create(ctx context.Context, segment segments_domain.Segment) error {
	executor, err := postgres.GetExecutor(ctx, repo.db)
	if err != nil {
		return err
	}

	_, err = executor.ExecContext(ctx, `INSERT INTO segments (slug) VALUES ($1)`, segment.Slug)
	if err != nil {
		return err
	}

	return nil
}

func (repo *SegmentRepository) Remove(ctx context.Context, slug string) error {
	executor, err := postgres.GetExecutor(ctx, repo.db)
	if err != nil {
		return err
	}

	historyEntries, err := repo.prepareHistoryEntriesForSegmentRemove(ctx, slug)
	if err != nil {
		return err
	}

	err = repo.userSegmentHistoryRepo.AddEntries(ctx, historyEntries)
	if err != nil {
		return err
	}

	_, err = executor.ExecContext(ctx, `DELETE FROM segments WHERE slug=$1`, slug)
	if err != nil {
		return err
	}

	return nil
}

func (repo *SegmentRepository) prepareHistoryEntriesForSegmentRemove(ctx context.Context, slug string) ([]segments_domain.UserSegmentHistoryEntry, error) {
	executor, err := postgres.GetExecutor(ctx, repo.db)
	if err != nil {
		return nil, err
	}

	rowsUsersInSegment, err := executor.QueryContext(ctx, `SELECT user_id FROM users_segments WHERE segment_slug=$1`, slug)
	if err != nil {
		return nil, err
	}

	userIDsInSegment := make([]string, 0)
	for rowsUsersInSegment.Next() {
		var userID string
		err = rowsUsersInSegment.Scan(&userID)
		if err != nil {
			return nil, err
		}

		userIDsInSegment = append(userIDsInSegment, userID)
	}

	if err = rowsUsersInSegment.Err(); err != nil {
		return nil, err
	}

	historyEntries := make([]segments_domain.UserSegmentHistoryEntry, len(userIDsInSegment))
	for i, userID := range userIDsInSegment {
		historyEntries[i] = segments_domain.UserSegmentHistoryEntry{
			UserID:     userID,
			Slug:       slug,
			ActionType: segments_domain.Removed,
			LogTime:    time.Now(),
		}
	}

	return historyEntries, nil
}

func (repo *SegmentRepository) AddUsersToSegments(ctx context.Context, userIDs, slugs []string) error {
	if len(userIDs)*len(slugs) == 0 {
		return nil
	}

	executor, err := postgres.GetExecutor(ctx, repo.db)
	if err != nil {
		return err
	}

	placeHolders := make([]string, 0)
	values := make([]any, 0)
	const nArgs = 2
	for i, userID := range userIDs {
		for j, slug := range slugs {
			startIndex := nArgs * (i*len(slugs) + j)
			placeHolders = append(placeHolders, postgres.NewPlaceHolders(startIndex, nArgs))
			values = append(values, userID, slug)
		}
	}

	sqlInsert := `INSERT INTO users_segments (user_id, segment_slug) VALUES ` + strings.Join(placeHolders, ",")
	_, err = executor.ExecContext(ctx, sqlInsert, values...)
	if err != nil {
		return err
	}

	historyEntries := make([]segments_domain.UserSegmentHistoryEntry, len(userIDs)*len(slugs))
	for i, userID := range userIDs {
		for j, slug := range slugs {
			index := i*len(slugs) + j
			historyEntries[index] = segments_domain.UserSegmentHistoryEntry{
				UserID:     userID,
				Slug:       slug,
				ActionType: segments_domain.Added,
				LogTime:    time.Now(),
			}
		}
	}

	err = repo.userSegmentHistoryRepo.AddEntries(ctx, historyEntries)
	if err != nil {
		return err
	}

	return nil
}

func (repo *SegmentRepository) RemoveSegmentsForUser(ctx context.Context, userID string, slugsToRemove []string) error {
	executor, err := postgres.GetExecutor(ctx, repo.db)
	if err != nil {
		return err
	}

	_, err = executor.ExecContext(ctx, `DELETE FROM users_segments WHERE user_id=$1 AND segment_slug = ANY($2)`, userID, pq.Array(slugsToRemove))
	if err != nil {
		return err
	}

	historyEntries := make([]segments_domain.UserSegmentHistoryEntry, len(slugsToRemove))
	for i, slug := range slugsToRemove {
		historyEntries[i] = segments_domain.UserSegmentHistoryEntry{
			UserID:     userID,
			Slug:       slug,
			ActionType: segments_domain.Removed,
			LogTime:    time.Now(),
		}
	}

	err = repo.userSegmentHistoryRepo.AddEntries(ctx, historyEntries)
	if err != nil {
		return err
	}

	return nil
}
