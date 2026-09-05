package refunds

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hemzahk/wallet-api/internal/dbtx"
	"github.com/hemzahk/wallet-api/internal/store"
	"go.uber.org/zap"
)

var ErrNoPendingRequests = errors.New("no pending refund requests")
var ErrMerchantInsufficientBalance = errors.New("merchant has insufficient balance for refund")

// Worker processes refund requests persisted by the API.
type Worker struct {
	refundRequests store.RefundRequests
	transactions   store.Transactions
	balances       store.Balances
	txManager      dbtx.TxManager
	interval       time.Duration
	logger         *zap.SugaredLogger
}

func NewWorker(
	refundRequests store.RefundRequests,
	transactions store.Transactions,
	balances store.Balances,
	txManager dbtx.TxManager,
	interval time.Duration,
	logger *zap.SugaredLogger,
) *Worker {
	if interval <= 0 {
		interval = time.Second
	}
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}

	return &Worker{
		refundRequests: refundRequests,
		transactions:   transactions,
		balances:       balances,
		txManager:      txManager,
		interval:       interval,
		logger:         logger,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	w.processAvailable(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			w.processAvailable(ctx)
		}
	}
}

func (w *Worker) processAvailable(ctx context.Context) {
	for {
		err := w.ProcessOne(ctx)
		if err != nil {
			if errors.Is(err, ErrNoPendingRequests) {
				w.logger.Debug("no pending refund requests")
				return
			}
			w.logger.Errorw("processing refund request failed", "error", err)
			return
		}
		w.logger.Info("refund request processed")
	}
}

func (w *Worker) ProcessOne(ctx context.Context) error {
	var request *store.RefundRequest
	err := w.txManager.WithTx(ctx, func(ctx context.Context) error {
		var err error
		request, err = w.refundRequests.ClaimPending(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNoPendingRequests
		}
		if err != nil {
			return fmt.Errorf("claiming refund request: %w", err)
		}

		if err := w.refund(ctx, request); err != nil {
			return fmt.Errorf("processing refund request %s: %w", request.ID, err)
		}

		if err := w.refundRequests.MarkCompleted(ctx, request.ID); err != nil {
			return fmt.Errorf("marking refund request %s completed: %w", request.ID, err)
		}

		return nil
	})
	if err == nil || request == nil || !errors.Is(err, ErrMerchantInsufficientBalance) {
		return err
	}

	if markErr := w.refundRequests.MarkFailed(ctx, request.ID, err.Error()); markErr != nil {
		return fmt.Errorf("marking refund request %s failed: %w", request.ID, markErr)
	}

	return err
}

func (w *Worker) refund(ctx context.Context, request *store.RefundRequest) error {
	transaction, err := w.transactions.GetByRef(ctx, request.TransactionRef)
	if err != nil {
		return fmt.Errorf("fetching transaction: %w", err)
	}

	merchantBalance, err := w.balances.GetByBalanceID(ctx, transaction.Destination)
	if err != nil {
		return fmt.Errorf("fetching merchant balance: %w", err)
	}

	customerBalance, err := w.balances.GetByBalanceID(ctx, transaction.Source)
	if err != nil {
		return fmt.Errorf("fetching customer balance: %w", err)
	}

	if merchantBalance.Balance.Cmp(transaction.PreciseAmount) < 0 {
		return ErrMerchantInsufficientBalance
	}

	merchantBalance.Balance.Sub(merchantBalance.Balance, transaction.PreciseAmount)
	customerBalance.Balance.Add(customerBalance.Balance, transaction.PreciseAmount)

	if err := w.balances.UpdateBalance(ctx, merchantBalance); err != nil {
		return fmt.Errorf("updating merchant balance: %w", err)
	}
	if err := w.balances.UpdateBalance(ctx, customerBalance); err != nil {
		return fmt.Errorf("updating customer balance: %w", err)
	}

	refundTransaction := &store.Transaction{
		ID:                uuid.New(),
		ParentTransaction: transaction.ID,
		PreciseAmount:     transaction.PreciseAmount,
		Reference:         transaction.Reference + "_refund",
		Source:            transaction.Destination,
		Destination:       transaction.Source,
		Status:            "applied",
		Description:       "refund",
		CreatedAt:         time.Now(),
	}

	if err := w.transactions.Record(ctx, refundTransaction); err != nil {
		return fmt.Errorf("recording refund transaction: %w", err)
	}
	if err := w.transactions.MarkRefunded(ctx, transaction.Reference); err != nil {
		return fmt.Errorf("marking original transaction refunded: %w", err)
	}

	return nil
}
