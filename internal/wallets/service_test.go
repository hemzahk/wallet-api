package wallets

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/google/uuid"
	txManagerMock "github.com/hemzahk/wallet-api/internal/dbtx/mocks"
	"github.com/hemzahk/wallet-api/internal/gateway"
	"github.com/hemzahk/wallet-api/internal/store"
	storeMock "github.com/hemzahk/wallet-api/internal/store/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTopup_Success(t *testing.T) {
	mockTransactions := new(storeMock.MockTransactions)
	mockBalances := new(storeMock.MockBalances)
	mockTxManager := new(txManagerMock.MockTxManager)
	mockPaymentGateway := new(gateway.MockPaymentGateway)

	mockStore := store.Storage{
		Transactions: mockTransactions,
		Balances: mockBalances,
	}

	payload := TopupDTO{Amount: "10000"}
	amount, err := amountInCentimes(payload.Amount)
	if err != nil {
		t.Fatalf("amountInCentimes returned unexpected error: %v", err)
	}

	user := store.User{
		ID:         uuid.New(),
		Email:      "example@example.com",
		IdentityID: uuid.New(),
	}

	destinationBalance := &store.Balance{
		ID:                     uuid.New(),
		BalanceID:              fmt.Sprintf("bln_%s", uuid.New().String()),
		Balance:                big.NewInt(100000),
		InflightCreditBalance:  big.NewInt(0),
		InflightDebitBalance:   big.NewInt(0),
		Currency:               "DZD",
		LedgerID:               "customer_ledger_id",
		IdentityID:             user.IdentityID,
		Version:                0,
		CreatedAt:              time.Now(),
	}
	initialInflight := new(big.Int).Set(destinationBalance.InflightCreditBalance)

	sourceBalance := &store.Balance{
		ID:                    uuid.New(),
		BalanceID:             "@World",
		Balance:               big.NewInt(0),
		InflightCreditBalance: big.NewInt(0),
		InflightDebitBalance:  big.NewInt(0),
		Currency:              "DZD",
		LedgerID:              "general_ledger_id",
		IdentityID:            uuid.Nil,
		Version:               0,
		CreatedAt:             time.Now(),
	}

	gatewaySession := gateway.GatewaySession{GatewayRef: "ref_123", PaymentURL: "https://pay.test/ref_123"}
	mockPaymentGateway.On("InitiatePayment", mock.Anything, amount).Return(gatewaySession, nil).Once()
	mockTxManager.On("WithTx", mock.Anything, mock.Anything).Return(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	mockBalances.On("GetByIdentityID", mock.Anything, user.IdentityID).Return(destinationBalance, nil).Once()
	mockBalances.On("GetByBalanceID", mock.Anything, "@World").Return(sourceBalance, nil).Once()
	mockTransactions.On("Record", mock.Anything, mock.MatchedBy(func(tx *store.Transaction) bool {
		if tx == nil {
			return false
		}
		return tx.Reference == "topup_ref_123" &&
			tx.Source == sourceBalance.BalanceID &&
			tx.Destination == destinationBalance.BalanceID &&
			tx.Status == "inflight" &&
			tx.Description == "topup" &&
			tx.PreciseAmount.Cmp(amount) == 0
	})).Return(nil).Once()
	mockBalances.On("UpdateBalance", mock.Anything, mock.MatchedBy(func(b *store.Balance) bool {
		if b == nil {
			return false
		}
		expectedInflight := new(big.Int).Add(initialInflight, amount)
		return b.BalanceID == destinationBalance.BalanceID &&
			b.InflightCreditBalance.Cmp(expectedInflight) == 0
	})).Return(nil).Once()

	svc := NewService(mockStore, mockTxManager, mockPaymentGateway)

	result, err := svc.Topup(context.Background(), payload, &user)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, gatewaySession.PaymentURL, result.PaymentURL)
	assert.Equal(t, gatewaySession.GatewayRef, result.GatewayRef)
	assert.Equal(t, 0, destinationBalance.InflightCreditBalance.Cmp(new(big.Int).Add(initialInflight, amount)))

	mockPaymentGateway.AssertExpectations(t)
	mockTxManager.AssertExpectations(t)
	mockBalances.AssertExpectations(t)
	mockTransactions.AssertExpectations(t)
}