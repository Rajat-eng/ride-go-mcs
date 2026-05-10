package domain

import "context"

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	GetByGoogleSub(ctx context.Context, googleSub string) (*User, error)
	LinkGoogleIdentity(ctx context.Context, userID, googleSub, avatarURL string, emailVerified bool) error
}
