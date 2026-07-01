package db

import (
	"context"
	"fmt"

	"github.com/gabrielgcosta/ticketblast-core/db/sqlc"
	"github.com/gabrielgcosta/ticketblast-core/internal/entity"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type PostgresUserRepository struct {
	queries *sqlc.Queries
}

func NewPostgresUserRepository(queries *sqlc.Queries) *PostgresUserRepository {
	return &PostgresUserRepository{queries: queries}
}

func (r *PostgresUserRepository) Create(ctx context.Context, name, email, passwordHash string, role entity.UserRole) (*entity.User, error) {
	params := sqlc.CreateUserParams{
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         string(role),
	}

	row, err := r.queries.CreateUser(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create user in db: %w", err)
	}

	return &entity.User{
		ID:           fromUUID(row.ID),
		Name:         row.Name,
		Email:        row.Email,
		PasswordHash: passwordHash, // CreateUser query doesn't return password_hash
		Role:         parseUserRole(row.Role),
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}, nil
}

func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	row, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &entity.User{
		ID:           fromUUID(row.ID),
		Name:         row.Name,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		Role:         parseUserRole(row.Role),
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}, nil
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id string) (*entity.User, error) {
	uid, err := toUUID(id)
	if err != nil {
		return nil, fmt.Errorf("invalid uuid format: %w", err)
	}

	row, err := r.queries.GetUserByID(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return &entity.User{
		ID:           fromUUID(row.ID),
		Name:         row.Name,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		Role:         parseUserRole(row.Role),
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}, nil
}

// Helpers for UUID conversion
func toUUID(s string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}

func fromUUID(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}

// Helper to convert interface{} user_role from sqlc to entity.UserRole
func parseUserRole(r interface{}) entity.UserRole {
	if s, ok := r.(string); ok {
		return entity.UserRole(s)
	}
	if b, ok := r.([]byte); ok {
		return entity.UserRole(b)
	}
	return entity.RoleCustomer
}
