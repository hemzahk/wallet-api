package gateway

import (
	"context"
	"math/big"

	"github.com/stretchr/testify/mock"
)

type MockPaymentGateway struct {
	mock.Mock
}

func (m *MockPaymentGateway) InitiatePayment(ctx context.Context, amount *big.Int) (GatewaySession, error) {
	args := m.Called(ctx, amount)
	return args.Get(0).(GatewaySession), args.Error(1)
}

func (m *MockPaymentGateway) InitiatePayout(ctx context.Context, amount *big.Int) (PayoutSession, error) {
	args := m.Called(ctx, amount)
	return args.Get(0).(PayoutSession), args.Error(1)
}

func (m *MockPaymentGateway) VerifyWebhookSignature(payload []byte, signature string) error {
	args := m.Called(payload, signature)
	return args.Error(0)
}