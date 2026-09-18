package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/hemzahk/wallet-api/internal/ratelimiter"
	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

type Config struct {
	Addr             string
	IdleTimeout      time.Duration
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	Env              string
	Db               DbConfig
	ApiURL           string
	FrontendURL      string
	Mail             MailConfig
	Auth             AuthConfig
	PaymentProcessor PaymentProcessorConfig
	Ratelimiter      ratelimiter.Config
}

type DbConfig struct {
	Addr         string
	MaxOpenConns int
	MaxIdleConns int
	MaxIdleTime  string
}

type MailConfig struct {
	MailTrap  MailTrapConfig
	FromEmail string
	Exp       time.Duration
}

type MailTrapConfig struct {
	Username string
	Password string
}

type AuthConfig struct {
	Token TokenConfig
}

type TokenConfig struct {
	Secret string
	Exp    time.Duration
	Iss    string
}

type PaymentProcessorConfig struct {
	WebhookSecret string
	BaseURL       string
}

func LoadFromEnv() (*Config, error) {
	cfg := &Config{}

	// server config
	cfg.Addr = getString("ADDR", ":8080")
	cfg.IdleTimeout = getDuration("IDLE_TIMEOUT", time.Minute)
	cfg.ReadTimeout = getDuration("READ_TIMEOUT", time.Second*20)
	cfg.WriteTimeout = getDuration("WRITE_TIMEOUT", time.Second*30)

	cfg.Env = getString("ENV", "development")
	cfg.ApiURL = getString("EXTERNAL_URL", "localhost:8080")

	// database config
	cfg.Db.Addr = getString("DB_ADDR", "postgres://hemzakareche:OnemustimagineSisyphushappy@localhost/wallet?sslmode=disable")
	cfg.Db.MaxOpenConns = getInt("DB_MAX_OPEN_CONNS", 30)
	cfg.Db.MaxIdleConns = getInt("DB_MAX_IDLE_CONNS", 30)
	cfg.Db.MaxIdleTime = getString("DB_MAX_IDLE_TIME", "15m")

	// mailer config
	cfg.Mail.Exp = getDuration("EMAIL_EXP", time.Hour*24*3)
	cfg.Mail.FromEmail = getString("FROM_EMAIL", "no-reply@wallet.io")
	cfg.Mail.MailTrap.Username = getString("MAILTRAP_USERNAME", "7bbad578d57780")
	cfg.Mail.MailTrap.Password = getString("MAILTRAP_PASSWORD", "c81f219c18e6c9")

	// frontend url
	cfg.FrontendURL = getString("FRONTEND_URL", "localhost:4000")

	// auth config
	cfg.Auth.Token.Secret = getString("AUTH_TOKEN_SECRET", "6RCMBDDJuc8aRv7us6zcwx2vvQqjmifyVK8O2Cun164")
	cfg.Auth.Token.Exp = getDuration("TOKEN_EXP", time.Hour*24*3)
	cfg.Auth.Token.Iss = getString("TOKEN_ISS", "wallet")

	// payment processor
	cfg.PaymentProcessor.BaseURL = getString("GATEWAY_BASE_URL", "https://cib.dz")
	cfg.PaymentProcessor.WebhookSecret = getString("WEBHOOK_SECRET", "OC83Vc7lna7sADKHzdWZZiWvVceu3gR5MGuOTgkGisc")

	// ratelimiter
	cfg.Ratelimiter.Enabled = getBool("RATELIMITER_ENABLED", true)
	cfg.Ratelimiter.RequestPerTimeFrame = getInt("RATELIMITER_REQUESTS_COUNT", 20)

	return cfg, nil
}

func getString(key, fallback string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	return val
}

func getInt(key string, fallback int) int {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	valAsInt, err := strconv.Atoi(val)
	if err != nil {
		panic(fmt.Errorf("environment variable %s=%q cannot be converted to an int", key, val))
	}

	return valAsInt
}

func getBool(key string, fallback bool) bool {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	valAsBool, err := strconv.ParseBool(val)
	if err != nil {
		panic(fmt.Errorf("environment variable %s=%q cannot be converted to a bool", key, val))
	}

	return valAsBool
}

func getDuration(key string, fallback time.Duration) time.Duration {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	valAsDuration, err := time.ParseDuration(val)
	if err != nil {
		panic(fmt.Errorf("environment variable %s=%q cannot be converted to a time.Duration", key, val))
	}

	return valAsDuration
}
