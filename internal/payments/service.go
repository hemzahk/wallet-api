package payments

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/hemzahk/wallet-api/internal/dbtx"
	"github.com/hemzahk/wallet-api/internal/store"
)

var (
	ErrInvalidAmount = errors.New("invalid DZD amount")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrNonRefundable = errors.New("transaction is non-refundable")
	ErrNonRefundableExpired = errors.New("transaction is non-refundable expired")
)

type Service interface {
	CreateCheckoutSession(ctx context.Context, payload CheckoutSessionDTO, user *store.User)(string, error)
	GetCheckoutSession(ctx context.Context, token string) (*store.CheckoutSession, error)
	Pay(ctx context.Context, token string, user *store.User) error
	RequestRefund(ctx context.Context, payload RefundDTO) error
}

type svc struct {
	checkoutSessions store.CheckoutSessions
	merchants store.Merchants
	transactions store.Transactions
	balances store.Balances
	refundRequests store.RefundRequests
	txManager dbtx.TxManager
}

func NewService(checkoutSessions store.CheckoutSessions,
				merchants store.Merchants,
				transactions store.Transactions,
				balances store.Balances,
				refundRequests store.RefundRequests,
				txManager dbtx.TxManager) Service{
	return &svc{
		checkoutSessions: checkoutSessions,
		merchants: merchants,
		transactions :transactions,
		balances: balances,
		refundRequests: refundRequests,
		txManager: txManager,
	}
}

func (s *svc) CreateCheckoutSession(ctx context.Context, payload CheckoutSessionDTO, user *store.User) (string, error) {
	amount, err := amountInCentimes(payload.Amount)
	if err != nil {
		return "", err
	}

	merchant, err := s.merchants.GetByUserID(ctx, user.ID)
	if err != nil {
		return "", err
	}

	plainToken := uuid.New().String()

	hash := sha256.Sum256([]byte(plainToken))
	hashToken := hex.EncodeToString(hash[:])	
	
	session := &store.CheckoutSession{
		ID: uuid.New(),
		Token: hashToken,
		MerchantID: merchant.ID,
		Amount: amount,
		ExpiresAt: time.Now().Add(time.Minute*5),
		CreatedAt: time.Now(),
	}

	if err := s.checkoutSessions.Create(ctx, session); err != nil {
		return "", err
	}

	return plainToken, nil
}

func (s *svc) GetCheckoutSession(ctx context.Context, token string) (*store.CheckoutSession, error) {
	session, err := s.checkoutSessions.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (s *svc) Pay(ctx context.Context, token string, user *store.User) error {
	err := s.txManager.WithTx(ctx, func(ctx context.Context) error {
		session, err := s.checkoutSessions.GetByToken(ctx, token)
		if err != nil {
			return err
		}

		sourceBalance, err := s.balances.GetByUserID(ctx, user.ID)
		if err != nil {
			return fmt.Errorf("fetching balance: %w", err)
		}

		// check sufficient balance:
		if sourceBalance.Balance.Cmp(session.Amount) < 0 {
			return ErrInsufficientBalance
		}

		merchantBalance, err := s.balances.GetByMerchantID(ctx, session.MerchantID)
		if err != nil {
			return fmt.Errorf("fetching balance: %w", err)
		}

		// @Revenue
		revenueBalance, err := s.balances.GetByBalanceID(ctx, "@Revenue")
		if err != nil {
			return fmt.Errorf("fetching balance: %w", err)
		}

		// debit customer
		sourceBalance.Balance.Sub(sourceBalance.Balance, session.Amount)

		// calculate platform fees
		platformFee := calculateFeeCeil(session.Amount)

		// calculate merchant net
		merchantNet := session.Amount.Sub(session.Amount, platformFee)

		// credit merchant 
		merchantBalance.Balance.Add(merchantBalance.Balance, merchantNet)

		// credit @Revenue
		revenueBalance.Balance.Add(revenueBalance.Balance, platformFee)

		parentTransaction := &store.Transaction{
			ID: uuid.New(),
			Reference: fmt.Sprintf("payment_%s", uuid.New().String()),
			PreciseAmount: merchantNet,
			Source: sourceBalance.BalanceID,
			Destination: merchantBalance.BalanceID,
			Status: "applied",
			Description: "payment",
			CreatedAt: time.Now(),
		}

		childTransaction := &store.Transaction{
			ID: uuid.New(),
			Reference: fmt.Sprintf("revenue_%s", uuid.New().String()),
			ParentTransaction: parentTransaction.ID,
			Source: sourceBalance.BalanceID,
			Destination: revenueBalance.BalanceID,
			PreciseAmount: platformFee,
			Status: "applied",
			Description: "platform revenue",
			CreatedAt: time.Now(),
		}

		// record customer -> merchant transaction
		if err := s.transactions.Record(ctx, parentTransaction); err != nil {
			return fmt.Errorf("recording transaction: %w", err)
		}

		// record customer -> @Revenue transaction
		if err := s.transactions.Record(ctx, childTransaction); err != nil {
			return fmt.Errorf("recording transaction: %w", err)
		}

		// update customer balances:
		if err := s.balances.UpdateBalance(ctx, sourceBalance); err != nil {
			return err
		}

		// update merchant balance:
		if err := s.balances.UpdateBalance(ctx, merchantBalance); err != nil {
			return err
		}

		// update @Revenue balance
		if err := s.balances.UpdateBalance(ctx, revenueBalance); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}
	
	return nil
}

func (s *svc) RequestRefund(ctx context.Context, payload RefundDTO) error {
	err := s.txManager.WithTx(ctx, func(ctx context.Context) error {
		// fetch transaction
		transaction, err := s.transactions.GetByRef(ctx, payload.TransactionRef)
		if err != nil {
			return fmt.Errorf("fetching transaction: %w", err)
		}

		// check that transaction is refundable (status == applied)
		if transaction.Status != "applied" {
			return ErrNonRefundable
		}

		// refundable within 30 days:
		if time.Since(transaction.CreatedAt) > time.Hour*24*30 {
			return ErrNonRefundableExpired
		}

		// record the refund request
		refundRequest := &store.RefundRequest{
			ID: uuid.New(),
			TransactionRef: payload.TransactionRef,
			Status: "pending",
			Description: "don't want to use the service anymore",
			CreatedAt: time.Now(),
		}

		if err := s.refundRequests.Create(ctx, refundRequest); err != nil {
			return fmt.Errorf("recording refund request: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("refund request failed: %w", err)
	}

	return nil
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

func calculateFeeCeil(amount *big.Int) *big.Int {
	fee := new(big.Int).Mul(amount, big.NewInt(15))
	fee.Add(fee, big.NewInt(999))
	fee.Div(fee, big.NewInt(1000))
	return fee
}

// func (s *svc) Refund(ctx context.Context, payload RefundDTO) error {
// 	err := s.txManager.WithTx(ctx, func(ctx context.Context) error {
// 		transaction, err := s.transactions.GetByRef(ctx, payload.TransactionRef)
// 		if err != nil {
// 			return fmt.Errorf("fetching transaction: %w", err)
// 		}

// 		merchantBalance, err := s.balances.GetByBalanceID(ctx, transaction.Destination)
// 		if err != nil {
// 			return fmt.Errorf("fetching merchant balance: %w", err)
// 		}

// 		customerBalance, err := s.balances.GetByBalanceID(ctx, transaction.Source)
// 		if err != nil {
// 			return fmt.Errorf("fetching customer balance: %w", err)
// 		}

// 		// check merchant has sufficient balance:
// 		if merchantBalance.Balance.Cmp(transaction.PreciseAmount) < 0 {
// 			return ErrInsufficientBalance
// 		}

// 		// debit merchant:
// 		merchantBalance.Balance.Sub(merchantBalance.Balance, transaction.PreciseAmount)

// 		// credit merchant:
// 		customerBalance.Balance.Add(customerBalance.Balance, transaction.PreciseAmount)

// 		// update balances:
// 		if err := s.balances.UpdateBalance(ctx, merchantBalance); err != nil {
// 			return fmt.Errorf("updating balance: %w", err)
// 		}

// 		if err := s.balances.UpdateBalance(ctx, customerBalance); err != nil {
// 			return fmt.Errorf("updating balance: %w", err)
// 		}

// 		// record transaction:
// 		refundTransaction := &store.Transaction{
// 			ID: uuid.New(),
// 			PreciseAmount: transaction.PreciseAmount,
// 			Reference: fmt.Sprintf("%s_refund", transaction.Reference),
// 			Source: transaction.Destination,
// 			Destination: transaction.Source,
// 			Status: "applied",
// 			Description: "refund",
// 			CreatedAt: time.Now(),
// 		}

// 		if err := s.transactions.Record(ctx, refundTransaction); err != nil {
// 			return fmt.Errorf("recording transaction: %w", err)
// 		}
// 		return nil
// 	})
// 	if err != nil {
// 		return fmt.Errorf("refund failed: %w", err)
// 	}

// 	return nil
// }