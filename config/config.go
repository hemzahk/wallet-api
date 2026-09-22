package config

import (
	"fmt"
	"time"

	"github.com/hemzahk/wallet-api/internal/ratelimiter"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Addr             string        `envconfig:"ADDR" required:"true" default:":8080"`
	IdleTimeout      time.Duration `envconfig:"IDLE_TIMEOUT" required:"true"`
	ReadTimeout      time.Duration `envconfig:"READ_TIMEOUT" required:"true"`
	WriteTimeout     time.Duration `envconfig:"WRITE_TIMEOUT" required:"true"`
	Env              string        `envconfig:"ENV" required:"true"`
	DB               DbConfig
	ExternalURL      string `envconfig:"EXTERNAL_URL" required:"true"`
	FrontendURL      string `envconfig:"FRONTEND_URL" required:"true"`
	Mail             MailConfig
	Auth             AuthConfig
	PaymentProcessor PaymentProcessorConfig
	Ratelimiter      ratelimiter.Config
}

type DbConfig struct {
	Addr         string `envconfig:"DB_ADDR" required:"true"`
	MaxOpenConns int    `envconfig:"DB_MAX_OPEN_CONNS" required:"true"`
	MaxIdleConns int    `envconfig:"DB_MAX_IDLE_CONNS" required:"true"`
	MaxIdleTime  string `envconfig:"DB_MAX_IDLE_TIME" required:"true"`
}

type MailConfig struct {
	MailTrap  MailTrapConfig
	FromEmail string        `enconfig:"MAIL_FROMEMAIL" required:"true"`
	Exp       time.Duration `envconfig:"MAIL_EMAILEXP" required:"true"`
}

type MailTrapConfig struct {
	Username string `envconfig:"MAILTRAP_USERNAME" required:"true"`
	Password string `envconfig:"MAILTRAP_PASSWORD" required:"true"`
}

type AuthConfig struct {
	Token TokenConfig
}

type TokenConfig struct {
	Secret string        `envconfig:"AUTH_TOKEN_SECRET" required:"true"`
	Exp    time.Duration `envconfig:"AUTH_TOKEN_EXP" required:"true"`
	Iss    string        `envconfig:"AUTH_TOKEN_ISS" required:"true"`
}

type PaymentProcessorConfig struct {
	WebhookSecret string `envconfig:"WEBHOOK_SECRET" required:"true"`
	BaseURL       string `envconfig:"GATEWAY_BASE_URL" required:"true"`
}

func LoadWithEnvConfig() (*Config, error) {
	var cfg Config

	err := envconfig.Process("", &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	return &cfg, nil
}
