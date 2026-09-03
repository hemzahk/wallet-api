package store

import (
	"context"
	"database/sql"
	"time"
)

type Roles interface {
	GetByName(ctx context.Context, roleName string) (*Role, error)
}

type Role struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type RoleStore struct {
	db *sql.DB
}

func (s *RoleStore) GetByName(ctx context.Context, roleName string) (*Role, error) {
	query := `
		SELECT id, name, description FROM roles
		WHERE name = $1
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	var role Role
	err := s.db.QueryRowContext(ctx, query, roleName).Scan(
		&role.ID,
		&role.Name,
		&role.Description,
	)
	if err != nil {
		return  nil, err
	}

	return &role, nil
}