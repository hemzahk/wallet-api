package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hemzahk/wallet-api/docs"
	"github.com/hemzahk/wallet-api/internal/auth"
	"github.com/hemzahk/wallet-api/internal/dbtx"
	"github.com/hemzahk/wallet-api/internal/gateway"
	"github.com/hemzahk/wallet-api/internal/health"
	"github.com/hemzahk/wallet-api/internal/mailer"
	"github.com/hemzahk/wallet-api/internal/middlewares"
	"github.com/hemzahk/wallet-api/internal/payments"
	"github.com/hemzahk/wallet-api/internal/ratelimiter"
	"github.com/hemzahk/wallet-api/internal/store"
	"github.com/hemzahk/wallet-api/internal/users"
	"github.com/hemzahk/wallet-api/internal/wallets"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"go.uber.org/zap"
)

type application struct {
	config config
	logger *zap.SugaredLogger
	store store.Storage
	txManager dbtx.TxManager
	mailer mailer.Client
	authenticator auth.Authenticator
	gateway gateway.PaymentGateway
	limiter ratelimiter.Limiter
}

type config struct {
	addr string
	env  string
	db dbConfig
	apiURL string
	frontendURL string
	mail mailConfig
	auth authConfig
	gateway gatewayConfig
	ratelimiter ratelimiter.Config
}

type dbConfig struct {
	addr         string
	maxOpenConns int
	maxIdleConns int
	maxIdleTime  string	
}

type mailConfig struct {
	mailTrap  mailTrapConfig
	fromEmail string
	exp       time.Duration
}

type mailTrapConfig struct {
	username string
	password string
}

type authConfig struct {
	token tokenConfig
}

type tokenConfig struct {
	secret string
	exp time.Duration
	iss string
}

type gatewayConfig struct {
	webhookSecret string
	baseURL string
}

func (app *application) mount() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.ClientIPFromRemoteAddr)
	r.Use(middleware.Timeout(60 * time.Second))

	rateLimiterMiddleware := middlewares.NewRateLimiterMiddleware(app.limiter)

	userService := users.NewService(app.store.Users,
									app.store.Identities, 
									app.store.Balances,
									app.store.Merchants,
									app.txManager, 
									app.mailer, 
									app.authenticator, 
									app.logger)
	userHandler := users.NewHandler(userService)
	authMiddleware := middlewares.NewAuthMiddleware(app.authenticator, app.store.Users, app.store.Roles)
	r.Use(rateLimiterMiddleware.RateLimiterMiddleware)
	r.Route("/api/v1", func(r chi.Router) {
		docsURL := fmt.Sprintf("%s/swagger/doc.json", app.config.addr)
		r.Get("/swagger/*", httpSwagger.Handler(httpSwagger.URL(docsURL)))
		healthCheckHandler := health.NewHandler(health.Config{
			Env: app.config.env,
			Version: version,
		})
		
		r.Get("/health", healthCheckHandler.HealthCheck)

		r.Route("/users", func(r chi.Router) {
			r.Put("/activate/{token}", userHandler.ActivateUser)
		})

		r.Route("/merchants", func(r chi.Router) {
			r.Put("/activate/{token}", userHandler.ActivateMerchant)
		})

		walletService := wallets.NewService(app.store.Transactions, app.store.Balances, app.store.WebhookEventIDs, app.txManager, app.gateway)
		walletHandler := wallets.NewHandler(walletService, app.gateway)

		r.Post("/webhooks/topup", walletHandler.TopupWebhook)
		r.Post("/webhooks/payout", walletHandler.PayoutWebhook)
		
		r.Route("/authentication", func(r chi.Router) {
			r.Post("/user", userHandler.RegisterUser)
			r.Post("/merchant", userHandler.RegisterMerchant)
			r.Post("/token", userHandler.CreateToken)
		})

		idempotencyMiddleware := middlewares.NewIdempotencyMiddleware(app.store.IdempotencyKeys)

		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.AuthTokenMiddleware)
			r.Get("/wallet", walletHandler.GetWallet)
			r.Get("/wallet/transactions", walletHandler.GetTransactionHistory)
			
			paymentService := payments.NewService(app.store.CheckoutSessions,
												  app.store.Merchants,
												  app.store.Transactions,
												  app.store.Balances, 
												  app.txManager,
												)
			paymentHandler := payments.NewHandler(paymentService)
			r.Post("/payments/checkout-sessions", authMiddleware.CheckRequiredRole("merchant", paymentHandler.CreateCheckoutSession))
			r.Get("/payments/checkout-sessions/{token}", authMiddleware.CheckRequiredRole("customer", paymentHandler.GetCheckoutSession))

			r.Group(func(r chi.Router) {
				r.Use(idempotencyMiddleware.Wrap)
				r.Post("/wallet/topup",authMiddleware.CheckRequiredRole("customer", walletHandler.Topup) )
				r.Post("/wallet/transfer", authMiddleware.CheckRequiredRole("customer", walletHandler.Transfer))
				r.Post("/wallet/withdraw",authMiddleware.CheckRequiredRole("customer", walletHandler.Withdraw) )
				r.Post("/payments/checkout-sessions/{token}/pay",authMiddleware.CheckRequiredRole("customer", paymentHandler.Pay))
			})
		})
	})

	return r
}

func (app *application) run(mux http.Handler) error {
	
	docs.SwaggerInfo.Version = version
	docs.SwaggerInfo.Host = app.config.apiURL
	docs.SwaggerInfo.BasePath = "/api/v1"

	srv := &http.Server{
		Addr: app.config.addr,
		Handler: mux,
		WriteTimeout: time.Second * 30,
		ReadTimeout: time.Second * 30,
		IdleTimeout: time.Minute,
	}

	shutdown := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)

		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		ctx, cancel := context.WithTimeout(context.Background(), 15 *time.Second)
		defer cancel()

		app.logger.Infow("signal caught", "signal", s.String())

		shutdown <- srv.Shutdown(ctx)
	}()

	app.logger.Infow("server has started", "addr", app.config.addr, "env", app.config.env)

	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdown
	if err != nil {
		return err
	}

	app.logger.Infow("server has stopped", "addr", app.config.addr, "env", app.config.env)

	return nil
}