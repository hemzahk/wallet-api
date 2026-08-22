package dbtx

import (
	"context"
	"database/sql"
)

type TxManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type txManager struct {
	db *sql.DB
}

func NewTxManager(db *sql.DB) TxManager {
	return &txManager{db: db}
}

func (t *txManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := t.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if err := fn(injectTx(ctx,tx)); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}