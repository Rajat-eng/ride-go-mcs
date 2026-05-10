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
		INSERT INTO users (email, password_hash, name, phone_number, role)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`

	return r.db.QueryRowContext(ctx, query,
		user.Email,
		user.PasswordHash,
		user.Name,
		user.PhoneNumber,
		user.Role,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *postgresUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, email, COALESCE(password_hash, ''), name, COALESCE(phone_number, ''), COALESCE(role, 'rider'), created_at, updated_at FROM users WHERE email = $1`

	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.PhoneNumber,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *postgresUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `SELECT id, email, COALESCE(password_hash, ''), name, COALESCE(phone_number, ''), COALESCE(role, 'rider'), created_at, updated_at FROM users WHERE id = $1`

	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.PhoneNumber,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *postgresUserRepository) GetByGoogleSub(ctx context.Context, googleSub string) (*domain.User, error) {
	query := `
		SELECT u.id, u.email, COALESCE(u.password_hash, ''), u.name, COALESCE(u.phone_number, ''), COALESCE(u.role, 'rider'), u.created_at, u.updated_at
		FROM users u
		JOIN user_identities ui ON ui.user_id = u.id
		WHERE ui.provider = 'google' AND ui.provider_subject = $1`

	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, googleSub).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.PhoneNumber,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *postgresUserRepository) LinkGoogleIdentity(ctx context.Context, userID, googleSub, avatarURL string, emailVerified bool) error {
	query := `
		INSERT INTO user_identities (user_id, provider, provider_subject, avatar_url, email_verified)
		VALUES ($1, 'google', $2, $3, $4)
		ON CONFLICT (user_id, provider)
		DO UPDATE SET provider_subject = EXCLUDED.provider_subject, avatar_url = EXCLUDED.avatar_url, email_verified = EXCLUDED.email_verified, updated_at = NOW()`

	_, err := r.db.ExecContext(ctx, query, userID, googleSub, avatarURL, emailVerified)
	return err
}
