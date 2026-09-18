package main

import (
	"context"
	"errors"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/hemzahk/wallet-api/config"
	"github.com/hemzahk/wallet-api/internal/database"
	"github.com/hemzahk/wallet-api/internal/dbtx"
	"github.com/hemzahk/wallet-api/internal/refunds"
	"github.com/hemzahk/wallet-api/internal/store"
	"go.uber.org/zap"
)

func main() {
	logger := zap.Must(zap.NewProduction()).Sugar()
	defer logger.Sync()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.New(
		cfg.Db.Addr,
		cfg.Db.MaxOpenConns,
		cfg.Db.MaxIdleConns,
		cfg.Db.MaxIdleTime,
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
