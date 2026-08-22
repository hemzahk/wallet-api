package dbtx

import (
	"context"
	"database/sql"
)

type TransactionKey string
var TxKey TransactionKey = "transaction-key"

func ExtractTx(ctx context.Context, db *sql.DB) DBTX {
	tx, ok := ctx.Value(TxKey).(*sql.Tx)
	if !ok || tx == nil {
		return NewSQLStore(db, nil)
	}

	return NewSQLStore(nil, tx)
}

func injectTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, TxKey, tx)
}