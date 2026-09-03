package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCreate_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	s := CheckoutSessionStore{db: db}

	session := &CheckoutSession{
		ID: uuid.New(),
		Token: "token-123",
		MerchantID: uuid.New(),
		Amount: big.NewInt(100000),
		ExpiresAt: time.Now().Add(time.Minute*5),
		CreatedAt: time.Now(),
	}

	mock.ExpectExec("INSERT INTO checkout_sessions").
		WithArgs(
			session.ID,
			session.Token,
			session.MerchantID,
			session.Amount.Int64(),
			session.ExpiresAt,
			session.CreatedAt,
		).
		WillReturnResult(sqlmock.NewResult(1,1))
	
	err = s.Create(ctx, session)
	assert.NoError(t, err)
}

func TestGetByToken_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() {_ = db.Close()}()

	ctx := context.Background()

	s := CheckoutSessionStore{db: db}

	id := uuid.New()
	merchantID := uuid.New()

	plainToken := uuid.New().String()

	hash := sha256.Sum256([]byte(plainToken))
	hashToken := hex.EncodeToString(hash[:])

	rows := sqlmock.NewRows([]string{"id", "token", "merchant_id", "amount", "expires_at", "created_at"}).
		AddRow(id, hashToken, merchantID, big.NewInt(100000).Int64(), time.Now().Add(time.Minute*5), time.Now())

	query := "SELECT id, token, merchant_id, amount, expires_at, created_at FROM checkout_sessions WHERE token = $1 AND expires_at > $2"

	mock.ExpectQuery(regexp.QuoteMeta(query)).
	WithArgs(hashToken, time.Now()).
	WillReturnRows(rows)

	session, err := s.GetByToken(ctx, plainToken)
	assert.NoError(t, err)
	assert.Equal(t, big.NewInt(100000), session.Amount)
}