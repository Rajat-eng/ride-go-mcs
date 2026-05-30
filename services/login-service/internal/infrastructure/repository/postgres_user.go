package repository

import (
	"context"
	"database/sql"
	"ride-sharing/services/login-service/internal/domain"
	"ride-sharing/shared/tracing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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

	return tracing.RunInSpan(ctx, "db", "postgres.users.insert", tracing.DBSpanAttrs("postgresql",
		attribute.String("db.operation", "insert"),
		attribute.String("user.email", user.Email),
	), func(ctx context.Context, _ trace.Span) error {
		return r.db.QueryRowContext(ctx, query,
			user.Email,
			user.PasswordHash,
			user.Name,
			user.PhoneNumber,
			user.Role,
		).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	})
}

func (r *postgresUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, email, COALESCE(password_hash, ''), name, COALESCE(phone_number, ''), COALESCE(role, 'rider'), created_at, updated_at FROM users WHERE email = $1`

	var user *domain.User
	err := tracing.RunInSpan(ctx, "db", "postgres.users.find_by_email", tracing.DBSpanAttrs("postgresql",
		attribute.String("db.operation", "select"),
		attribute.String("user.email", email),
	), func(ctx context.Context, _ trace.Span) error {
		decoded := &domain.User{}
		if err := r.db.QueryRowContext(ctx, query, email).Scan(
			&decoded.ID,
			&decoded.Email,
			&decoded.PasswordHash,
			&decoded.Name,
			&decoded.PhoneNumber,
			&decoded.Role,
			&decoded.CreatedAt,
			&decoded.UpdatedAt,
		); err != nil {
			return err
		}
		user = decoded
		return nil
	})
	return user, err
}

func (r *postgresUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `SELECT id, email, COALESCE(password_hash, ''), name, COALESCE(phone_number, ''), COALESCE(role, 'rider'), created_at, updated_at FROM users WHERE id = $1`

	var user *domain.User
	err := tracing.RunInSpan(ctx, "db", "postgres.users.find_by_id", tracing.DBSpanAttrs("postgresql",
		attribute.String("db.operation", "select"),
		attribute.String("user.id", id),
	), func(ctx context.Context, _ trace.Span) error {
		decoded := &domain.User{}
		if err := r.db.QueryRowContext(ctx, query, id).Scan(
			&decoded.ID,
			&decoded.Email,
			&decoded.PasswordHash,
			&decoded.Name,
			&decoded.PhoneNumber,
			&decoded.Role,
			&decoded.CreatedAt,
			&decoded.UpdatedAt,
		); err != nil {
			return err
		}
		user = decoded
		return nil
	})
	return user, err
}

func (r *postgresUserRepository) GetByGoogleSub(ctx context.Context, googleSub string) (*domain.User, error) {
	query := `
		SELECT u.id, u.email, COALESCE(u.password_hash, ''), u.name, COALESCE(u.phone_number, ''), COALESCE(u.role, 'rider'), u.created_at, u.updated_at
		FROM users u
		JOIN user_identities ui ON ui.user_id = u.id
		WHERE ui.provider = 'google' AND ui.provider_subject = $1`

	var user *domain.User
	err := tracing.RunInSpan(ctx, "db", "postgres.users.find_by_google_sub", tracing.DBSpanAttrs("postgresql",
		attribute.String("db.operation", "select"),
		attribute.String("identity.provider", "google"),
	), func(ctx context.Context, _ trace.Span) error {
		decoded := &domain.User{}
		if err := r.db.QueryRowContext(ctx, query, googleSub).Scan(
			&decoded.ID,
			&decoded.Email,
			&decoded.PasswordHash,
			&decoded.Name,
			&decoded.PhoneNumber,
			&decoded.Role,
			&decoded.CreatedAt,
			&decoded.UpdatedAt,
		); err != nil {
			return err
		}
		user = decoded
		return nil
	})
	return user, err
}

func (r *postgresUserRepository) LinkGoogleIdentity(ctx context.Context, userID, googleSub, avatarURL string, emailVerified bool) error {
	query := `
		INSERT INTO user_identities (user_id, provider, provider_subject, avatar_url, email_verified)
		VALUES ($1, 'google', $2, $3, $4)
		ON CONFLICT (user_id, provider)
		DO UPDATE SET provider_subject = EXCLUDED.provider_subject, avatar_url = EXCLUDED.avatar_url, email_verified = EXCLUDED.email_verified, updated_at = NOW()`

	return tracing.RunInSpan(ctx, "db", "postgres.user_identities.upsert", tracing.DBSpanAttrs("postgresql",
		attribute.String("db.operation", "upsert"),
		attribute.String("user.id", userID),
		attribute.String("identity.provider", "google"),
	), func(ctx context.Context, _ trace.Span) error {
		_, err := r.db.ExecContext(ctx, query, userID, googleSub, avatarURL, emailVerified)
		return err
	})
}
