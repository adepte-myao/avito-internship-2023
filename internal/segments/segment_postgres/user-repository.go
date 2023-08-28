package segment_postgres

import (
	"context"
	"database/sql"
	"errors"
	"math/rand"
	"time"

	"avito-internship-2023/internal/pkg/common"
	"avito-internship-2023/internal/segments"
)

var (
	ErrNoUsersToPick = errors.New("no users to pick")
)

type UserRepository struct {
	logger                 common.Logger
	db                     *sql.DB
	userSegmentHistoryRepo *UserSegmentHistoryRepository
}

func NewUserRepository(logger common.Logger, db *sql.DB, userSegmentHistoryRepo *UserSegmentHistoryRepository) *UserRepository {
	return &UserRepository{logger: logger, db: db, userSegmentHistoryRepo: userSegmentHistoryRepo}
}

func (repo *UserRepository) BeginTransaction(ctx context.Context) (context.Context, error) {
	tx, err := repo.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}

	return setTx(ctx, tx), nil
}

func (repo *UserRepository) Rollback(ctx context.Context) error {
	tx, err := getTx(ctx)
	if err != nil {
		return err
	}

	if err = tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		return err
	}
	return nil
}

func (repo *UserRepository) Commit(ctx context.Context) error {
	tx, err := getTx(ctx)
	if err != nil {
		return err
	}

	if err = tx.Commit(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		return err
	}
	return nil
}

func (repo *UserRepository) Create(ctx context.Context, user segments.User) error {
	executor, err := getExecutor(ctx, repo.db)
	if err != nil {
		return err
	}

	_, err = executor.ExecContext(ctx, `INSERT INTO users (id, status) VALUES ($1, $2)`, user.Id, user.Status)
	if err != nil {
		return err
	}

	return nil
}

func (repo *UserRepository) GetAll(ctx context.Context) ([]segments.User, error) {
	executor, err := getExecutor(ctx, repo.db)
	if err != nil {
		return nil, err
	}

	rows, err := executor.QueryContext(ctx, `SELECT id, status FROM users`)
	if err != nil {
		return nil, err
	}

	users := make([]segments.User, 0)
	for rows.Next() {
		var user segments.User
		err = rows.Scan(&user.Id, &user.Status)
		if err != nil {
			return nil, err
		}

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (repo *UserRepository) GetRandom(ctx context.Context) (segments.User, error) {
	executor, err := getExecutor(ctx, repo.db)
	if err != nil {
		return segments.User{}, err
	}

	var usersCount int
	err = executor.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&usersCount)
	if err != nil {
		return segments.User{}, err
	}

	if usersCount < 1 {
		return segments.User{}, ErrNoUsersToPick
	}

	offset := rand.Intn(usersCount)
	var user segments.User
	err = executor.QueryRowContext(ctx, `SELECT id, status FROM users OFFSET $1 LIMIT 1`, offset).Scan(&user.Id, &user.Status)
	if err != nil {
		return segments.User{}, err
	}

	return user, nil
}

func (repo *UserRepository) Exists(ctx context.Context, userID string) (bool, error) {
	executor, err := getExecutor(ctx, repo.db)
	if err != nil {
		return false, err
	}

	err = executor.QueryRowContext(ctx, `SELECT id FROM users WHERE id=$1`, userID).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (repo *UserRepository) Remove(ctx context.Context, userID string) error {
	executor, err := getExecutor(ctx, repo.db)
	if err != nil {
		return err
	}

	historyEntries, err := repo.prepareHistoryEntriesForUserRemove(ctx, userID)
	if err != nil {
		return err
	}

	err = repo.userSegmentHistoryRepo.AddEntries(ctx, historyEntries)
	if err != nil {
		return err
	}

	_, err = executor.ExecContext(ctx, `DELETE FROM users WHERE id=$1`, userID)
	if err != nil {
		return err
	}

	return nil
}

func (repo *UserRepository) prepareHistoryEntriesForUserRemove(ctx context.Context, userID string) ([]segments.UserSegmentHistoryEntry, error) {
	executor, err := getExecutor(ctx, repo.db)
	if err != nil {
		return nil, err
	}

	rowsUserSegments, err := executor.QueryContext(ctx, `SELECT segment_slug FROM users_segments WHERE user_id=$1`, userID)
	if err != nil {
		return nil, err
	}

	userSegments := make([]string, 0)
	for rowsUserSegments.Next() {
		var slug string
		err = rowsUserSegments.Scan(&slug)
		if err != nil {
			return nil, err
		}

		userSegments = append(userSegments, slug)
	}

	if err = rowsUserSegments.Err(); err != nil {
		return nil, err
	}

	historyEntries := make([]segments.UserSegmentHistoryEntry, len(userSegments))
	for i, slug := range userSegments {
		historyEntries[i] = segments.UserSegmentHistoryEntry{
			UserID:     userID,
			Slug:       slug,
			ActionType: segments.Removed,
			LogTime:    time.Now(),
		}
	}

	return historyEntries, nil
}

func (repo *UserRepository) Update(ctx context.Context, user segments.User) error {
	executor, err := getExecutor(ctx, repo.db)
	if err != nil {
		return err
	}

	_, err = executor.ExecContext(ctx, `UPDATE users SET status=$2 WHERE id=$1`, user.Id, user.Status)
	if err != nil {
		return err
	}

	return nil
}
