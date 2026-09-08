package users

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/hemzahk/wallet-api/internal/auth"
	"github.com/hemzahk/wallet-api/internal/dbtx"
	"github.com/hemzahk/wallet-api/internal/mailer"
	"github.com/hemzahk/wallet-api/internal/store"
	"go.uber.org/zap"
)

const (
	customerRoleID int64 = 1
	merchantRoleID int64 = 2
)

var (
	ErrUnauthorized = errors.New("unauthorized")
)

type Service interface {
	RegisterCustomer(ctx context.Context,req RegisterRequest) (string, error)
	ActivateCustomer(ctx context.Context, token string) error

	RegisterMerchant(ctx context.Context, req RegisterMerchantRequest) (string, error)
	ActivateMerchant(ctx context.Context, token string) error
	
	CreateToken(ctx context.Context, req CreateUserTokenRequest) (string, error)
}

type svc struct {
	users store.Users
	identities store.Identities
	balances store.Balances
	merchants store.Merchants
	txManager dbtx.TxManager
	mailer mailer.Client
	auth auth.Authenticator
	logger *zap.SugaredLogger
}

func NewService(users store.Users,
				identities store.Identities,
				balances store.Balances,
				merchants store.Merchants,
				txManager dbtx.TxManager,
				mailer mailer.Client, 
				auth auth.Authenticator, 
				logger *zap.SugaredLogger) Service {
	return &svc{
		users: users,
		identities: identities,
		balances: balances,
		merchants: merchants,
		txManager: txManager,
		mailer: mailer,
		auth: auth,
		logger : logger,
	}
}

func (s *svc) RegisterCustomer(ctx context.Context, req RegisterRequest) (string, error) {
	user := &store.User{
		ID: uuid.New(),
		Email: req.Email,
		IdentityID: uuid.New(),
		IsActive: false,
		RoleID: customerRoleID,
		CreatedAt: time.Now(),
	}
	
	if err := user.Password.Set(req.Password); err != nil {
		return "", err
	}

	identity := &store.Identity{
		ID: user.IdentityID,
		FirstName: req.FirstName,
		LastName: req.LastName,
		EmailAddress: req.Email,
		PhoneNumber: req.PhoneNumber,
		Gender: req.Gender,
		State: req.State,
		City: req.City,
		Street: req.Street,
		PostCode: req.PostCode,
		IdentityType: store.IdentityTypeIndividual,
		Category: store.CategoryCustomer,
	}

	plainToken := uuid.New().String()

	hash := sha256.Sum256([]byte(plainToken))
	hashToken := hex.EncodeToString(hash[:])


	err := s.txManager.WithTx(ctx, func(ctx context.Context) error {
		if err := s.identities.Create(ctx, identity); err != nil {
			return fmt.Errorf("creating identity: %w", err)
		}

		if err := s.users.Create(ctx, user); err != nil {
			return fmt.Errorf("creating user: %w", err)
		}

		if err := s.users.CreateUserInvitation(ctx, hashToken, user.ID, time.Hour * 24 * 3); err != nil {
			return fmt.Errorf("creating user invitation: %w", err)
		}

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("user creation failed: %w", err)
	}

	activationURL := fmt.Sprintf("http://localhost:4000/confirm/%s", plainToken)

	vars := struct {
		FirstName string
		ActivationURL string
	} {
		FirstName: req.FirstName,
		ActivationURL: activationURL,
	}

	status, err := s.mailer.Send(mailer.UserWelcomeTemplate, req.FirstName, req.Email, vars, true)
	if err != nil {
		s.logger.Errorw("error sending welcome email", "error", err)
		
		// delete user
		err := s.txManager.WithTx(ctx, func(ctx context.Context) error {
			if err := s.users.Delete(ctx, user.ID); err != nil {
				return fmt.Errorf("deleting user: %w", err)
			}

			if err := s.users.DeleteUserInvitation(ctx, user.ID); err != nil {
				return fmt.Errorf("deleting user invitation: %w", err)
			}

			if err := s.identities.Delete(ctx, user.IdentityID); err != nil {
				return fmt.Errorf("deleting identity: %w", err)
			}

			return nil
		})
		if err != nil {
			s.logger.Errorw("error deleting user", "error", err)
			return "", fmt.Errorf("user deletion failed: %w", err)
		}

		return "", err
	}

	s.logger.Infow("Email sent", "status code", status)

	return plainToken, nil
}

func (s *svc) ActivateCustomer(ctx context.Context, token string) error {
	return s.activate(ctx, token, "customer_ledger_id", "customer")
}

func (s *svc) RegisterMerchant(ctx context.Context, req RegisterMerchantRequest) (string, error) {
	user := &store.User{
		ID: uuid.New(),
		Email: req.Email,
		IdentityID: uuid.New(),
		IsActive: false,
		RoleID: merchantRoleID,
		CreatedAt: time.Now(),
	}
	
	if err := user.Password.Set(req.Password); err != nil {
		return "", err
	}

	identity := &store.Identity{
		ID: user.IdentityID,
		FirstName: req.FirstName,
		LastName: req.LastName,
		EmailAddress: req.Email,
		PhoneNumber: req.PhoneNumber,
		Gender: req.Gender,
		State: req.State,
		City: req.City,
		Street: req.Street,
		PostCode: req.PostCode,
		IdentityType: store.IdentityTypeOrganization,
		Category: store.CategoryMerchant,
		OrganizationName: req.BusinessName,
	}

	merchant := &store.Merchant{
		ID: uuid.New(),
		UserID: user.ID,
		BusinessName: req.BusinessName,
		CreatedAt: time.Now(),
	}

	plainToken := uuid.New().String()

	hash := sha256.Sum256([]byte(plainToken))
	hashToken := hex.EncodeToString(hash[:])


	err := s.txManager.WithTx(ctx, func(ctx context.Context) error {
		if err := s.identities.Create(ctx, identity); err != nil {
			return fmt.Errorf("creating identity: %w", err)
		}

		if err := s.users.Create(ctx, user); err != nil {
			return fmt.Errorf("creating user: %w", err)
		}

		if err := s.users.CreateUserInvitation(ctx, hashToken, user.ID, time.Hour * 24 * 3); err != nil {
			return fmt.Errorf("creating user invitation: %w", err)
		}

		if err := s.merchants.Create(ctx, merchant); err != nil {
			return fmt.Errorf("creating merchant: %w", err)
		}

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("merchant creation failed: %w", err)
	}

	activationURL := fmt.Sprintf("http://localhost:4000/confirm/%s", plainToken)

	vars := struct {
		FirstName string
		ActivationURL string
	} {
		FirstName: req.FirstName,
		ActivationURL: activationURL,
	}

	status, err := s.mailer.Send(mailer.UserWelcomeTemplate, req.FirstName, req.Email, vars, true)
	if err != nil {
		s.logger.Errorw("error sending welcome email", "error", err)
		
		// delete user
		err := s.txManager.WithTx(ctx, func(ctx context.Context) error {
			if err := s.users.Delete(ctx, user.ID); err != nil {
				return fmt.Errorf("deleting user: %w", err)
			}

			if err := s.users.DeleteUserInvitation(ctx, user.ID); err != nil {
				return fmt.Errorf("deleting user invitation: %w", err)
			}

			if err := s.identities.Delete(ctx, user.IdentityID); err != nil {
				return fmt.Errorf("deleting user identity: %w", err)
			}

			if err := s.merchants.Delete(ctx, merchant.ID); err != nil {
				return fmt.Errorf("deleting merchant: %w", err)
			}

			return nil
		})
		if err != nil {
			s.logger.Errorw("error deleting merchant", "error", err)
			return "", fmt.Errorf("merchant deletion failed: %w", err)
		}

		return "", err
	}

	s.logger.Infow("Email sent", "status code", status)

	return plainToken, nil
}

func (s *svc) ActivateMerchant(ctx context.Context, token string) error {
	return s.activate(ctx, token, "merchant_ledger_id", "merchant")
}

func (s *svc) CreateToken(ctx context.Context, req CreateUserTokenRequest) (string, error) {
	user, err := s.users.GetByEmail(ctx, req.Email)
	if err != nil {
		return "", ErrUnauthorized 
	}

	if err := user.Password.Compare(req.Password); err != nil {
		return "", ErrUnauthorized 
	}

	claims := jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(time.Hour * 24 * 3).Unix(), // change later pass config
		"iat": time.Now().Unix(),
		"nbf": time.Now().Unix(),
		"iss": "wallet", // change later pass config
		"aud": "wallet", // change later pass config
	}

	token, err := s.auth.GenerateToken(claims)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *svc) activate(ctx context.Context, token, ledgerID, label string) error {
	err := s.txManager.WithTx(ctx, func(ctx context.Context) error {
		user, err := s.users.GetUserFromInvitation(ctx, token)
		if err != nil {
			return fmt.Errorf("fetching user from invitations: %w", err)
		}

		user.IsActive = true
		if err := s.users.Update(ctx, user); err != nil {
			return fmt.Errorf("updating user: %w", err)
		}

		balance := &store.Balance{
			ID: uuid.New(),
			BalanceID: generateUUIDWithSuffix("bln"),
			IdentityID: user.IdentityID,
			LedgerID: ledgerID,
			CreatedAt: time.Now(),
		}

		if err := s.balances.CreateBalance(ctx, balance); err != nil {
			return fmt.Errorf("creating balance: %w", err)
		}

		if err := s.users.DeleteUserInvitation(ctx, user.ID); err != nil {
			return fmt.Errorf("deleting user invitation: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("%s activation failed: %w", label, err)
	}

	return nil
}

func generateUUIDWithSuffix(module string) string {
	id := uuid.New() // Generate a new UUID.
	uuidStr := id.String()
	idWithSuffix := fmt.Sprintf("%s_%s", module, uuidStr) // Append the module as a suffix to the UUID.
	return idWithSuffix
}