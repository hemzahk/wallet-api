package gateway

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/google/uuid"
)

// MockGateway simulates a payment gateway (CIB / EDAHABIA).
// In production, replace this with a concrete HTTP client
// that calls the real gateway API.
type MockGateway struct {
	webhookSecret string
	baseURL       string
}

func NewMockGateway(webhookSecret, baseURL string) *MockGateway {
	return &MockGateway{
		webhookSecret: webhookSecret,
		baseURL:       baseURL,
	}
}

// InitiatePayment fakes a gateway session. It generates a gateway_ref
// and returns a mock payment URL the client would be redirected to.
func (g *MockGateway) InitiatePayment(_ context.Context, amount *big.Int) (GatewaySession, error) {
	gatewayRef := uuid.New().String()

	paymentURL := fmt.Sprintf(
		"%s/pay?ref=%s&amount=%s",
		g.baseURL,
		gatewayRef,
		amount.String(),
	)

	return GatewaySession{
		GatewayRef: gatewayRef,
		PaymentURL: paymentURL,
	}, nil
}

func (g *MockGateway) InitiatePayout(_ context.Context, amount *big.Int) (PayoutSession, error) {
	gatewayRef := uuid.New().String()

	return PayoutSession{
		GatewayRef: gatewayRef,
	}, nil
}

// VerifyWebhookSignature checks the HMAC-SHA256 signature sent by the gateway.
// The gateway signs the raw request body with the shared secret.
// Signature format expected in the header: hex-encoded HMAC-SHA256.
func (g *MockGateway) VerifyWebhookSignature(payload []byte, signature string) error {
	mac := hmac.New(sha256.New, []byte(g.webhookSecret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return ErrInvalidWebhookSignature
	}

	return nil
}

/*
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func main() {
	secret := "OC83Vc7lna7sADKHzdWZZiWvVceu3gR5MGuOTgkGisc"
	payload := `{"gateway_ref":"feea0ed4-31c2-40a4-813d-224447cb3343","topup_id":"9f27828b-3943-42b1-b854-e7b1ee4b12c3","status":"success","amount":"10000.00"}`

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))

	fmt.Println(signature)
}

*/