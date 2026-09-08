package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/hemzahk/wallet-api/internal/dbtx"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrDuplicateEmail = errors.New("a user with that email already exists")
	ErrUserNotFound = errors.New("user not found")
	ErrInvitationNotFound = errors.New("user invitation not found or expired")
)

type Users interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	Update(ctx  context.Context, user *User) error
	Delete(ctx context.Context, userID uuid.UUID) error
	
	CreateUserInvitation(ctx context.Context, token string, userID uuid.UUID, exp time.Duration) error
	GetUserFromInvitation(ctx context.Context, token string) (*User, error)
	DeleteUserInvitation(ctx context.Context, userID uuid.UUID) error
}

type User struct {
	ID uuid.UUID `json:"id"`
	Email string `json:"email"`
	Password password `json:"-"`
	IdentityID uuid.UUID `json:"identity_id"`
	RoleID int64 `json:"role_id"`
	Role Role `json:"role"`
	IsActive bool `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type password struct {
	text *string
	hash []byte
}

func (p *password) Set(text string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(text), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	p.text = &text
	p.hash = hash

	return nil
}

func (p *password) Compare(text string) error {
	return bcrypt.CompareHashAndPassword(p.hash, []byte(text))
}

type UserStore struct {
	db *sql.DB
}

func (s *UserStore) Create(ctx context.Context, user *User) error {
	dbtx := dbtx.ExtractTx(ctx, s.db)

	query := `
		INSERT INTO users (id, email, password, identity_id, is_active, role_id, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
	`
	
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	_, err := dbtx.ExecContext(ctx, 
							 query, 
							 &user.ID,
							 &user.Email,
							 &user.Password.hash,
							 &user.IdentityID,
							 &user.IsActive,
							 &user.RoleID,
							 &user.CreatedAt,
							)		
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		default:
			return err
		}
	}

	return nil
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, password, identity_id, is_active, created_at
		FROM users
		WHERE email = $1 AND is_active = true
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	user := &User{}
	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Password.hash,
		&user.IdentityID, 
		&user.IsActive,
		&user.CreatedAt,
	)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, ErrUserNotFound
		default:
			return nil, err
		}
	}

	return user, nil
}

func (s *UserStore) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
		SELECT users.id, email, identity_id, is_active, role_id, created_at, roles.*
		FROM users
		JOIN roles ON (users.role_id = roles.id)
		WHERE users.id = $1 AND users.is_active = true
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	user := &User{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email, 
		&user.IdentityID,
		&user.IsActive,
		&user.RoleID,
		&user.CreatedAt,
		&user.Role.ID,
		&user.Role.Name,
		&user.Role.Description,
	)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, ErrUserNotFound
		default:
			return nil, err
		}
	}

	return user, nil
}

func (s *UserStore) Update(ctx  context.Context, user *User) error {
	dbtx := dbtx.ExtractTx(ctx, s.db)
	query := `
		UPDATE users SET email = $1, is_active = $2 WHERE id = $3
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	_, err := dbtx.ExecContext(ctx, query, user.Email, user.IsActive, user.ID)
	if err != nil {
		return err
	}

	return nil
}

func (s *UserStore) Delete(ctx context.Context, userID uuid.UUID) error {
	dbtx := dbtx.ExtractTx(ctx, s.db)
	query := `
		DELETE FROM users
		WHERE id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	_, err := dbtx.ExecContext(ctx, query, userID)
	if err != nil {
		return err
	}

	return nil
}

func (s *UserStore) CreateUserInvitation(ctx context.Context, token string, userID uuid.UUID, exp time.Duration) error {
	dbtx := dbtx.ExtractTx(ctx, s.db)

	query := `
		INSERT INTO user_invitations (token, user_id, expiry)
		VALUES ($1, $2, $3)
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	_, err := dbtx.ExecContext(ctx, query, token, userID, time.Now().Add(exp))
	if err != nil {
		return err
	}

	return nil
}

func (s *UserStore) GetUserFromInvitation(ctx context.Context, token string) (*User, error) {
	dbtx := dbtx.ExtractTx(ctx, s.db)
	query := `
		SELECT u.id, u.email, u.identity_id, u.is_active, u.created_at
		FROM users u
		JOIN user_invitations ui ON (u.id = ui.user_id)
		WHERE token = $1 AND expiry > $2
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	hash := sha256.Sum256([]byte(token))
	hashToken := hex.EncodeToString(hash[:])

	user := &User{}
	err := dbtx.QueryRowContext(ctx, query,hashToken, time.Now()).Scan(
		&user.ID,
		&user.Email,
		&user.IdentityID, 
		&user.IsActive,
		&user.CreatedAt,
	)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, ErrInvitationNotFound
		default:
			return nil, err
		}
	}

	return user, nil
}

func (s *UserStore) DeleteUserInvitation(ctx context.Context, userID uuid.UUID) error {
	dbtx := dbtx.ExtractTx(ctx, s.db)
	query := `
		DELETE FROM user_invitations 
		WHERE user_id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	_, err := dbtx.ExecContext(ctx, query, userID)
	if err != nil {
		return err
	}

	return nil
}