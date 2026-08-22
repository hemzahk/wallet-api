package gateway

import "errors"

var (
	// ErrInvalidWebhookSignature is returned when the HMAC
	// signature on a webhook payload doesn't match.
	ErrInvalidWebhookSignature = errors.New("invalid webhook signature")

	// ErrGatewayUnavailable is returned when the gateway API
	// cannot be reached or returns an unexpected error.
	ErrGatewayUnavailable = errors.New("payment gateway unavailable")
)