package mocks

import (
	"context"

	"github.com/google/uuid"
	"github.com/hemzahk/wallet-api/internal/store"
	"github.com/stretchr/testify/mock"
)

type MockBalances struct {
	mock.Mock
}

func (m *MockBalances) CreateBalance(ctx context.Context, balance *store.Balance) error {
	args := m.Called(ctx, balance)
	return args.Error(0)
}

func (m *MockBalances) GetByIdentityID(ctx context.Context, identityID uuid.UUID) (*store.Balance, error) {
	args := m.Called(ctx, identityID)
	return args.Get(0).(*store.Balance), args.Error(1)
}

func (m *MockBalances) GetByEmail(ctx context.Context, email string) (*store.Balance, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Balance), args.Error(1)
}

func (m *MockBalances) GetByUserID(ctx context.Context, userID uuid.UUID) (*store.Balance, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Balance), args.Error(1)
}

func (m *MockBalances) GetByBalanceID(ctx context.Context, balanceID string) (*store.Balance, error) {
	args := m.Called(ctx, balanceID)
	return args.Get(0).(*store.Balance), args.Error(1)
}

func (m *MockBalances) GetByMerchantID(ctx context.Context, merchantID uuid.UUID) (*store.Balance, error) {
	args := m.Called(ctx, merchantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*store.Balance), args.Error(1)
}

func (m *MockBalances) UpdateBalance(ctx context.Context, balance *store.Balance) error {
	args := m.Called(ctx, balance)
	return args.Error(0)
}