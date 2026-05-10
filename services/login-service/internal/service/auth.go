package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"ride-sharing/services/login-service/internal/domain"
	"ride-sharing/services/login-service/pkg/token"
	"ride-sharing/shared/env"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

const defaultRole = "rider"

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
		Role:         defaultRole,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, "", "", fmt.Errorf("failed to create user: %w", err)
	}

	accessToken, err := s.tokenManager.GenerateAccessToken(user.ID, user.Email, user.Role)
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

	accessToken, err := s.tokenManager.GenerateAccessToken(user.ID, user.Email, user.Role)
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

	user, err := s.repo.GetByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", ErrInvalidToken
		}
		return "", "", fmt.Errorf("failed to get user for refresh: %w", err)
	}

	accessToken, err := s.tokenManager.GenerateAccessToken(user.ID, user.Email, user.Role)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, err := s.tokenManager.GenerateRefreshToken(claims.UserID)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return accessToken, newRefreshToken, nil
}

func (s *AuthService) GoogleAuth(ctx context.Context, googleIDToken string) (*domain.User, string, string, error) {
	audience := env.GetString("GOOGLE_CLIENT_ID", "")
	if audience == "" {
		return nil, "", "", fmt.Errorf("GOOGLE_CLIENT_ID is required")
	}

	googleClaims, err := validateGoogleIDToken(ctx, googleIDToken)
	if err != nil {
		return nil, "", "", ErrInvalidCredentials
	}
	if googleClaims.Aud != audience {
		return nil, "", "", ErrInvalidCredentials
	}

	email := strings.ToLower(strings.TrimSpace(googleClaims.Email))
	googleSub := strings.TrimSpace(googleClaims.Sub) // unique identifier for the user from Google, used to link the user in our database to their Google account
	name := strings.TrimSpace(googleClaims.Name)
	avatarURL := strings.TrimSpace(googleClaims.Picture)
	emailVerified := googleClaims.EmailVerified

	if email == "" || googleSub == "" || !emailVerified {
		return nil, "", "", ErrInvalidCredentials
	}

	user, err := s.repo.GetByGoogleSub(ctx, googleSub)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, "", "", fmt.Errorf("failed to fetch user by google sub: %w", err)
		}

		user, err = s.repo.GetByEmail(ctx, email) // if not found by google sub, try to find by email to link accounts
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return nil, "", "", fmt.Errorf("failed to fetch user by email: %w", err)
			}

			if name == "" {
				name = strings.Split(email, "@")[0]
			}

			user = &domain.User{
				Email:        email,
				PasswordHash: "",
				Name:         name,
				Role:         defaultRole,
			}

			// insert into users table and link to google identity in user_identities table in a transaction to ensure consistency
			if err := s.repo.Create(ctx, user); err != nil {
				return nil, "", "", fmt.Errorf("failed to create google user: %w", err)
			}

			// insert into user_identities table to link the new user to their google subject
			if err := s.repo.LinkGoogleIdentity(ctx, user.ID, googleSub, avatarURL, emailVerified); err != nil {
				return nil, "", "", fmt.Errorf("failed to create google identity link: %w", err)
			}
			// reload the user after linking to get the google_sub populated
			user, err = s.repo.GetByID(ctx, user.ID)
			if err != nil {
				return nil, "", "", fmt.Errorf("failed to reload created google user: %w", err)
			}
		} else {
			// user exists with the same email but doesn't have a google_sub linked yet, so link it now
			if err := s.repo.LinkGoogleIdentity(ctx, user.ID, googleSub, avatarURL, emailVerified); err != nil {
				return nil, "", "", fmt.Errorf("failed to link google identity: %w", err)
			}
			// reload the user after linking to get the google_sub populated
			user, err = s.repo.GetByID(ctx, user.ID)
			if err != nil {
				return nil, "", "", fmt.Errorf("failed to reload linked user: %w", err)
			}
		}
	}
	// generate tokens for the user (whether newly created or existing) and return
	accessToken, err := s.tokenManager.GenerateAccessToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.tokenManager.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return user, accessToken, refreshToken, nil
}

type googleTokenInfoResponse struct {
	Aud           string `json:"aud"`
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

type googleClaims struct {
	Aud           string
	Sub           string
	Email         string
	EmailVerified bool
	Name          string
	Picture       string
}

func validateGoogleIDToken(ctx context.Context, idToken string) (*googleClaims, error) {
	endpoint := "https://oauth2.googleapis.com/tokeninfo?id_token=" + url.QueryEscape(idToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google token validation failed with status %d", resp.StatusCode)
	}

	var tokenInfo googleTokenInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenInfo); err != nil {
		return nil, err
	}

	claims := &googleClaims{
		Aud:           tokenInfo.Aud, // should match our GOOGLE_CLIENT_ID
		Sub:           tokenInfo.Sub, // unique identifier for the user from Google
		Email:         tokenInfo.Email,
		EmailVerified: strings.EqualFold(tokenInfo.EmailVerified, "true"), // whether the user's email is verified by Google
		Name:          tokenInfo.Name,                                     // user's full name from Google
		Picture:       tokenInfo.Picture,                                  // user's profile picture URL from Google
	}

	return claims, nil
}
