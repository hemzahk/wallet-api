package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/hemzahk/wallet-api/internal/store"
	"github.com/stretchr/testify/mock"
)

type MockTransactions struct {
	mock.Mock
}

func (m *MockTransactions) Record(ctx context.Context, transaction *store.Transaction) error {
	args := m.Called(ctx, transaction)
	return args.Error(0)
}

func (m *MockTransactions) GetByRef(ctx context.Context, reference string) (*store.Transaction, error) {
	args := m.Called(ctx, reference)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Transaction), args.Error(1)
}

func (m *MockTransactions) GetByIdentityID(ctx context.Context, identityID uuid.UUID) ([]store.Transaction, error) {
	args := m.Called(ctx, identityID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]store.Transaction), args.Error(1)
}