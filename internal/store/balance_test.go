package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCreateBalance_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	s := BalanceStore{db: db}

	balance := &Balance{
		ID:         uuid.New(),
		BalanceID:  fmt.Sprintf("bln_%s", uuid.New().String()),
		IdentityID: uuid.New(),
		LedgerID:   "customer_ledger_id",
		CreatedAt:  time.Now(),
	}

	mock.ExpectExec("INSERT INTO balances").WithArgs(
		balance.ID,
		balance.BalanceID,
		balance.LedgerID,
		balance.IdentityID,
		balance.CreatedAt,
	).WillReturnResult(sqlmock.NewResult(1, 1))

	err = s.CreateBalance(ctx, balance)
	assert.NoError(t, err)
}

func TestUpdateBalance_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	s := BalanceStore{db: db}
	balanceID := fmt.Sprintf("bln_%s", uuid.New().String())
	balance := &Balance{
		BalanceID:             balanceID,
		Balance:               big.NewInt(100000),
		InflightCreditBalance: big.NewInt(0),
		InflightDebitBalance:  big.NewInt(0),
		Version:               3,
	}

	query := `
		UPDATE balances 
		SET balance = $1, 
			inflight_credit_balance = $2, 
			inflight_debit_balance = $3, 
			version = version + 1
		WHERE balance_id = $4 AND version = $5
		RETURNING version
	`

	rows := sqlmock.NewRows([]string{"version"}).AddRow(4)

	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(balance.Balance.Int64(), balance.InflightCreditBalance.Int64(), balance.InflightDebitBalance.Int64(), balance.BalanceID, balance.Version).
		WillReturnRows(rows)

	err = s.UpdateBalance(ctx, balance)

	assert.NoError(t, err)
	assert.Equal(t, 4, balance.Version)
}

func TestGetByIdentityID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	identityID := uuid.New()
	mock.ExpectQuery("SELECT id, balance_id").
		WithArgs(identityID).
		WillReturnError(sql.ErrNoRows)

	s := BalanceStore{db: db}
	_, err = s.GetByIdentityID(context.Background(), identityID)

	assert.ErrorIs(t, err, ErrBalanceNotFound)
	assert.NotErrorIs(t, err, sql.ErrNoRows)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateBalance_OptimisticLockConflict(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	balance := &Balance{
		BalanceID:             "bln_test",
		Balance:               big.NewInt(100),
		InflightCreditBalance: big.NewInt(0),
		InflightDebitBalance:  big.NewInt(0),
		Version:               3,
	}

	mock.ExpectQuery(regexp.QuoteMeta(`
		UPDATE balances 
		SET balance = $1, 
			inflight_credit_balance = $2, 
			inflight_debit_balance = $3, 
			version = version + 1
		WHERE balance_id = $4 AND version = $5
		RETURNING version
	`)).
		WithArgs(int64(100), int64(0), int64(0), balance.BalanceID, balance.Version).
		WillReturnError(sql.ErrNoRows)

	s := BalanceStore{db: db}
	err = s.UpdateBalance(context.Background(), balance)

	assert.ErrorIs(t, err, ErrOptimisticLock)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateBalance_InvalidBalance(t *testing.T) {
	s := BalanceStore{}

	err := s.UpdateBalance(context.Background(), nil)

	assert.True(t, errors.Is(err, ErrInvalidBalance))
}
