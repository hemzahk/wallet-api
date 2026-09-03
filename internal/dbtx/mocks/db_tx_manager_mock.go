package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MockTxManager struct {
	mock.Mock
}

func (m *MockTxManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	args := m.Called(ctx, fn)
	if returnFn, ok := args.Get(0).(func(context.Context, func(context.Context) error) error); ok {
		return returnFn(ctx, fn)
	}
	return args.Error(0)
}