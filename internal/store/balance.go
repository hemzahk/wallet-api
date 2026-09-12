package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/hemzahk/wallet-api/internal/dbtx"
)

var (
	ErrBalanceNotFound = errors.New("balance not found")
	ErrOptimisticLock  = errors.New("balance: version conflict or balance not found")
	ErrInvalidBalance  = errors.New("balance is nil or incomplete")
)

type Balances interface {
	CreateBalance(ctx context.Context, balance *Balance) error
	GetByIdentityID(ctx context.Context, identityID uuid.UUID) (*Balance, error)
	GetByEmail(ctx context.Context, email string) (*Balance, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*Balance, error)
	GetByBalanceID(ctx context.Context, balanceID string) (*Balance, error)
	GetByMerchantID(ctx context.Context, merchantID uuid.UUID) (*Balance, error)
	UpdateBalance(ctx context.Context, balance *Balance) error
}

type Balance struct {
	ID                    uuid.UUID `json:"-"`
	BalanceID             string    `json:"balance_id"`
	Balance               *big.Int  `json:"balance"`
	InflightCreditBalance *big.Int  `json:"inflight_credit_balance"`
	InflightDebitBalance  *big.Int  `json:"inflight_debit_balance"`
	Currency              string    `json:"currency"`
	LedgerID              string    `json:"ledger_id"`
	IdentityID            uuid.UUID `json:"identity_id"`
	Version               int       `json:"version"`
	CreatedAt             time.Time `json:"created_at"`
}

type BalanceStore struct {
	db *sql.DB
}

func (s *BalanceStore) CreateBalance(ctx context.Context, balance *Balance) error {
	if balance == nil {
		return ErrInvalidBalance
	}

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
		return fmt.Errorf("creating balance: %w", err)
	}

	return nil
}

func (s *BalanceStore) GetByIdentityID(ctx context.Context, identityID uuid.UUID) (*Balance, error) {
	query := `
		SELECT id, balance_id, balance, inflight_credit_balance, inflight_debit_balance, identity_id, ledger_id, currency , version, created_at
		FROM balances
		WHERE identity_id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	balance := &Balance{}
	var rawBalance, inflightCreditBalance, inflightDebitBalance int64

	err := s.db.QueryRowContext(ctx, query, identityID).Scan(
		&balance.ID,
		&balance.BalanceID,
		&rawBalance,
		&inflightCreditBalance,
		&inflightDebitBalance,
		&balance.IdentityID,
		&balance.LedgerID,
		&balance.Currency,
		&balance.Version,
		&balance.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("getting balance by identity ID: %w", balanceLookupError(err))
	}

	balance.Balance = big.NewInt(rawBalance)
	balance.InflightCreditBalance = big.NewInt(inflightCreditBalance)
	balance.InflightDebitBalance = big.NewInt(inflightDebitBalance)

	return balance, nil
}

func (s *BalanceStore) GetByEmail(ctx context.Context, email string) (*Balance, error) {
	query := `
		SELECT b.id, b.identity_id, b.ledger_id, b.balance_id, b.balance, b.inflight_credit_balance, b.inflight_debit_balance, b.currency, b.version, b.created_at 
		FROM balances b 
		JOIN users u ON (b.identity_id = u.identity_id)
		WHERE u.email = $1
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	balance := &Balance{}
	var balanceAsInt, inflightCreditBalance, inflightDebitBalance int64

	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&balance.ID,
		&balance.IdentityID,
		&balance.LedgerID,
		&balance.BalanceID,
		&balanceAsInt,
		&inflightCreditBalance,
		&inflightDebitBalance,
		&balance.Currency,
		&balance.Version,
		&balance.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("getting balance by email: %w", balanceLookupError(err))
	}

	balance.Balance = big.NewInt(balanceAsInt)
	balance.InflightCreditBalance = big.NewInt(inflightCreditBalance)
	balance.InflightDebitBalance = big.NewInt(inflightDebitBalance)

	return balance, nil
}

func (s *BalanceStore) GetByUserID(ctx context.Context, userID uuid.UUID) (*Balance, error) {
	query := `
		SELECT b.id, b.identity_id, b.ledger_id, b.balance_id, b.balance, b.inflight_credit_balance, b.inflight_debit_balance, b.currency, b.version, b.created_at 
		FROM balances b 
		JOIN users u ON (b.identity_id = u.identity_id)
		WHERE u.id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	balance := &Balance{}
	var rawBalance, inflightCreditBalance, inflightDebitBalance int64

	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&balance.ID,
		&balance.IdentityID,
		&balance.LedgerID,
		&balance.BalanceID,
		&rawBalance,
		&inflightCreditBalance,
		&inflightDebitBalance,
		&balance.Currency,
		&balance.Version,
		&balance.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("getting balance by user ID: %w", balanceLookupError(err))
	}

	balance.Balance = big.NewInt(rawBalance)
	balance.InflightCreditBalance = big.NewInt(inflightCreditBalance)
	balance.InflightDebitBalance = big.NewInt(inflightDebitBalance)

	return balance, nil
}

func (s *BalanceStore) GetByBalanceID(ctx context.Context, balanceID string) (*Balance, error) {
	dbtx := dbtx.ExtractTx(ctx, s.db)
	query := `
		SELECT id, balance_id, balance, inflight_credit_balance, inflight_debit_balance, identity_id, ledger_id, currency, version, created_at
		FROM balances
		WHERE balance_id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	balance := &Balance{}
	var rawBalance, inflightCreditBalance, inflightDebitBalance int64

	err := dbtx.QueryRowContext(ctx, query, balanceID).Scan(
		&balance.ID,
		&balance.BalanceID,
		&rawBalance,
		&inflightCreditBalance,
		&inflightDebitBalance,
		&balance.IdentityID,
		&balance.LedgerID,
		&balance.Currency,
		&balance.Version,
		&balance.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("getting balance by balance ID: %w", balanceLookupError(err))
	}

	balance.Balance = big.NewInt(rawBalance)
	balance.InflightCreditBalance = big.NewInt(inflightCreditBalance)
	balance.InflightDebitBalance = big.NewInt(inflightDebitBalance)

	return balance, nil
}

func (s *BalanceStore) GetByMerchantID(ctx context.Context, merchantID uuid.UUID) (*Balance, error) {
	query := `
		SELECT b.id, b.identity_id, b.ledger_id, b.balance_id, b.balance, b.inflight_credit_balance, b.inflight_debit_balance, b.currency, b.version, b.created_at 
		FROM balances b 
		JOIN users u ON (b.identity_id = u.identity_id)
		JOIN merchants m ON (u.id = m.user_id)
		WHERE m.id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	balance := &Balance{}
	var rawBalance, inflightCreditBalance, inflightDebitBalance int64

	err := s.db.QueryRowContext(ctx, query, merchantID).Scan(
		&balance.ID,
		&balance.IdentityID,
		&balance.LedgerID,
		&balance.BalanceID,
		&rawBalance,
		&inflightCreditBalance,
		&inflightDebitBalance,
		&balance.Currency,
		&balance.Version,
		&balance.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("getting balance by merchant ID: %w", balanceLookupError(err))
	}

	balance.Balance = big.NewInt(rawBalance)
	balance.InflightCreditBalance = big.NewInt(inflightCreditBalance)
	balance.InflightDebitBalance = big.NewInt(inflightDebitBalance)

	return balance, nil
}

func (s *BalanceStore) UpdateBalance(ctx context.Context, balance *Balance) error {
	if balance == nil || balance.Balance == nil || balance.InflightCreditBalance == nil || balance.InflightDebitBalance == nil {
		return ErrInvalidBalance
	}

	dbtx := dbtx.ExtractTx(ctx, s.db)

	query := `
		UPDATE balances 
		SET balance = $1, 
			inflight_credit_balance = $2, 
			inflight_debit_balance = $3, 
			version = version + 1
		WHERE balance_id = $4 AND version = $5
		RETURNING version
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	var newVersion int64
	err := dbtx.QueryRowContext(
		ctx, query,
		balance.Balance.Int64(),
		balance.InflightCreditBalance.Int64(),
		balance.InflightDebitBalance.Int64(),
		balance.BalanceID,
		balance.Version).Scan(
		&newVersion,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrOptimisticLock
	}

	if err != nil {
		return fmt.Errorf("updating balance %q: %w", balance.BalanceID, err)
	}

	balance.Version = int(newVersion)

	return nil
}

func balanceLookupError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrBalanceNotFound
	}

	return err
}
