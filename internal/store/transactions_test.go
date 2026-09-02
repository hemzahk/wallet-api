package store

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRecordTransaction_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	s := TransactionStore{db: db}

	transaction := &Transaction{
		ID: uuid.New(),
		PreciseAmount: big.NewInt(100000),
		Reference: fmt.Sprintf("test_%s", uuid.New().String()),
		Source: "src1",
		Destination: "dest1",
		Description: "transfer",
		Status: "applied",
		CreatedAt: time.Now(),
	}

	mock.ExpectExec("INSERT INTO transactions").WithArgs(
		transaction.ID,
		transaction.ParentTransaction,
		transaction.Reference,
		transaction.PreciseAmount.Int64(),
		transaction.Source,
		transaction.Destination,
		transaction.Status,
		transaction.Description,
		transaction.CreatedAt,
	).WillReturnResult(sqlmock.NewResult(1,1))

	err = s.Record(ctx, transaction)
	assert.NoError(t, err)
}

func TestGetByRef_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	s := TransactionStore{db: db}

	id := uuid.New()
	
	rows := sqlmock.NewRows([]string{"id", "parent_transaction", "reference", "precise_amount", "source", "destination", "status", "description", "created_at"}).AddRow(
		id, "", "test_123", big.NewInt(100000).Int64(), "src1", "dest1", "inflight", "test transaction", time.Now())

	mock.ExpectQuery("SELECT id, parent_transaction, reference, precise_amount, source, destination, status, description, created_at FROM transactions WHERE reference = ?").
		WithArgs("test_123").WillReturnRows(rows)

	txn, err := s.GetByRef(ctx, "test_123")
	assert.NoError(t, err)
	assert.Equal(t, "src1", txn.Source)
	assert.Equal(t, "dest1", txn.Destination)
	assert.Equal(t, id, txn.ID)
}

func TestGetByIdentityID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	s := TransactionStore{db: db}

	id := uuid.New()
	id2 := uuid.New()	
	identityID := uuid.New()

	rows := sqlmock.NewRows([]string{"id", "parent_transaction", "reference", "precise_amount", "source", "destination", "status", "description", "created_at"}).
		AddRow(id, sql.NullString{}, "test_123", big.NewInt(100000).Int64(), "src1", "dest1", "inflight", "test transaction", time.Now()).
		AddRow(id2, sql.NullString{String: id.String(), Valid: true}, "test_456", big.NewInt(100000).Int64(), "src1", "dest1", "applied", "test transaction", time.Now())
	
	query := `
		SELECT t.id, t.parent_transaction, t.reference, t.precise_amount, t.source, t.destination, t.status, t.description, t.created_at
		FROM transactions t
		JOIN balances b ON (b.balance_id = t."source" OR b.balance_id = t.destination)
		WHERE b.identity_id = $1 AND t.destination <> '@Revenue' AND status <> 'inflight'		
	`
	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(identityID).WillReturnRows(rows)

	result, err := s.GetByIdentityID(ctx, identityID)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, id, result[0].ID)
}