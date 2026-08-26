package store

import (
	"context"
	"database/sql"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/hemzahk/wallet-api/internal/dbtx"
)

type Balance struct {
	ID uuid.UUID `json:"-"`
	BalanceID string `json:"balance_id"`
	Balance *big.Int `json:"balance"`
	Currency string `json:"currency"`
	LedgerID string `json:"ledger_id"`
	IdentityID uuid.UUID `json:"identity_id"`
	Version int `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}

type BalanceStore struct {
	db *sql.DB
}

func (s *BalanceStore) CreateBalance(ctx context.Context, balance *Balance) error {
	dbtx := dbtx.ExtractTx(ctx, s.db)
	query := `
		INSERT INTO balances (id, balance_id, ledger_id, identity_id, created_at)
		VALUES ($1,$2,$3,$4,$5)
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	_, err := dbtx.ExecContext(
		ctx,
		query, 
		&balance.ID,
		&balance.BalanceID, 
		&balance.LedgerID,
		&balance.IdentityID,
		&balance.CreatedAt,
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *BalanceStore) GetByIdentityID(ctx context.Context, identityID uuid.UUID) (*Balance, error) {
	query := `
		SELECT id, balance_id, balance, identity_id, ledger_id, currency , version, created_at
		FROM balances
		WHERE identity_id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	balance := &Balance{}
	var rawBalance int64
	err := s.db.QueryRowContext(ctx, query, identityID).Scan(
		&balance.ID,
		&balance.BalanceID,
		&rawBalance,
		&balance.IdentityID, 
		&balance.LedgerID,
		&balance.Currency, 
		&balance.Version,
		&balance.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	balanceAsBigInt := big.NewInt(rawBalance)
	balance.Balance = balanceAsBigInt
	return balance, nil
}

func (s *BalanceStore) GetByEmail(ctx context.Context, email string) (*Balance, error) {
	query := `
		SELECT b.id, b.identity_id, b.ledger_id, b.balance_id, b.balance, b.currency, b.version, b.created_at 
		FROM balances b 
		JOIN users u ON (b.identity_id = u.identity_id)
		WHERE u.email = $1
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	balance := &Balance{}
	var balanceAsInt int64
	
	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&balance.ID,
		&balance.IdentityID,
		&balance.LedgerID,
		&balance.BalanceID, 
		&balanceAsInt,
		&balance.Currency,
		&balance.Version,
		&balance.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	
	balanceAsBigInt := big.NewInt(balanceAsInt)
	balance.Balance = balanceAsBigInt

	return balance, nil
}

func (s *BalanceStore) GetByUserID(ctx context.Context, userID uuid.UUID) (*Balance, error) {
	query := `
		SELECT b.id, b.identity_id, b.ledger_id, b.balance_id, b.balance, b.currency, b.version, b.created_at 
		FROM balances b 
		JOIN users u ON (b.identity_id = u.identity_id)
		WHERE u.id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	balance := &Balance{}
	var balanceAsInt int64
	
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&balance.ID,
		&balance.IdentityID,
		&balance.LedgerID,
		&balance.BalanceID, 
		&balanceAsInt,
		&balance.Currency,
		&balance.Version,
		&balance.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	
	balanceAsBigInt := big.NewInt(balanceAsInt)
	balance.Balance = balanceAsBigInt

	return balance, nil
}

func (s *BalanceStore) GetByBalanceID(ctx context.Context, balanceID string) (*Balance, error) {
	query := `
		SELECT id, balance_id, balance, identity_id, ledger_id, currency, version, created_at
		FROM balances
		WHERE balance_id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	balance := &Balance{}
	var rawBalance int64
	
	err := s.db.QueryRowContext(ctx, query, balanceID).Scan(
		&balance.ID,
		&balance.BalanceID,
		&rawBalance,
		&balance.IdentityID, 
		&balance.LedgerID,
		&balance.Currency, 
		&balance.Version,
		&balance.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	balanceAsBigInt := big.NewInt(rawBalance)
	balance.Balance = balanceAsBigInt

	return balance, nil
}

func (s *BalanceStore) GetByMerchantID(ctx context.Context, merchantID uuid.UUID) (*Balance, error) {
	query := `
		SELECT b.id, b.identity_id, b.ledger_id, b.balance_id, b.balance, b.currency, b.version, b.created_at 
		FROM balances b 
		JOIN users u ON (b.identity_id = u.identity_id)
		JOIN merchants m ON (u.id = m.user_id)
		WHERE m.id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	balance := &Balance{}
	var rawBalance int64
	err := s.db.QueryRowContext(ctx, query, merchantID).Scan(
		&balance.ID,
		&balance.IdentityID,
		&balance.LedgerID,
		&balance.BalanceID,
		&rawBalance,
		&balance.Currency,
		&balance.Version,
		&balance.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	balanceAsBigInt := big.NewInt(rawBalance)
	balance.Balance = balanceAsBigInt
	
	return balance, nil
}

func (s *BalanceStore) UpdateBalance(ctx context.Context, balance *Balance) error {
	dbtx := dbtx.ExtractTx(ctx, s.db)
	
	query := `
		UPDATE balances SET balance = $1, version = version + 1
		WHERE balance_id = $2 AND version = $3
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	_, err := dbtx.ExecContext(ctx, query, balance.Balance.Int64(), balance.BalanceID, balance.Version)
	if err != nil {
		return err
	}

	return nil
}