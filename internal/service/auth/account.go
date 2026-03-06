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
	"vngrocery/internal/service/audit"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailTaken         = errors.New("email is already registered")
)

type AccountService struct {
	authUsers      repository.AuthUserRepository
	users          repository.UserRepository
	keys           AccountKeyStore
	audit          AuditLogger
	googleTokens   GoogleTokenValidator
	jwt            Issuer
	jwtTTL         time.Duration
	googleClientID string
}

type AccountKey struct {
	PublicKey  string
	Algorithm  string
	VaultPath  string
	PrivateKey string
}

type AccountKeyStore interface {
	CreateAccountKey(ctx context.Context, userID string) (AccountKey, error)
	DeleteAccountKey(ctx context.Context, vaultPath string) error
}

type GoogleIdentity struct {
	Subject string
	Email   string
}

type GoogleTokenValidator interface {
	Validate(ctx context.Context, idToken, audience string) (GoogleIdentity, error)
}

type AuditLogger interface {
	Log(ctx context.Context, input audit.Input) error
}

type googleTokenValidator struct{}

func (googleTokenValidator) Validate(ctx context.Context, idToken, audience string) (GoogleIdentity, error) {
	payload, err := idtoken.Validate(ctx, idToken, audience)
	if err != nil {
		return GoogleIdentity{}, err
	}

	email, _ := payload.Claims["email"].(string)
	return GoogleIdentity{
		Subject: payload.Subject,
		Email:   email,
	}, nil
}

func NewAccountService(authUsers repository.AuthUserRepository, users repository.UserRepository, keys AccountKeyStore, auditLogger AuditLogger, googleTokens GoogleTokenValidator, jwt Issuer, jwtTTL time.Duration, googleClientID string) *AccountService {
	if googleTokens == nil {
		googleTokens = googleTokenValidator{}
	}
	return &AccountService{
		authUsers:      authUsers,
		users:          users,
		keys:           keys,
		audit:          auditLogger,
		googleTokens:   googleTokens,
		jwt:            jwt,
		jwtTTL:         jwtTTL,
		googleClientID: googleClientID,
	}
}

func (s *AccountService) Register(ctx context.Context, email, password, displayName string) (string, Principal, string, error) {
	emailLower := strings.ToLower(strings.TrimSpace(email))
	if emailLower == "" || password == "" {
		return "", Principal{}, "", fmt.Errorf("%w: email and password are required", ErrInvalidCredentials)
	}
	if len(password) < 8 {
		return "", Principal{}, "", fmt.Errorf("%w: password must be at least 8 characters", ErrInvalidCredentials)
	}

	if s.authUsers == nil || s.users == nil || s.keys == nil || s.jwt == nil {
		return "", Principal{}, "", fmt.Errorf("auth service is not configured")
	}

	existing, err := s.authUsers.GetByEmail(ctx, emailLower)
	if err == nil && existing.UserID != "" {
		return "", Principal{}, "", ErrEmailTaken
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", Principal{}, "", fmt.Errorf("failed to hash password: %w", err)
	}

	userID := s.authUsers.NewUserID()
	now := time.Now().UTC()
	key, err := s.keys.CreateAccountKey(ctx, userID)
	if err != nil {
		return "", Principal{}, "", fmt.Errorf("failed to create account key: %w", err)
	}

	authUser := domain.AuthUser{
		UserID:       userID,
		EmailLower:   emailLower,
		PasswordHash: string(passwordHash),
		Providers:    []string{"password"},
		PublicKey:    key.PublicKey,
		KeyAlgorithm: key.Algorithm,
		VaultKeyPath: key.VaultPath,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.authUsers.Save(ctx, authUser); err != nil {
		s.cleanupAccountKey(ctx, key.VaultPath)
		return "", Principal{}, "", err
	}

	if err := s.users.Save(ctx, domain.User{
		UserID:      userID,
		Email:       emailLower,
		DisplayName: strings.TrimSpace(displayName),
		Role:        "user",
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		s.cleanupAccountKey(ctx, key.VaultPath)
		return "", Principal{}, "", err
	}
	if err := s.logAccountCreated(ctx, authUser); err != nil {
		return "", Principal{}, "", err
	}

	principal := Principal{UserID: userID, Email: emailLower}
	token, err := s.jwt.IssueToken(principal, s.jwtTTL)
	if err != nil {
		return "", Principal{}, "", err
	}

	return token, principal, key.PublicKey, nil
}

func (s *AccountService) Login(ctx context.Context, email, password string) (string, Principal, string, error) {
	emailLower := strings.ToLower(strings.TrimSpace(email))
	if emailLower == "" || password == "" {
		return "", Principal{}, "", ErrInvalidCredentials
	}
	if s.authUsers == nil || s.jwt == nil {
		return "", Principal{}, "", fmt.Errorf("auth service is not configured")
	}

	authUser, err := s.authUsers.GetByEmail(ctx, emailLower)
	if err != nil || authUser.UserID == "" {
		return "", Principal{}, "", ErrInvalidCredentials
	}
	if authUser.PasswordHash == "" {
		return "", Principal{}, "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(authUser.PasswordHash), []byte(password)); err != nil {
		return "", Principal{}, "", ErrInvalidCredentials
	}

	principal := Principal{UserID: authUser.UserID, Email: emailLower}
	token, err := s.jwt.IssueToken(principal, s.jwtTTL)
	if err != nil {
		return "", Principal{}, "", err
	}

	return token, principal, authUser.PublicKey, nil
}

func (s *AccountService) GoogleLogin(ctx context.Context, googleIDToken string) (string, Principal, string, error) {
	if strings.TrimSpace(googleIDToken) == "" {
		return "", Principal{}, "", ErrInvalidCredentials
	}
	if s.googleClientID == "" {
		return "", Principal{}, "", fmt.Errorf("GOOGLE_CLIENT_ID is required for google login")
	}
	if s.authUsers == nil || s.users == nil || s.keys == nil || s.jwt == nil {
		return "", Principal{}, "", fmt.Errorf("auth service is not configured")
	}

	payload, err := s.googleTokens.Validate(ctx, googleIDToken, s.googleClientID)
	if err != nil {
		return "", Principal{}, "", ErrInvalidCredentials
	}

	emailLower := strings.ToLower(strings.TrimSpace(payload.Email))
	googleSub := strings.TrimSpace(payload.Subject)
	if googleSub == "" {
		return "", Principal{}, "", ErrInvalidCredentials
	}

	authUser, err := s.authUsers.GetByGoogleSub(ctx, googleSub)
	if err != nil || authUser.UserID == "" {
		userID := s.authUsers.NewUserID()
		now := time.Now().UTC()
		key, err := s.keys.CreateAccountKey(ctx, userID)
		if err != nil {
			return "", Principal{}, "", fmt.Errorf("failed to create account key: %w", err)
		}

		authUser = domain.AuthUser{
			UserID:       userID,
			EmailLower:   emailLower,
			GoogleSub:    googleSub,
			Providers:    []string{"google"},
			PublicKey:    key.PublicKey,
			KeyAlgorithm: key.Algorithm,
			VaultKeyPath: key.VaultPath,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := s.authUsers.Save(ctx, authUser); err != nil {
			s.cleanupAccountKey(ctx, key.VaultPath)
			return "", Principal{}, "", err
		}
		if err := s.users.Save(ctx, domain.User{
			UserID:      userID,
			Email:       emailLower,
			DisplayName: "",
			Role:        "user",
			CreatedAt:   now,
			UpdatedAt:   now,
		}); err != nil {
			s.cleanupAccountKey(ctx, key.VaultPath)
			return "", Principal{}, "", err
		}
		if err := s.logAccountCreated(ctx, authUser); err != nil {
			return "", Principal{}, "", err
		}
	}

	principal := Principal{UserID: authUser.UserID, Email: emailLower}
	token, err := s.jwt.IssueToken(principal, s.jwtTTL)
	if err != nil {
		return "", Principal{}, "", err
	}

	return token, principal, authUser.PublicKey, nil
}

func (s *AccountService) cleanupAccountKey(ctx context.Context, vaultPath string) {
	if s.keys == nil || strings.TrimSpace(vaultPath) == "" {
		return
	}
	_ = s.keys.DeleteAccountKey(ctx, vaultPath)
}

func (s *AccountService) logAccountCreated(ctx context.Context, authUser domain.AuthUser) error {
	if s.audit == nil {
		return nil
	}
	return s.audit.Log(ctx, audit.Input{
		ActorUserID:    authUser.UserID,
		ResourceType:   "account",
		ResourceID:     authUser.UserID,
		Action:         "account.created",
		PublicKey:      authUser.PublicKey,
		KeyAlgorithm:   authUser.KeyAlgorithm,
		SignerVaultKey: authUser.VaultKeyPath,
		Payload: map[string]any{
			"after": map[string]any{
				"userId":       authUser.UserID,
				"emailLower":   authUser.EmailLower,
				"providers":    authUser.Providers,
				"publicKey":    authUser.PublicKey,
				"keyAlgorithm": authUser.KeyAlgorithm,
			},
		},
	})
}
