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

var (
	ErrUnauthorized = errors.New("unauthorized")
)

type Service interface {
	Register(context.Context, RegisterDTO) (string, error)
	Activate(ctx context.Context, token string) error

	RegisterMerchant(ctx context.Context, payload RegisterMerchantDTO) (string, error)
	ActivateMerchant(ctx context.Context, token string) error
	
	CreateToken(ctx context.Context, payload CreateUserTokenPayload) (string, error)
}

type svc struct {
	store store.Storage
	txManager dbtx.TxManager
	mailer mailer.Client
	auth auth.Authenticator
	logger *zap.SugaredLogger
}

func NewService(store store.Storage, txManager dbtx.TxManager, mailer mailer.Client, auth auth.Authenticator, logger *zap.SugaredLogger) Service {
	return &svc{
		store: store,
		txManager: txManager,
		mailer: mailer,
		auth: auth,
		logger : logger,
	}
}



func (s *svc) Register(ctx context.Context, payload RegisterDTO) (string, error) {
	user := &store.User{
		ID: uuid.New(),
		Email: payload.Email,
		IdentityID: uuid.New(),
		IsActive: false,
		CreatedAt: time.Now(),
	}
	
	if err := user.Password.Set(payload.Password); err != nil {
		return "", err
	}

	identity := &store.Identity{
		ID: user.IdentityID,
		FirstName: payload.FirstName,
		LastName: payload.LastName,
		EmailAddress: payload.Email,
		PhoneNumber: payload.PhoneNumber,
		Gender: payload.Gender,
		State: payload.State,
		City: payload.City,
		Street: payload.Street,
		PostCode: payload.PostCode,
		IdentityType: store.IdentityTypeIndividual,
		Category: store.CategoryCustomer,
	}

	plainToken := uuid.New().String()

	hash := sha256.Sum256([]byte(plainToken))
	hashToken := hex.EncodeToString(hash[:])


	err := s.txManager.WithTx(ctx, func(ctx context.Context) error {
		if err := s.store.Identities.Create(ctx, identity); err != nil {
			return err
		}

		if err := s.store.Users.Create(ctx, user); err != nil {
			return err
		}

		if err := s.store.Users.CreateUserInvitation(ctx, hashToken, user.ID, time.Hour * 24 * 3); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	activationURL := fmt.Sprintf("http://localhost:4000/confirm/%s", plainToken)

	vars := struct {
		FirstName string
		ActivationURL string
	} {
		FirstName: payload.FirstName,
		ActivationURL: activationURL,
	}

	status, err := s.mailer.Send(mailer.UserWelcomeTemplate, payload.FirstName, payload.Email, vars, true)
	if err != nil {
		s.logger.Errorw("error sending welcome email", "error", err)
		
		// delete user
		err := s.txManager.WithTx(ctx, func(ctx context.Context) error {
			if err := s.store.Users.Delete(ctx, user.ID); err != nil {
				return err
			}

			if err := s.store.Users.DeleteUserInvitation(ctx, user.ID); err != nil {
				return err
			}

			if err := s.store.Identities.Delete(ctx, user.IdentityID); err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			s.logger.Errorw("error deleting user", "error", err)
			return "", err
		}

		return "", err
	}

	s.logger.Infow("Email sent", "status code", status)

	return plainToken, nil
}

func (s *svc) Activate(ctx context.Context, token string) error {
	return s.txManager.WithTx(ctx, func(ctx context.Context) error {
		user, err := s.store.Users.GetUserFromInvitation(ctx, token) 
		if err != nil {
			return err
		}

		user.IsActive = true
		if err := s.store.Users.Update(ctx, user); err != nil {
			return err
		}

		balance := &store.Balance{
			ID: uuid.New(),
			BalanceID: generateUUIDWithSuffix("bln"),
			IdentityID: user.IdentityID,
			LedgerID: "customer_ledger_id",
			CreatedAt: time.Now(),
		}
		if err := s.store.Balances.CreateBalance(ctx, balance); err != nil {
			return err
		}

		if err := s.store.Users.DeleteUserInvitation(ctx, user.ID); err != nil {
			return err
		}

		return nil
	})
}

func (s *svc) RegisterMerchant(ctx context.Context, payload RegisterMerchantDTO) (string, error) {
	user := &store.User{
		ID: uuid.New(),
		Email: payload.Email,
		IdentityID: uuid.New(),
		IsActive: false,
		CreatedAt: time.Now(),
	}
	
	if err := user.Password.Set(payload.Password); err != nil {
		return "", err
	}

	identity := &store.Identity{
		ID: user.IdentityID,
		FirstName: payload.FirstName,
		LastName: payload.LastName,
		EmailAddress: payload.Email,
		PhoneNumber: payload.PhoneNumber,
		Gender: payload.Gender,
		State: payload.State,
		City: payload.City,
		Street: payload.Street,
		PostCode: payload.PostCode,
		IdentityType: store.IdentityTypeOrganization,
		Category: store.CategoryMerchant,
		OrganizationName: payload.BusinessName,
	}

	merchant := &store.Merchant{
		ID: uuid.New(),
		UserID: user.ID,
		BusinessName: payload.BusinessName,
		CreatedAt: time.Now(),
	}

	plainToken := uuid.New().String()

	hash := sha256.Sum256([]byte(plainToken))
	hashToken := hex.EncodeToString(hash[:])


	err := s.txManager.WithTx(ctx, func(ctx context.Context) error {
		if err := s.store.Identities.Create(ctx, identity); err != nil {
			return err
		}

		if err := s.store.Users.Create(ctx, user); err != nil {
			return err
		}

		if err := s.store.Users.CreateUserInvitation(ctx, hashToken, user.ID, time.Hour * 24 * 3); err != nil {
			return err
		}

		if err := s.store.Merchants.Create(ctx, merchant); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	activationURL := fmt.Sprintf("http://localhost:4000/confirm/%s", plainToken)

	vars := struct {
		FirstName string
		ActivationURL string
	} {
		FirstName: payload.FirstName,
		ActivationURL: activationURL,
	}

	status, err := s.mailer.Send(mailer.UserWelcomeTemplate, payload.FirstName, payload.Email, vars, true)
	if err != nil {
		s.logger.Errorw("error sending welcome email", "error", err)
		
		// delete user
		err := s.txManager.WithTx(ctx, func(ctx context.Context) error {
			if err := s.store.Users.Delete(ctx, user.ID); err != nil {
				return err
			}

			if err := s.store.Users.DeleteUserInvitation(ctx, user.ID); err != nil {
				return err
			}

			if err := s.store.Identities.Delete(ctx, user.IdentityID); err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			s.logger.Errorw("error deleting user", "error", err)
			return "", err
		}

		return "", err
	}

	s.logger.Infow("Email sent", "status code", status)

	return plainToken, nil
}

func (s *svc) ActivateMerchant(ctx context.Context, token string) error {
	return s.txManager.WithTx(ctx, func(ctx context.Context) error {
		user, err := s.store.Users.GetUserFromInvitation(ctx, token) 
		if err != nil {
			return err
		}

		user.IsActive = true
		if err := s.store.Users.Update(ctx, user); err != nil {
			return err
		}

		balance := &store.Balance{
			ID: uuid.New(),
			BalanceID: generateUUIDWithSuffix("bln"),
			IdentityID: user.IdentityID,
			LedgerID: "merchant_ledger_id",
			CreatedAt: time.Now(),
		}
		if err := s.store.Balances.CreateBalance(ctx, balance); err != nil {
			return err
		}

		if err := s.store.Users.DeleteUserInvitation(ctx, user.ID); err != nil {
			return err
		}

		return nil
	})
}

func (s *svc) CreateToken(ctx context.Context, payload CreateUserTokenPayload) (string, error) {
	user, err := s.store.Users.GetByEmail(ctx, payload.Email)
	if err != nil {
		return "", ErrUnauthorized // add some proper error handling
	}

	if err := user.Password.Compare(payload.Password); err != nil {
		return "", ErrUnauthorized // add some proper error handling
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

func generateUUIDWithSuffix(module string) string {
	id := uuid.New() // Generate a new UUID.
	uuidStr := id.String()
	idWithSuffix := fmt.Sprintf("%s_%s", module, uuidStr) // Append the module as a suffix to the UUID.
	return idWithSuffix
}