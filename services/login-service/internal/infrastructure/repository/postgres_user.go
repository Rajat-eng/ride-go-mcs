package repository

import (
	"context"
	"database/sql"
	"ride-sharing/services/login-service/internal/domain"
)

type postgresUserRepository struct {
	db *sql.DB
}

func NewPostgresUserRepository(db *sql.DB) domain.UserRepository {
	return &postgresUserRepository{db: db}
}

func (r *postgresUserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (email, password_hash, name, phone_number)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRowContext(ctx, query,
		user.Email, user.PasswordHash, user.Name, user.PhoneNumber,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *postgresUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, email, password_hash, name, phone_number, created_at, updated_at FROM users WHERE email = $1`

	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.PhoneNumber, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *postgresUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `SELECT id, email, password_hash, name, phone_number, created_at, updated_at FROM users WHERE id = $1`

	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.PhoneNumber, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}
