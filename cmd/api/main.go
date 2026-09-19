package main

import (
	"github.com/hemzahk/wallet-api/config"
	"github.com/hemzahk/wallet-api/internal/auth"
	"github.com/hemzahk/wallet-api/internal/database"
	"github.com/hemzahk/wallet-api/internal/dbtx"
	"github.com/hemzahk/wallet-api/internal/gateway"
	"github.com/hemzahk/wallet-api/internal/mailer"
	"github.com/hemzahk/wallet-api/internal/ratelimiter"
	"github.com/hemzahk/wallet-api/internal/store"
	"go.uber.org/zap"
)

const version = "0.0.1"

// @title Wallet API
// @description This is digital wallet API.

// @contact.name API Support
// @contact.email hemza.hk18@gmail.com

// @BasePath /api/v1

// @securityDefinitions.apiKey ApiKeyAuth
// @in header
// @name Authorization
// @description

func main() {
	// Logger
	logger := zap.Must(zap.NewProduction()).Sugar()
	defer logger.Sync()

	cfg, err := config.LoadWithEnvConfig()
	if err != nil {
		logger.Fatal(err)
	}

	db, err := database.New(
		cfg.DB.Addr,
		cfg.DB.MaxOpenConns,
		cfg.DB.MaxIdleConns,
		cfg.DB.MaxIdleTime,
	)

	if err != nil {
		logger.Fatal(err)
	}

	defer db.Close()
	logger.Info("database connection pool established")

	store := store.NewStorage(db)
	txManager := dbtx.NewTxManager(db)

	mailtrap, err := mailer.NewMailTrapClient(cfg.Mail.MailTrap.Username, cfg.Mail.MailTrap.Password, cfg.Mail.FromEmail)
	if err != nil {
		logger.Fatal(err)
	}

	jwtAuthenticator := auth.NewJWTAuthenticator(
		cfg.Auth.Token.Secret,
		cfg.Auth.Token.Iss,
		cfg.Auth.Token.Iss,
	)

	gateway := gateway.NewStubGateway(cfg.PaymentProcessor.WebhookSecret, cfg.PaymentProcessor.BaseURL)

	ratelimiter := ratelimiter.NewFixedWindowLimiter(
		cfg.Ratelimiter.RequestPerTimeFrame,
		cfg.Ratelimiter.TimeFrame,
	)

	app := &application{
		config:        cfg,
		logger:        logger,
		store:         store,
		txManager:     txManager,
		mailer:        mailtrap,
		authenticator: jwtAuthenticator,
		gateway:       gateway,
		limiter:       ratelimiter,
	}

	mux := app.mount()
	logger.Fatal(app.run(mux))
}
