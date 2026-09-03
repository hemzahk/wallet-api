package main

import (
	"time"

	"github.com/hemzahk/wallet-api/internal/auth"
	"github.com/hemzahk/wallet-api/internal/database"
	"github.com/hemzahk/wallet-api/internal/dbtx"
	"github.com/hemzahk/wallet-api/internal/env"
	"github.com/hemzahk/wallet-api/internal/gateway"
	"github.com/hemzahk/wallet-api/internal/mailer"
	"github.com/hemzahk/wallet-api/internal/store"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

const version = "0.0.2"

// @title Wallet API
// @description This is digital wallet API.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @BasePath /api/v1

// @securityDefinitions.apiKey ApiKeyAuth
// @in header
// @name Authorization
// @description

func main() {
	// Logger
	logger := zap.Must(zap.NewProduction()).Sugar()
	defer logger.Sync()
	
	// Load env file
	err := godotenv.Load()
	if err != nil {
		logger.Fatal("Error loading .env file")
	}

	cfg := config{
		addr: env.GetString("ADDR"),
		env: env.GetString("ENV"),
		apiURL: env.GetString("EXTERNAL_URL"),
		db: dbConfig{
			addr: env.GetString("DB_ADDR"),
			maxOpenConns: env.GetInt("DB_MAX_OPEN_CONNS"),
			maxIdleConns: env.GetInt("DB_MAX_IDLE_CONNS"),
			maxIdleTime: env.GetString("DB_MAX_IDLE_TIME"),
		},
		mail: mailConfig{
			exp: time.Hour * 24 * 3,
			fromEmail: env.GetString("FROM_EMAIL"),
			mailTrap: mailTrapConfig{
				username: env.GetString("MAILTRAP_USERNAME"),
				password: env.GetString("MAILTRAP_PASSWORD"),
			},
		},
		frontendURL: env.GetString("FRONTEND_URL"),
		auth: authConfig{
			token: tokenConfig{
				secret: env.GetString("AUTH_TOKEN_SECRET"),
				exp: time.Hour * 24 * 3, // 3 days
				iss: "wallet", 
			},
		},
		gateway: gatewayConfig{
			webhookSecret: env.GetString("WEBHOOK_SECRET"),
			baseURL: env.GetString("GATEWAY_BASE_URL"),
		},
	}

	db, err := database.New(
		cfg.db.addr,
		cfg.db.maxOpenConns,
		cfg.db.maxIdleConns,
		cfg.db.maxIdleTime,
	)

	if err != nil {
		logger.Fatal(err)
	}

	defer db.Close()
	logger.Info("database connection pool established")

	store := store.NewStorage(db)
	txManager := dbtx.NewTxManager(db)

	mailtrap, err := mailer.NewMailTrapClient(cfg.mail.mailTrap.username, cfg.mail.mailTrap.password, cfg.mail.fromEmail)
	if err != nil {
		logger.Fatal(err)
	}

	jwtAuthenticator := auth.NewJWTAuthenticator(
		cfg.auth.token.secret,
		cfg.auth.token.iss,
		cfg.auth.token.iss,
	)

	gateway := gateway.NewStubGateway(cfg.gateway.webhookSecret, cfg.gateway.baseURL)

	app := &application{
		config: cfg,
		logger: logger,
		store: store,
		txManager: txManager,
		mailer: mailtrap,
		authenticator: jwtAuthenticator,
		gateway: gateway,
	}

	mux := app.mount()
	logger.Fatal(app.run(mux))
}