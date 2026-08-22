package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/hemzahk/wallet-api/internal/dbtx"
)
const (
	IdentityTypeIndividual = "individual"
	IdentityTypeOrganization = "organization"

	CategoryCustomer = "customer"
	CategoryMerchant = "merchant"
)
type Identity struct {
	ID uuid.UUID `json:"id"`
	FirstName string `json:"first_name"`
	LastName string `json:"last_name"`
	EmailAddress string `json:"email_address"`
	PhoneNumber string `json:"phone_number"`
	DOB *time.Time `json:"dob"`
	Gender string `json:"gender"`
	State string `json:"state"`
	City string `json:"city"`
	Street string `json:"street"`
	PostCode string `json:"post_code"`
	IdentityType string `json:"identity_type"`
	OrganizationName string `json:"organization_name"`
	Category string `json:"category"`
}


type IdentityStore struct {
	db *sql.DB
}

func (s *IdentityStore) Create(ctx context.Context, identity *Identity) error {
	dbtx := dbtx.ExtractTx(ctx, s.db)

	query := `
		INSERT INTO identities (
							id,
							first_name,
							last_name,
							email_address,
							phone_number,
							dob,
							gender,
							state,
							city,
							street,
							post_code,
							identity_type,
							organization_name,
							category
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	_, err := dbtx.ExecContext(ctx,
							 query,
							 &identity.ID, 
							 &identity.FirstName,
							 &identity.LastName,
							 &identity.EmailAddress,
							 &identity.PhoneNumber,
							 &identity.DOB,
							 &identity.Gender,
							 &identity.State,
							 &identity.City,
							 &identity.Street,
							 &identity.PostCode,
							 &identity.IdentityType,
							 &identity.OrganizationName,
							 &identity.Category, 
							)	
	if err != nil {
		return err
	}

	return nil
}

func (s *IdentityStore) Delete(ctx context.Context, identityID uuid.UUID) error {
	dbtx := dbtx.ExtractTx(ctx, s.db)
	
	query := `
		DELETE FROM identities
		WHERE id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	_, err := dbtx.ExecContext(ctx, query, identityID)
	if err != nil {
		return err
	}

	return nil
}