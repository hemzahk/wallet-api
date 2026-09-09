package store

import (
	"database/sql"
)

type Storage struct {
	Users Users
	Roles Roles
	Identities Identities
	Balances Balances
	Transactions Transactions
	Merchants Merchants
	CheckoutSessions CheckoutSessions
	RefundRequests RefundRequests
	IdempotencyKeys IdempotencyKeys
	WebhookEventIDs WebhookEventIDs
}

func NewStorage(db *sql.DB) Storage {
	return Storage{
		Users : &UserStore{db},
		Identities: &IdentityStore{db},
		Transactions: &TransactionStore{db},
		Balances: &BalanceStore{db},
		Merchants: &MerchantStore{db},
		CheckoutSessions: &CheckoutSessionStore{db},
		RefundRequests: &RefundRequestStore{db},
		IdempotencyKeys: &IdempotencyKeyStore{db},
		WebhookEventIDs: &WebhookEventIDStore{db},
		Roles: &RoleStore{db},
	}
}

