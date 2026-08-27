package store

import (
	"context"
	"database/sql"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/hemzahk/wallet-api/internal/dbtx"
)

type Transaction struct {
	ID uuid.UUID `json:"id"`
	ParentTransaction uuid.UUID `json:"parent_transaction"`
	PreciseAmount *big.Int `json:"precise_amount"` 
	Reference string `json:"reference"`
	Source string `json:"source"`
	Destination string `json:"destination"`
	Status string `json:"status"`
	Description string `json:"description"`
	CreatedAt time.Time `json:"created_at"`
}

type TransactionStore struct {
	db *sql.DB
}

func (s *TransactionStore) recordTransaction(ctx context.Context, transaction *Transaction) error {
	query := `
		INSERT INTO transactions (
			id,
			reference,
			precise_amount,
			source,
			destination,
			status,
			description,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	transaction.Reference = generateUUIDWithSuffix("txn")

	_, err := s.db.ExecContext(
		ctx,
		query,
		transaction.ID,
		transaction.Reference,
		transaction.PreciseAmount.Int64(),
		transaction.Source,
		transaction.Destination,
		transaction.Status,
		transaction.Description,
		transaction.CreatedAt,
	)
	if err != nil {
		return err
	}

	return nil
}

// func (s *TransactionStore) RecordTransactionAndUpdateBalance(ctx context.Context, transaction *Transaction, sourceBalance, destinationBalance *Balance) error {
// 	return withTx(s.db, ctx, func(tx *sql.Tx) error {
// 		if err := s.record(ctx, tx, transaction); err != nil {
// 			return err
// 		}

// 		if err := s.updateBalance(ctx, tx, sourceBalance); err != nil {
// 			return err
// 		}

// 		if err := s.updateBalance(ctx, tx, destinationBalance); err != nil {
// 			return err
// 		}

// 		return nil
// 	})
// }

func (s *TransactionStore) Record(ctx context.Context, transaction *Transaction) error {
	dbtx := dbtx.ExtractTx(ctx, s.db)

	query := `
		INSERT INTO transactions (
			id,
			parent_transaction,
			reference,
			precise_amount,
			source,
			destination,
			status,
			description,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	// transaction.Reference = generateUUIDWithSuffix("txn") send this from the frontend ??? 

	_, err := dbtx.ExecContext(
		ctx,
		query,
		transaction.ID,
		transaction.ParentTransaction,
		transaction.Reference,
		transaction.PreciseAmount.Int64(),
		transaction.Source,
		transaction.Destination,
		transaction.Status,
		transaction.Description,
		transaction.CreatedAt,
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *TransactionStore) GetByRef(ctx context.Context, reference string) (*Transaction, error) {
	query := `
		SELECT id, precise_amount, reference, source, destination, status, description, created_at
		FROM transactions
		WHERE reference = $1
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	transaction := &Transaction{}
	var rawAmount int64
	err := s.db.QueryRowContext(ctx, query, reference).Scan(
		&transaction.ID,
		&rawAmount,
		&transaction.Reference,
		&transaction.Source,
		&transaction.Destination,
		&transaction.Status,
		&transaction.Description, 
		&transaction.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	amountAsBigInt := big.NewInt(rawAmount)
	transaction.PreciseAmount = amountAsBigInt

	return transaction, nil
}

func (s *TransactionStore) GetByIdentityID(ctx context.Context, identityID uuid.UUID) ([]Transaction, error) {
	query := `
		SELECT t.precise_amount, t.description, t.created_at, t.source, t.destination
		FROM transactions t
		JOIN balances b ON (b.balance_id = t."source" OR b.balance_id = t.destination)
		WHERE b.identity_id = $1 AND t.destination <> '@Revenue' AND status <> 'inflight'
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, query, identityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []Transaction
	for rows.Next() {
		var t Transaction
		var rawAmount int64
		if err := rows.Scan(&rawAmount, &t.Description, &t.CreatedAt, &t.Source, &t.Destination); err != nil {
			return nil, err
		}
		amountAsBigInt := big.NewInt(rawAmount)
		t.PreciseAmount = amountAsBigInt
		transactions = append(transactions, t)
	}

	return transactions, nil
}



