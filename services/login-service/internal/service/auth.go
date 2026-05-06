package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"ride-sharing/services/login-service/internal/domain"
	"ride-sharing/services/login-service/pkg/token"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

type AuthService struct {
	repo         domain.UserRepository
	tokenManager *token.Manager
}

func NewAuthService(repo domain.UserRepository, tokenManager *token.Manager) *AuthService {
	return &AuthService{
		repo:         repo,
		tokenManager: tokenManager,
	}
}

func (s *AuthService) Signup(ctx context.Context, email, password, name, phoneNumber string) (*domain.User, string, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	_, err := s.repo.GetByEmail(ctx, email)
	if err == nil {
		return nil, "", "", ErrEmailAlreadyExists
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, "", "", fmt.Errorf("failed to check existing user: %w", err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to hash password: %w", err)
	}

	user := &domain.User{
		Email:        email,
		PasswordHash: string(hashedPassword),
		Name:         name,
		PhoneNumber:  phoneNumber,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, "", "", fmt.Errorf("failed to create user: %w", err)
	}

	accessToken, err := s.tokenManager.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.tokenManager.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return user, accessToken, refreshToken, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*domain.User, string, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", "", ErrInvalidCredentials
		}
		return nil, "", "", fmt.Errorf("failed to get user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", "", ErrInvalidCredentials
	}

	accessToken, err := s.tokenManager.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.tokenManager.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return user, accessToken, refreshToken, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshTokenStr string) (string, string, error) {
	claims, err := s.tokenManager.ValidateRefreshToken(refreshTokenStr)
	if err != nil {
		return "", "", ErrInvalidToken
	}

	accessToken, err := s.tokenManager.GenerateAccessToken(claims.UserID, "")
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, err := s.tokenManager.GenerateRefreshToken(claims.UserID)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return accessToken, newRefreshToken, nil
}
