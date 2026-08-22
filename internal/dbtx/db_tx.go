package dbtx

import (
	"context"
	"database/sql"
)

type DBTX interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type SQLStore struct {
	NonTx *sql.DB
	Tx *sql.Tx
}

func NewSQLStore(db *sql.DB, tx *sql.Tx) DBTX {
	if tx != nil {
		return &SQLStore{Tx: tx}
	}

	return &SQLStore{NonTx: db}
}

func (s *SQLStore) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	if s.Tx != nil {
		return s.Tx.QueryRowContext(ctx, query, args...)
	}

	return s.NonTx.QueryRowContext(ctx, query, args...)
}

func (s *SQLStore) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if s.Tx != nil {
		return s.Tx.ExecContext(ctx, query, args...)
	}

	return s.NonTx.ExecContext(ctx, query, args...)
}