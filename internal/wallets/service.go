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
	ErrInvalidAmount = errors.New("invalid DZD amount")
	ErrTopupAlreadyCompleted = errors.New("topup already completed")
	ErrTopupAlreadyFailed =  errors.New("topup already failed")

	ErrWithdrawalAlreadyCompleted = errors.New("withdrawal already completed")
	ErrWithdrawalAlreadyFailed = errors.New("withdrawal already failed")

	ErrGatewayFailure = errors.New("gateway error: topup failed")

	ErrWebhookAlreadyProcessed = errors.New("webhook already processed")
)

type Service interface {
	Topup(ctx context.Context, payload TopupDTO, user *store.User) (*gateway.GatewaySession,error)
	TopupWebhook(ctx context.Context, payload TopupWebhookDTO) error
	Withdraw(ctx context.Context, payload WithdrawalDTO, user *store.User) (*gateway.PayoutSession, error)
	PayoutWebhook(ctx context.Context, payload PayoutWebhookDTO) error
	Transfer(ctx context.Context, payload TransferDTO, user *store.User) error
	GetWallet(ctx context.Context, userID uuid.UUID) (*big.Float, error)
	GetTransactionHistory(ctx context.Context, identityID uuid.UUID) (*store.Balance, []store.Transaction, error)
}

type svc struct {
	store store.Storage
	txManager dbtx.TxManager
	gateway gateway.PaymentGateway
}

func NewService(store store.Storage, txManager dbtx.TxManager, gateway gateway.PaymentGateway) Service {
	return &svc{
		store: store,
		txManager: txManager,
		gateway: gateway,
	}
}

func (s *svc) Topup(ctx context.Context, payload TopupDTO, user *store.User) (*gateway.GatewaySession, error) {
	amount, err := amountInCentimes(payload.Amount)
	if err != nil {
		return nil, err
	}

	destinationBalance, err := s.store.Balances.GetByIdentityID(ctx, user.IdentityID)
	if err != nil {
		return nil, err // add some proper error handling
	}

	sourceBalance, err := s.store.Balances.GetByBalanceID(ctx, "@World")
	if err != nil {
		return nil, err // add some proper error handling
	}

	session, err := s.gateway.InitiatePayment(ctx, amount)

	ref := fmt.Sprintf("topup_%s", session.GatewayRef)
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

	// record transaction
	if err := s.store.Transactions.Record(ctx, transaction); err != nil {
			return nil, err
	}

	return &session, nil
}

func (s *svc) TopupWebhook(ctx context.Context, payload TopupWebhookDTO) error {

	isDuplicate, err := s.store.WebhookEventIDs.IsDuplicate(ctx, payload.SourceID, payload.EventID)
	if err != nil {
		return err
	}

	if isDuplicate {
		return ErrWebhookAlreadyProcessed
	}

	ref := fmt.Sprintf("topup_%s", payload.GatewayRef)
	parentTransaction, err := s.store.Transactions.GetByRef(ctx, ref)
	if err != nil {
		return err
	}

	// payment gateway failure
	if payload.Status == "failed" {
		childTransaction := &store.Transaction{
			ID: uuid.New(),
			ParentTransaction: parentTransaction.ID,
			PreciseAmount: parentTransaction.PreciseAmount,
			Reference: fmt.Sprintf("topup_%s", uuid.New().String()),
			Source: parentTransaction.Source,
			Destination: parentTransaction.Destination,
			Status: "rejected",
			Description: parentTransaction.Description,
			CreatedAt: time.Now(),
		}

		if err := s.store.Transactions.Record(ctx, childTransaction); err != nil {
			return err
		}

		return ErrGatewayFailure
	}

	// already applied transaction or rejected transaction : return an error no further processing
	switch parentTransaction.Status {
		case "applied": 
			return ErrTopupAlreadyCompleted
		case "rejected":
			return ErrTopupAlreadyFailed
	}

	balance, err := s.store.Balances.GetByBalanceID(ctx, parentTransaction.Destination)
	if err != nil {
		return err
	}

	err = s.txManager.WithTx(ctx, func(ctx context.Context) error {
		// insert new transaction to update status 
		childTransaction := &store.Transaction{
			ID: uuid.New(),
			ParentTransaction: parentTransaction.ID,
			PreciseAmount: parentTransaction.PreciseAmount,
			Reference: fmt.Sprintf("topup_%s", uuid.New().String()),
			Source: parentTransaction.Source,
			Destination: parentTransaction.Destination,
			Status: "applied",
			Description: parentTransaction.Description,
			CreatedAt: time.Now(),
		}

		if err := s.store.Transactions.Record(ctx,childTransaction ); err != nil {
			return err
		}

		// update customer's balance: 
		balance.Balance.Add(balance.Balance, parentTransaction.PreciseAmount)
		if err := s.store.Balances.UpdateBalance(ctx, balance); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *svc) Withdraw(ctx context.Context, payload WithdrawalDTO, user *store.User) (*gateway.PayoutSession, error) {
	amount, err := amountInCentimes(payload.Amount)
	if err != nil {
		return nil, err
	}

	sourceBalance, err := s.store.Balances.GetByIdentityID(ctx, user.IdentityID)
	if err != nil {
		return nil, err
	}

	// check sufficient balance
	if sourceBalance.Balance.Cmp(amount) < 0 {
		return nil, err // return a comprehensive error
	}

	destinationBalance, err := s.store.Balances.GetByBalanceID(ctx, "@World")

	session, err := s.gateway.InitiatePayout(ctx, amount)
	if err != nil {
		return nil,err
	}

	// don't update balance until confirmation from payout webhook
	// sourceBalance.Balance.Sub(sourceBalance.Balance, amount)
	// destinationBalance.Balance.Add(destinationBalance.Balance, amount)
	ref := fmt.Sprintf("withdrawal_%s", session.GatewayRef)
	transaction := &store.Transaction{
		ID: uuid.New(),
		Reference: ref,
		PreciseAmount: amount,
		Source: sourceBalance.BalanceID,
		Destination: destinationBalance.BalanceID,
		Status: "inflight",
		Description: "Withdrawal",
		CreatedAt: time.Now(),
	}

	err = s.txManager.WithTx(ctx, func(ctx context.Context) error {
		if err := s.store.Transactions.Record(ctx, transaction); err != nil {
			return err
		}

		// don't update balance until confirmation from payout webhook
		// if err := s.store.Balances.UpdateBalance(ctx, sourceBalance); err != nil {
		// 	return err
		// }

		// if err := s.store.Balances.UpdateBalance(ctx, destinationBalance); err != nil {
		// 	return err
		// }

		return nil
	})
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *svc) PayoutWebhook(ctx context.Context, payload PayoutWebhookDTO) error {

	isDuplicate, err := s.store.WebhookEventIDs.IsDuplicate(ctx, payload.SourceID, payload.EventID)
	if err != nil {
		return err
	}

	if isDuplicate {
		return ErrWebhookAlreadyProcessed
	}

	ref := fmt.Sprintf("withdrawal_%s", payload.GatewayRef)
	parentTransaction, err := s.store.Transactions.GetByRef(ctx, ref)
	if err != nil {
		return err
	}

	// payment gateway failure
	if payload.Status == "failed" {
		childTransaction := &store.Transaction{
			ID: uuid.New(),
			ParentTransaction: parentTransaction.ID,
			PreciseAmount: parentTransaction.PreciseAmount,
			Reference: fmt.Sprintf("withdrawal_%s", uuid.New().String()),
			Source: parentTransaction.Source,
			Destination: parentTransaction.Destination,
			Status: "rejected",
			Description: parentTransaction.Description,
			CreatedAt: time.Now(),
		}

		if err := s.store.Transactions.Record(ctx, childTransaction); err != nil {
			return err
		}

		return ErrGatewayFailure
	}
	
	// already applied transaction or rejected transaction : return an error no further processing
	switch parentTransaction.Status {
		case "applied": 
			return ErrWithdrawalAlreadyCompleted
		case "rejected":
			return ErrWithdrawalAlreadyFailed
	}

	balance, err := s.store.Balances.GetByBalanceID(ctx, parentTransaction.Source)
	if err != nil {
		return err
	}


	err = s.txManager.WithTx(ctx, func(ctx context.Context) error {
		// insert new transaction to update status 
		childTransaction := &store.Transaction{
			ID: uuid.New(),
			ParentTransaction: parentTransaction.ID,
			PreciseAmount: parentTransaction.PreciseAmount,
			Reference: fmt.Sprintf("withdrawal_%s", uuid.New().String()),
			Source: parentTransaction.Source,
			Destination: parentTransaction.Destination,
			Status: "applied",
			Description: parentTransaction.Description,
			CreatedAt: time.Now(),
		}

		if err := s.store.Transactions.Record(ctx,childTransaction ); err != nil {
			return err
		}

		// update customer's balance: 
		balance.Balance.Sub(balance.Balance, parentTransaction.PreciseAmount)
		if err := s.store.Balances.UpdateBalance(ctx, balance); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil 
}

func (s *svc) Transfer(ctx context.Context, payload TransferDTO, user *store.User) error {
	amount, err := amountInCentimes(payload.Amount)
	if err != nil {
		return err
	}

	sourceBalance, err := s.store.Balances.GetByIdentityID(ctx, user.IdentityID)
	if err != nil {
		return err
	}

	// check for sufficient balance
	if sourceBalance.Balance.Cmp(amount) < 0 {
		return  fmt.Errorf("insufficient balance")
	}

	destinationBalance, err := s.store.Balances.GetByEmail(ctx, payload.Email)
	if err != nil {
		return err
	}

	// check source != destination
	if sourceBalance.BalanceID == destinationBalance.BalanceID {
		return fmt.Errorf("you can't transfer money to yourself")
	}

	sourceBalance.Balance.Sub(sourceBalance.Balance, amount)
	destinationBalance.Balance.Add(destinationBalance.Balance, amount)

	transaction := &store.Transaction{
		ID: uuid.New(),
		Reference: payload.Reference,
		PreciseAmount: amount,
		Source: sourceBalance.BalanceID,
		Destination: destinationBalance.BalanceID,
		Status: "applied",
		Description: "P2P Transfer",
		CreatedAt: time.Now(),
	}
	
	err = s.txManager.WithTx(ctx, func(ctx context.Context) error {
		if err := s.store.Transactions.Record(ctx, transaction); err != nil {
			return err
		}

		if err := s.store.Balances.UpdateBalance(ctx, sourceBalance); err != nil {
			return err
		}

		if err := s.store.Balances.UpdateBalance(ctx, destinationBalance); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *svc) GetWallet(ctx context.Context, userID uuid.UUID) (*big.Float, error) {
	balance, err := s.store.Balances.GetByUserID(ctx, userID); 
	if err != nil {
		return nil, err
	}

	dzdBalance := toFloat(balance.Balance)

	return dzdBalance, nil
}

func (s *svc) GetTransactionHistory(ctx context.Context, identityID uuid.UUID) (*store.Balance, []store.Transaction, error) {
	balance, err := s.store.Balances.GetByIdentityID(ctx, identityID)
	if err != nil {
		return nil, nil, err
	}

	transactions, err := s.store.Transactions.GetByIdentityID(ctx, identityID)
	if err != nil {
		return nil, nil, err
	}

	return balance, transactions, nil
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