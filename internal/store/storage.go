package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Storage struct {
	Users interface {
		Create(ctx context.Context, user *User) error
		GetByEmail(ctx context.Context, email string) (*User, error)
		GetByID(ctx context.Context, id uuid.UUID) (*User, error)
		Update(ctx  context.Context, user *User) error
		Delete(ctx context.Context, userID uuid.UUID) error
		
		CreateUserInvitation(ctx context.Context, token string, userID uuid.UUID, exp time.Duration) error
		GetUserFromInvitation(ctx context.Context, token string) (*User, error)
		DeleteUserInvitation(ctx context.Context, userID uuid.UUID) error
	}

	Identities interface {
		Create(ctx context.Context, identity *Identity) error
		Delete(ctx context.Context, identityID uuid.UUID) error
	}

	Transactions interface {
		//RecordTransaction(ctx context.Context, transaction *Transaction) error
		Record(ctx context.Context, transaction *Transaction) error
		GetByRef(ctx context.Context, reference string) (*Transaction, error)
		GetByIdentityID(ctx context.Context, identityID uuid.UUID) ([]Transaction, error)
		// RecordTransactionAndUpdateBalance(ctx context.Context, transaction *Transaction, sourceBalance, destinationBalance *Balance) error
	}

	Balances interface {
		CreateBalance(ctx context.Context, balance *Balance) error
		GetByIdentityID(ctx context.Context, identityID uuid.UUID) (*Balance, error)
		GetByEmail(ctx context.Context, email string) (*Balance, error)
		GetByUserID(ctx context.Context, userID uuid.UUID) (*Balance, error)
		GetByBalanceID(ctx context.Context, balanceID string) (*Balance, error)
		GetByMerchantID(ctx context.Context, merchantID uuid.UUID) (*Balance, error)
		UpdateBalance(ctx context.Context, balance *Balance) error
	}

	Merchants interface {
		Create(ctx context.Context, merchant *Merchant) error
		GetByUserID(ctx context.Context, userID uuid.UUID) (*Merchant, error)
	}

	CheckoutSessions interface {
		Create(ctx context.Context, session *CheckoutSession) error
		GetByToken(ctx context.Context, token string) (*CheckoutSession, error)
	}

	IdempotencyKeys interface {
		Create(ctx context.Context, record IdempotencyKey)  error
		Get(ctx context.Context, key string, userID uuid.UUID) (*IdempotencyKey, error)
	}

	WebhookEventIDs interface {
		IsDuplicate(ctx context.Context, sourceID, eventID string) (bool, error)
	}

	Roles interface {
		GetByName(ctx context.Context, roleName string) (*Role, error)
	}
}

func NewStorage(db *sql.DB) Storage {
	return Storage{
		Users : &UserStore{db},
		Identities: &IdentityStore{db},
		Transactions: &TransactionStore{db},
		Balances: &BalanceStore{db},
		Merchants: &MerchantStore{db},
		CheckoutSessions: &CheckoutSessionStore{db},
		IdempotencyKeys: &IdempotencyKeyStore{db},
		WebhookEventIDs: &WebhookEventIDStore{db},
		Roles: &RoleStore{db},
	}
}

func generateUUIDWithSuffix(module string) string {
	id := uuid.New() // Generate a new UUID.
	uuidStr := id.String()
	idWithSuffix := fmt.Sprintf("%s_%s", module, uuidStr) // Append the module as a suffix to the UUID.
	return idWithSuffix
}
