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
)

type Service interface {
	CreateCheckoutSession(ctx context.Context, payload CheckoutSessionDTO, user *store.User)(string, error)
	GetCheckoutSession(ctx context.Context, token string) (*store.CheckoutSession, error)
	Pay(ctx context.Context, payload PaymentDTO, user *store.User) error
}

type svc struct {
	store store.Storage
	txManager dbtx.TxManager
}

func NewService(store store.Storage, txManager dbtx.TxManager) Service {
	return &svc{
		store: store,
		txManager: txManager,
	}
}

func (s *svc) CreateCheckoutSession(ctx context.Context, payload CheckoutSessionDTO, user *store.User) (string, error) {
	amount, err := amountInCentimes(payload.Amount)
	if err != nil {
		return "", err
	}

	merchant, err := s.store.Merchants.GetByUserID(ctx, user.ID)
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

	if err := s.store.CheckoutSessions.Create(ctx, session); err != nil {
		return "", err
	}

	return plainToken, nil
}

func (s *svc) GetCheckoutSession(ctx context.Context, token string) (*store.CheckoutSession, error) {
	session, err := s.store.CheckoutSessions.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (s *svc) Pay(ctx context.Context, payload PaymentDTO, user *store.User) error {
	amount, err := amountInCentimes(payload.Amount)
	if err != nil {
		return err
	}
	// get user's balance:
	sourceBalance, err := s.store.Balances.GetByUserID(ctx, user.ID)
	if err != nil {
		return err
	}

	// check sufficient balance:
	if sourceBalance.Balance.Cmp(amount) < 0 {
		return fmt.Errorf("insufficient balance")
	}

	merchantID, _ := uuid.Parse(payload.MerchantID)

	// get merchant balance 
	merchantBalance, err := s.store.Balances.GetByMerchantID(ctx, merchantID)
	if err != nil {
		return err
	}
	
	// debit customer
	sourceBalance.Balance.Sub(sourceBalance.Balance, amount)
	
	// calculate platform fees
	platformFee := calculateFeeCeil(amount)

	// calculate merchant net
	merchantNet := amount.Sub(amount, platformFee)

	// credit merchant 
	merchantBalance.Balance.Add(merchantBalance.Balance, merchantNet)

	// @Revenue
	revenueBalance, err := s.store.Balances.GetByBalanceID(ctx, "@Revenue")
	if err != nil {
		return err
	}

	// credit @Revenue
	revenueBalance.Balance.Add(revenueBalance.Balance, platformFee)

	parentTransaction := &store.Transaction{
		ID: uuid.New(),
		Reference: payload.Reference,
		PreciseAmount: merchantNet,
		Source: sourceBalance.BalanceID,
		Destination: merchantBalance.BalanceID,
		Status: "applied",
		Description: "payment",
		CreatedAt: time.Now(),
	}

	id := uuid.New().String()
	ref := fmt.Sprintf("revenue_%s", id)

	childTransaction := &store.Transaction {
		ID: uuid.New(),
		Reference: ref,
		ParentTransaction: parentTransaction.ID,
		Source: sourceBalance.BalanceID,
		Destination: revenueBalance.BalanceID,
		PreciseAmount: platformFee,
		Status: "applied",
		Description: "platform revenue",
		CreatedAt: time.Now(),
	}

	// atomic operation
	err = s.txManager.WithTx(ctx, func(ctx context.Context) error {
		// record customer -> merchant transaction
		if err := s.store.Transactions.Record(ctx, parentTransaction); err != nil {
			return err
		}

		// update customer balances:
		if err := s.store.Balances.UpdateBalance(ctx, sourceBalance); err != nil {
			return err
		}

		// update merchant balance:
		if err := s.store.Balances.UpdateBalance(ctx, merchantBalance); err != nil {
			return err
		}

		// record customer -> @Revenue transaction
		if err := s.store.Transactions.Record(ctx, childTransaction); err != nil {
			return err
		}

		// update @Revenue balance
		if err := s.store.Balances.UpdateBalance(ctx, revenueBalance); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
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