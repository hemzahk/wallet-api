package wallets

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/hemzahk/wallet-api/internal/dbtx"
	"github.com/hemzahk/wallet-api/internal/gateway"
	"github.com/hemzahk/wallet-api/internal/store"
)

var (
	MaxTopupAmount *big.Int = big.NewInt(10000000)
	MaxWithdrawalAmount *big.Int = big.NewInt(5000000)
	MaxTransferAmount *big.Int = big.NewInt(5000000)

	ErrInvalidAmount = errors.New("invalid DZD amount")
	ErrMaxTopupAmountExceeded = errors.New("maximum topup amount exceeded")
	ErrTopupAlreadyCompleted = errors.New("topup already completed")
	ErrTopupAlreadyFailed =  errors.New("topup already failed")

	ErrMaxTransferAmountExceeded = errors.New("maximum transfer amount exceeded")

	ErrMaxWithdrawalAmountExceeded = errors.New("maximum withdrawal amount exceeded")
	ErrWithdrawalAlreadyCompleted = errors.New("withdrawal already completed")
	ErrWithdrawalAlreadyFailed = errors.New("withdrawal already failed")

	ErrInsufficientBalance = errors.New("insufficient balance")

	ErrGatewayFailure = errors.New("gateway error: failed")

	ErrWebhookAlreadyProcessed = errors.New("webhook already processed")
)


type Service interface {
	Topup(ctx context.Context, payload TopupDTO, user *store.User) (*gateway.GatewaySession,error)
	TopupWebhook(ctx context.Context, payload TopupWebhookDTO) error
	Withdraw(ctx context.Context, payload WithdrawalDTO, user *store.User) (*gateway.PayoutSession, error)
	PayoutWebhook(ctx context.Context, payload PayoutWebhookDTO) error
	Transfer(ctx context.Context, payload TransferDTO, user *store.User) error
	GetWallet(ctx context.Context, userID uuid.UUID) (*big.Float, error)
	GetTransactionHistory(ctx context.Context, identityID uuid.UUID) ([]Transaction, error)
}

type svc struct {
	transactions store.Transactions
	balances store.Balances
	webhookEventIDs store.WebhookEventIDs
	txManager dbtx.TxManager
	gateway gateway.PaymentGateway
}

func NewService(transactions store.Transactions,
				balances store.Balances,
				webhookEventIDs store.WebhookEventIDs,
	 			txManager dbtx.TxManager, 
				gateway gateway.PaymentGateway) Service {
	return &svc{
		transactions: transactions,
		balances: balances,
		webhookEventIDs: webhookEventIDs,
		txManager: txManager,
		gateway: gateway,
	}
}

func (s *svc) Topup(ctx context.Context, payload TopupDTO, user *store.User) (*gateway.GatewaySession, error) {
	amount, err := amountInCentimes(payload.Amount)
	if err != nil {
		return nil, err
	}

	if MaxTopupAmount.Cmp(amount) < 0 {
		return nil, ErrMaxTopupAmountExceeded
	} 
		
	session, err := s.gateway.InitiatePayment(ctx, amount)

	ref := fmt.Sprintf("topup_%s", session.GatewayRef)

	err = s.txManager.WithTx(ctx, func(ctx context.Context) error {
		destinationBalance, err := s.balances.GetByIdentityID(ctx, user.IdentityID)
		if err != nil {
			return fmt.Errorf("fetching destination balance: %w", err)
		}

		sourceBalance, err := s.balances.GetByBalanceID(ctx, "@World")
		if err != nil {
			return fmt.Errorf("fetching source balance: %w", err)
		}

		transaction := &store.Transaction{
			ID: uuid.New(),
			Reference: ref,
			PreciseAmount: amount,
			Source: sourceBalance.BalanceID,
			Destination: destinationBalance.BalanceID,
			Status: "inflight",
			Description: "topup",
			CreatedAt: time.Now(),
		}

		if err := s.transactions.Record(ctx, transaction); err != nil {
			return fmt.Errorf("recording transaction: %w", err)
		}

		destinationBalance.InflightCreditBalance.Add(destinationBalance.InflightCreditBalance, amount)

		if err := s.balances.UpdateBalance(ctx, destinationBalance); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("topup failed: %w", err)
	}

	return &session, nil
}

func (s *svc) TopupWebhook(ctx context.Context, payload TopupWebhookDTO) error {
	ref := fmt.Sprintf("topup_%s", payload.GatewayRef)

	parentTransaction, err := s.transactions.GetByRef(ctx, ref)
	if err != nil {
		return fmt.Errorf("fetching parent transaction: %w", err)
	}
	
	switch parentTransaction.Status {
	case "applied":
		return ErrTopupAlreadyCompleted
	case "rejected", "void":
		return ErrTopupAlreadyFailed
	}

	var newStatus string
	switch payload.Status {
		case "succeeded":
			newStatus = "applied"
		case "failed":
			newStatus = "void"
		default:
			return fmt.Errorf("unknown webhook status %s", payload.Status)
	}
	
	err = s.txManager.WithTx(ctx, func(ctx context.Context) error {
		isNew, err := s.webhookEventIDs.MarkProcessed(ctx, payload.SourceID, payload.EventID)
		if err != nil {
			return err
		}

		if !isNew {
			return ErrWebhookAlreadyProcessed
		}
		
		destinationBalance, err := s.balances.GetByBalanceID(ctx, parentTransaction.Destination)
		if err != nil {
			return fmt.Errorf("fetching destination balance: %w", err)
		}

		childTransaction := &store.Transaction{
			ID: uuid.New(),
			ParentTransaction: parentTransaction.ID,
			PreciseAmount: parentTransaction.PreciseAmount,
			Reference: fmt.Sprintf("topup_%s", uuid.New().String()),
			Source: parentTransaction.Source,
			Destination: parentTransaction.Destination,
			Status: newStatus,
			Description: parentTransaction.Description,
			CreatedAt: time.Now(),
		}

		if err := s.transactions.Record(ctx, childTransaction); err != nil {
			return fmt.Errorf("recording transaction: %w", err)
		}

		destinationBalance.InflightCreditBalance.Sub(destinationBalance.InflightCreditBalance, parentTransaction.PreciseAmount)
		if newStatus == "applied" {
			destinationBalance.Balance.Add(destinationBalance.Balance, parentTransaction.PreciseAmount)
		}

		if err := s.balances.UpdateBalance(ctx, destinationBalance); err != nil {
			return err 
		}

		return nil
	})

	if errors.Is(err, ErrWebhookAlreadyProcessed) {
		return err
	}

	if err != nil {
		return fmt.Errorf("processsing topup webhook: %w", err)
	}

	if newStatus == "void" {
		return ErrGatewayFailure
	}
	
	return nil
}

func (s *svc) Withdraw(ctx context.Context, payload WithdrawalDTO, user *store.User) (*gateway.PayoutSession, error) {
	amount, err := amountInCentimes(payload.Amount)
	if err != nil {
		return nil, err
	}

	if MaxWithdrawalAmount.Cmp(amount) < 0 {
		return nil, ErrMaxWithdrawalAmountExceeded
	}

	session, err := s.gateway.InitiatePayout(ctx, amount)
	if err != nil {
		return nil, err
	}

	ref := fmt.Sprintf("withdrawal_%s", session.GatewayRef)

	err = s.txManager.WithTx(ctx, func(ctx context.Context) error {
		sourceBalance, err := s.balances.GetByIdentityID(ctx, user.IdentityID)
		if err != nil {
			return fmt.Errorf("fetching source balance: %w", err)
		}

		destinationBalance, err := s.balances.GetByBalanceID(ctx, "@World")
		if err != nil {
			return fmt.Errorf("fetching destination balance: %w", err)
		}

		// check sufficient balance
		if sourceBalance.Balance.Cmp(amount) < 0 {
			return ErrInsufficientBalance
		}

		transaction := &store.Transaction{
			ID: uuid.New(),
			Reference: ref,
			PreciseAmount: amount,
			Source: sourceBalance.BalanceID,
			Destination: destinationBalance.BalanceID,
			Status: "inflight",
			Description: "withdrawal",
			CreatedAt: time.Now(),
		}

		if err := s.transactions.Record(ctx, transaction); err != nil {
			return fmt.Errorf("recording transaction: %w", err)
		}

		// fund reservation
		sourceBalance.Balance.Sub(sourceBalance.Balance, amount)
		sourceBalance.InflightDebitBalance.Add(sourceBalance.InflightDebitBalance, amount)

		if err := s.balances.UpdateBalance(ctx, sourceBalance); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("withdrawal failed: %w", err)
	}

	return &session, nil
}

func (s *svc) PayoutWebhook(ctx context.Context, payload PayoutWebhookDTO) error {
	ref := fmt.Sprintf("withdrawal_%s", payload.GatewayRef)
	parentTransaction, err := s.transactions.GetByRef(ctx, ref)
	if err != nil {
		return fmt.Errorf("fetching transaction: %w", err)
	}

	switch parentTransaction.Status {
	case "applied":
		return ErrTopupAlreadyCompleted
	case "rejected", "void":
		return ErrTopupAlreadyFailed
	}

	var newStatus string
	switch payload.Status {
		case "succeeded":
			newStatus = "applied"
		case "failed":
			newStatus = "void"
		default:
			return fmt.Errorf("unknown webhook status %s", payload.Status)
	}

	err = s.txManager.WithTx(ctx, func(ctx context.Context) error {
		isNew, err := s.webhookEventIDs.MarkProcessed(ctx, payload.SourceID, payload.EventID)
		if err != nil {
			return err
		}

		if !isNew {
			return ErrWebhookAlreadyProcessed
		}

		sourceBalance, err := s.balances.GetByBalanceID(ctx, parentTransaction.Source)
		if err != nil {
			return fmt.Errorf("fetching source balance: %w", err)
		}

		childTransaction := &store.Transaction{
			ID: uuid.New(),
			ParentTransaction: parentTransaction.ID,
			PreciseAmount: parentTransaction.PreciseAmount,
			Reference: fmt.Sprintf("withdrawal_%s", uuid.New().String()),
			Source: parentTransaction.Source,
			Destination: parentTransaction.Destination,
			Status: newStatus,
			Description: parentTransaction.Description,
			CreatedAt: time.Now(),
		}

		if err := s.transactions.Record(ctx, childTransaction); err != nil {
			return fmt.Errorf("recording transaction: %w", err)
		}

		sourceBalance.InflightDebitBalance.Sub(sourceBalance.InflightDebitBalance, parentTransaction.PreciseAmount)
		if newStatus == "void" {
			sourceBalance.Balance.Add(sourceBalance.Balance, parentTransaction.PreciseAmount)
		}

		if err := s.balances.UpdateBalance(ctx, sourceBalance); err != nil {
			return err
		}

		return nil
	})

	if errors.Is(err, ErrWebhookAlreadyProcessed) {
		return err
	}

	if err != nil {
		return fmt.Errorf("processing withdrawal webhook: %w", err)
	}

	if newStatus == "void" {
		return ErrGatewayFailure
	}

	return nil 
}

func (s *svc) Transfer(ctx context.Context, payload TransferDTO, user *store.User) error {
	amount, err := amountInCentimes(payload.Amount)
	if err != nil {
		return err
	}

	if MaxTransferAmount.Cmp(amount) < 0 {
		return ErrMaxTransferAmountExceeded
	}

	err = s.txManager.WithTx(ctx, func(ctx context.Context) error {
		sourceBalance, err := s.balances.GetByIdentityID(ctx, user.IdentityID)
		if err != nil {
			return fmt.Errorf("fetching source balance: %w", err)
		}

		// check for sufficient balance
		if sourceBalance.Balance.Cmp(amount) < 0 {
			return  ErrInsufficientBalance
		}

		destinationBalance, err := s.balances.GetByEmail(ctx, payload.Email)
		if err != nil {
			return fmt.Errorf("fetching destination balance: %w", err)
		}

		// check source != destination
		if sourceBalance.BalanceID == destinationBalance.BalanceID {
			return fmt.Errorf("you can't transfer money to yourself")
		}

		// preventing direct money transfers to merchants (customer -> merchant)
		if destinationBalance.LedgerID == "merchant_ledger_id" {
			return fmt.Errorf("you can't transfer money to a merchant")
		}

		sourceBalance.Balance.Sub(sourceBalance.Balance, amount)
		destinationBalance.Balance.Add(destinationBalance.Balance, amount)

		transaction := &store.Transaction{
			ID: uuid.New(),
			Reference: payload.Reference, // TODO: extract from idempotency header directly
			PreciseAmount: amount,
			Source: sourceBalance.BalanceID,
			Destination: destinationBalance.BalanceID,
			Status: "applied",
			Description: "P2P Transfer",
			CreatedAt: time.Now(),
		}

		if err := s.transactions.Record(ctx, transaction); err != nil {
			return fmt.Errorf("recording transaction: %w", err)
		}

		if err := s.balances.UpdateBalance(ctx, sourceBalance); err != nil {
			return err
		}

		if err := s.balances.UpdateBalance(ctx, destinationBalance); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("transfer failed: %w", err)
	}

	return nil
}

func (s *svc) GetWallet(ctx context.Context, userID uuid.UUID) (*big.Float, error) {
	balance, err := s.balances.GetByUserID(ctx, userID); 
	if err != nil {
		return nil, err
	}

	dzdBalance := toFloat(balance.Balance)

	return dzdBalance, nil
}

type Transaction struct {
	Amount string `json:"amount"`
	TransactionRef string `json:"transaction_ref"`
	Description string `json:"description"`
	Type string `json:"type"`
	CreatedAt string `json:"created_at"`
}

func (s *svc) GetTransactionHistory(ctx context.Context, identityID uuid.UUID) ([]Transaction, error) {
	balance, err := s.balances.GetByIdentityID(ctx, identityID)
	if err != nil {
		return nil, err
	}

	transactions, err := s.transactions.GetByIdentityID(ctx, identityID)
	if err != nil {
		return nil, err
	}

	var txns []Transaction
	for _, val := range transactions {
		amountAsFloat := toFloat(val.PreciseAmount)
		t := Transaction{
			TransactionRef: val.Reference,
			Amount: amountAsFloat.String(),
			CreatedAt: val.CreatedAt.String(),
			Description: val.Description,
		}

		if val.Source == balance.BalanceID {
			t.Type = "debit"
		} else if val.Destination == balance.BalanceID {
			t.Type = "credit"
		}

		txns = append(txns, t)
	}

	return txns, nil
}

func amountInCentimes(amount string) (*big.Int, error) {
	rat, ok := new(big.Rat).SetString(amount)
	if !ok {
		return nil, ErrInvalidAmount
	}

	multiplier := big.NewRat(100,1)
	rat.Mul(rat, multiplier)

	centimes := new(big.Int).Set(rat.Num())

	return centimes, nil
}

func toFloat(val *big.Int) *big.Float {
	bf := new(big.Float).SetInt(val)

	// Divide by 100
	divisor := big.NewFloat(100)
	return new(big.Float).Quo(bf, divisor)
}