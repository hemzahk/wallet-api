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
type StubGateway struct {
	webhookSecret string
	baseURL       string
}

func NewStubGateway(webhookSecret, baseURL string) *StubGateway {
	return &StubGateway{
		webhookSecret: webhookSecret,
		baseURL:       baseURL,
	}
}

// InitiatePayment fakes a gateway session. It generates a gateway_ref
// and returns a mock payment URL the client would be redirected to.
func (g *StubGateway) InitiatePayment(_ context.Context, amount *big.Int) (GatewaySession, error) {
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

func (g *StubGateway) InitiatePayout(_ context.Context, amount *big.Int) (PayoutSession, error) {
	gatewayRef := uuid.New().String()

	return PayoutSession{
		GatewayRef: gatewayRef,
	}, nil
}

// VerifyWebhookSignature checks the HMAC-SHA256 signature sent by the gateway.
// The gateway signs the raw request body with the shared secret.
// Signature format expected in the header: hex-encoded HMAC-SHA256.
func (g *StubGateway) VerifyWebhookSignature(payload []byte, signature string) error {
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
	payload := `{
  "amount": "17000.00",
  "event_id": "19376e3a-9dd2-47ba-be36-298459e64677",
  "gateway_ref": "3b2d04c9-78ad-456e-b5cb-01ac5f301f43",
  "source_id": "645b44fb-314f-4e34-8106-17b09cc9660a",
  "status": "succeeded"
}`

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))

	fmt.Println(signature)
}

*/