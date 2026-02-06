package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/idtoken"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailTaken         = errors.New("email is already registered")
)

type AccountService struct {
	authUsers      repository.AuthUserRepository
	users          repository.UserRepository
	jwt            Issuer
	jwtTTL         time.Duration
	googleClientID string
}

func NewAccountService(authUsers repository.AuthUserRepository, users repository.UserRepository, jwt Issuer, jwtTTL time.Duration, googleClientID string) *AccountService {
	return &AccountService{
		authUsers:      authUsers,
		users:          users,
		jwt:            jwt,
		jwtTTL:         jwtTTL,
		googleClientID: googleClientID,
	}
}

func (s *AccountService) Register(ctx context.Context, email, password, displayName string) (string, Principal, error) {
	emailLower := strings.ToLower(strings.TrimSpace(email))
	if emailLower == "" || password == "" {
		return "", Principal{}, fmt.Errorf("%w: email and password are required", ErrInvalidCredentials)
	}
	if len(password) < 8 {
		return "", Principal{}, fmt.Errorf("%w: password must be at least 8 characters", ErrInvalidCredentials)
	}

	if s.authUsers == nil || s.users == nil || s.jwt == nil {
		return "", Principal{}, fmt.Errorf("auth service is not configured")
	}

	existing, err := s.authUsers.GetByEmail(ctx, emailLower)
	if err == nil && existing.UserID != "" {
		return "", Principal{}, ErrEmailTaken
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", Principal{}, fmt.Errorf("failed to hash password: %w", err)
	}

	userID := s.authUsers.NewUserID()
	now := time.Now().UTC()

	authUser := domain.AuthUser{
		UserID:       userID,
		EmailLower:   emailLower,
		PasswordHash: string(passwordHash),
		Providers:    []string{"password"},
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.authUsers.Save(ctx, authUser); err != nil {
		return "", Principal{}, err
	}

	if err := s.users.Save(ctx, domain.User{
		UserID:      userID,
		Email:       emailLower,
		DisplayName: strings.TrimSpace(displayName),
		Role:        "user",
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		return "", Principal{}, err
	}

	principal := Principal{UserID: userID, Email: emailLower}
	token, err := s.jwt.IssueToken(principal, s.jwtTTL)
	if err != nil {
		return "", Principal{}, err
	}

	return token, principal, nil
}

func (s *AccountService) Login(ctx context.Context, email, password string) (string, Principal, error) {
	emailLower := strings.ToLower(strings.TrimSpace(email))
	if emailLower == "" || password == "" {
		return "", Principal{}, ErrInvalidCredentials
	}
	if s.authUsers == nil || s.jwt == nil {
		return "", Principal{}, fmt.Errorf("auth service is not configured")
	}

	authUser, err := s.authUsers.GetByEmail(ctx, emailLower)
	if err != nil || authUser.UserID == "" {
		return "", Principal{}, ErrInvalidCredentials
	}
	if authUser.PasswordHash == "" {
		return "", Principal{}, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(authUser.PasswordHash), []byte(password)); err != nil {
		return "", Principal{}, ErrInvalidCredentials
	}

	principal := Principal{UserID: authUser.UserID, Email: emailLower}
	token, err := s.jwt.IssueToken(principal, s.jwtTTL)
	if err != nil {
		return "", Principal{}, err
	}

	return token, principal, nil
}

func (s *AccountService) GoogleLogin(ctx context.Context, googleIDToken string) (string, Principal, error) {
	if strings.TrimSpace(googleIDToken) == "" {
		return "", Principal{}, ErrInvalidCredentials
	}
	if s.googleClientID == "" {
		return "", Principal{}, fmt.Errorf("GOOGLE_CLIENT_ID is required for google login")
	}
	if s.authUsers == nil || s.users == nil || s.jwt == nil {
		return "", Principal{}, fmt.Errorf("auth service is not configured")
	}

	payload, err := idtoken.Validate(ctx, googleIDToken, s.googleClientID)
	if err != nil {
		return "", Principal{}, ErrInvalidCredentials
	}

	email, _ := payload.Claims["email"].(string)
	emailLower := strings.ToLower(strings.TrimSpace(email))
	googleSub := strings.TrimSpace(payload.Subject)
	if googleSub == "" {
		return "", Principal{}, ErrInvalidCredentials
	}

	authUser, err := s.authUsers.GetByGoogleSub(ctx, googleSub)
	if err != nil || authUser.UserID == "" {
		userID := s.authUsers.NewUserID()
		now := time.Now().UTC()

		authUser = domain.AuthUser{
			UserID:     userID,
			EmailLower: emailLower,
			GoogleSub:  googleSub,
			Providers:  []string{"google"},
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if err := s.authUsers.Save(ctx, authUser); err != nil {
			return "", Principal{}, err
		}
		if err := s.users.Save(ctx, domain.User{
			UserID:      userID,
			Email:       emailLower,
			DisplayName: "",
			Role:        "user",
			CreatedAt:   now,
			UpdatedAt:   now,
		}); err != nil {
			return "", Principal{}, err
		}
	}

	principal := Principal{UserID: authUser.UserID, Email: emailLower}
	token, err := s.jwt.IssueToken(principal, s.jwtTTL)
	if err != nil {
		return "", Principal{}, err
	}

	return token, principal, nil
}
