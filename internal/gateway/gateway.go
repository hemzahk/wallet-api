package gateway

import (
	"context"
	"math/big"
)

type GatewaySession struct {
	GatewayRef string
	PaymentURL string
}

type PayoutSession struct {
    GatewayRef string
}

type PaymentGateway interface {
	InitiatePayment(ctx context.Context, amount *big.Int) (GatewaySession, error)
	InitiatePayout(_ context.Context, amount *big.Int) (PayoutSession, error)
	VerifyWebhookSignature(payload []byte, signature string) error
}