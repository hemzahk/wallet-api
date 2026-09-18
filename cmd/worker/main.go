package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hemzahk/wallet-api/internal/database"
	"github.com/hemzahk/wallet-api/internal/dbtx"
	"github.com/hemzahk/wallet-api/internal/env"
	"github.com/hemzahk/wallet-api/internal/refunds"
	"github.com/hemzahk/wallet-api/internal/store"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	logger := zap.Must(zap.NewProduction()).Sugar()
	defer logger.Sync()

	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		logger.Fatalw("loading .env file", "error", err)
	}

	db, err := database.New(
		env.GetString("DB_ADDR"),
		env.GetInt("DB_MAX_OPEN_CONNS"),
		env.GetInt("DB_MAX_IDLE_CONNS"),
		env.GetString("DB_MAX_IDLE_TIME"),
	)
	if err != nil {
		logger.Fatalw("connecting to database", "error", err)
	}
	defer db.Close()

	storage := store.NewStorage(db)
	worker := refunds.NewWorker(
		storage.RefundRequests,
		storage.Transactions,
		storage.Balances,
		dbtx.NewTxManager(db),
		5*time.Second,
		logger,
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("refund worker started")
	if err := worker.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Fatalw("refund worker stopped", "error", err)
	}
	logger.Info("refund worker stopped")
}
