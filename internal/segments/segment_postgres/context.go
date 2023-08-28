package segment_postgres

import (
	"context"
	"database/sql"
	"errors"
)

type ctxKey struct{}

var (
	ErrInvalidContext   = errors.New("invalid context")
	ErrInvalidValueType = errors.New("invalid type of context value")
)

type sqlExecutor interface {
	Exec(query string, args ...any) (sql.Result, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func getTx(ctx context.Context) (*sql.Tx, error) {
	txVal := ctx.Value(ctxKey{})
	if txVal == nil {
		return nil, ErrInvalidContext
	}

	tx, ok := txVal.(*sql.Tx)
	if !ok {
		return nil, ErrInvalidValueType
	}

	return tx, nil
}

func setTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, ctxKey{}, tx)
}

func getExecutor(ctx context.Context, db *sql.DB) (sqlExecutor, error) {
	txVal := ctx.Value(ctxKey{})
	if txVal == nil {
		return db, nil
	}

	tx, ok := txVal.(*sql.Tx)
	if !ok {
		return nil, ErrInvalidValueType
	}

	return tx, nil
}
