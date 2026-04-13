package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/idtoken"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	"vngrocery/internal/service/audit"
)

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrEmailTaken          = errors.New("email is already registered")
	ErrAccountDeleted      = errors.New("account is deleted")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrInvalidResetToken   = errors.New("invalid password reset token")
	ErrVersionConflict     = errors.New("version conflict")
)

const (
	AccountStatusActive            = "active"
	AccountStatusDeleted           = "deleted"
	RefreshTokenStatusActive       = "active"
	RefreshTokenStatusRevoked      = "revoked"
	PasswordResetTokenStatusActive = "active"
	PasswordResetTokenStatusUsed   = "used"
)

type AccountService struct {
	authUsers        repository.AuthUserRepository
	users            repository.UserRepository
	refreshTokens    repository.RefreshTokenRepository
	passwordResets   repository.PasswordResetTokenRepository
	keys             AccountKeyStore
	audit            AuditLogger
	googleTokens     GoogleTokenValidator
	jwt              Issuer
	jwtTTL           time.Duration
	refreshTTL       time.Duration
	passwordResetTTL time.Duration
	googleClientID   string
	bootstrapAdmins  map[string]struct{}
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

type DeleteResult struct {
	UserID string
	Status string
}

type AuthResult struct {
	AccessToken  string
	RefreshToken string
	Principal    Principal
	PublicKey    string
}

type PasswordResetResult struct {
	ResetToken string
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

func NewAccountService(authUsers repository.AuthUserRepository, users repository.UserRepository, refreshTokens repository.RefreshTokenRepository, passwordResets repository.PasswordResetTokenRepository, keys AccountKeyStore, auditLogger AuditLogger, googleTokens GoogleTokenValidator, jwt Issuer, jwtTTL, refreshTTL time.Duration, googleClientID string, bootstrapAdminEmails ...string) *AccountService {
	if googleTokens == nil {
		googleTokens = googleTokenValidator{}
	}
	if refreshTTL <= 0 {
		refreshTTL = 30 * 24 * time.Hour
	}
	return &AccountService{
		authUsers:        authUsers,
		users:            users,
		refreshTokens:    refreshTokens,
		passwordResets:   passwordResets,
		keys:             keys,
		audit:            auditLogger,
		googleTokens:     googleTokens,
		jwt:              jwt,
		jwtTTL:           jwtTTL,
		refreshTTL:       refreshTTL,
		passwordResetTTL: time.Hour,
		googleClientID:   googleClientID,
		bootstrapAdmins:  normalizeBootstrapAdmins(bootstrapAdminEmails),
	}
}

func (s *AccountService) Register(ctx context.Context, email, password, displayName string) (AuthResult, error) {
	emailLower := strings.ToLower(strings.TrimSpace(email))
	if emailLower == "" || password == "" {
		return AuthResult{}, fmt.Errorf("%w: email and password are required", ErrInvalidCredentials)
	}
	if len(password) < 8 {
		return AuthResult{}, fmt.Errorf("%w: password must be at least 8 characters", ErrInvalidCredentials)
	}

	if s.authUsers == nil || s.users == nil || s.refreshTokens == nil || s.keys == nil || s.jwt == nil {
		return AuthResult{}, fmt.Errorf("auth service is not configured")
	}

	existing, err := s.authUsers.GetByEmail(ctx, emailLower)
	if err == nil && existing.UserID != "" {
		return AuthResult{}, ErrEmailTaken
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return AuthResult{}, fmt.Errorf("failed to hash password: %w", err)
	}

	userID := s.authUsers.NewUserID()
	now := time.Now().UTC()
	role := s.roleForNewUser(emailLower)
	key, err := s.keys.CreateAccountKey(ctx, userID)
	if err != nil {
		return AuthResult{}, fmt.Errorf("failed to create account key: %w", err)
	}

	authUser := domain.AuthUser{
		UserID:       userID,
		EmailLower:   emailLower,
		PasswordHash: string(passwordHash),
		Providers:    []string{"password"},
		Status:       AccountStatusActive,
		Version:      1,
		PublicKey:    key.PublicKey,
		KeyAlgorithm: key.Algorithm,
		VaultKeyPath: key.VaultPath,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.authUsers.Save(ctx, authUser); err != nil {
		s.cleanupAccountKey(ctx, key.VaultPath)
		return AuthResult{}, err
	}

	if err := s.users.Save(ctx, domain.User{
		UserID:      userID,
		Email:       emailLower,
		DisplayName: strings.TrimSpace(displayName),
		Role:        role,
		Status:      AccountStatusActive,
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		s.cleanupAccountKey(ctx, key.VaultPath)
		return AuthResult{}, err
	}
	if err := s.logAccountCreated(ctx, authUser); err != nil {
		return AuthResult{}, err
	}

	principal := Principal{UserID: userID, Email: emailLower, Role: role}
	return s.issueAuthResult(ctx, principal, key.PublicKey)
}

func (s *AccountService) Login(ctx context.Context, email, password string) (AuthResult, error) {
	emailLower := strings.ToLower(strings.TrimSpace(email))
	if emailLower == "" || password == "" {
		return AuthResult{}, ErrInvalidCredentials
	}
	if s.authUsers == nil || s.users == nil || s.refreshTokens == nil || s.jwt == nil {
		return AuthResult{}, fmt.Errorf("auth service is not configured")
	}

	authUser, err := s.authUsers.GetByEmail(ctx, emailLower)
	if err != nil || authUser.UserID == "" {
		return AuthResult{}, ErrInvalidCredentials
	}
	if authUser.Status != AccountStatusActive {
		return AuthResult{}, ErrAccountDeleted
	}
	if authUser.PasswordHash == "" {
		return AuthResult{}, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(authUser.PasswordHash), []byte(password)); err != nil {
		return AuthResult{}, ErrInvalidCredentials
	}

	user, err := s.users.GetByID(ctx, authUser.UserID)
	if err != nil || user.UserID == "" {
		return AuthResult{}, ErrInvalidCredentials
	}
	if user.Status != AccountStatusActive {
		return AuthResult{}, ErrAccountDeleted
	}

	principal := Principal{UserID: authUser.UserID, Email: emailLower, Role: user.Role}
	return s.issueAuthResult(ctx, principal, authUser.PublicKey)
}

func (s *AccountService) GoogleLogin(ctx context.Context, googleIDToken string) (AuthResult, error) {
	if strings.TrimSpace(googleIDToken) == "" {
		return AuthResult{}, ErrInvalidCredentials
	}
	if s.googleClientID == "" {
		return AuthResult{}, fmt.Errorf("GOOGLE_CLIENT_ID is required for google login")
	}
	if s.authUsers == nil || s.users == nil || s.refreshTokens == nil || s.keys == nil || s.jwt == nil {
		return AuthResult{}, fmt.Errorf("auth service is not configured")
	}

	payload, err := s.googleTokens.Validate(ctx, googleIDToken, s.googleClientID)
	if err != nil {
		return AuthResult{}, ErrInvalidCredentials
	}

	emailLower := strings.ToLower(strings.TrimSpace(payload.Email))
	googleSub := strings.TrimSpace(payload.Subject)
	if googleSub == "" {
		return AuthResult{}, ErrInvalidCredentials
	}

	authUser, err := s.authUsers.GetByGoogleSub(ctx, googleSub)
	if err != nil || authUser.UserID == "" {
		userID := s.authUsers.NewUserID()
		now := time.Now().UTC()
		role := s.roleForNewUser(emailLower)
		key, err := s.keys.CreateAccountKey(ctx, userID)
		if err != nil {
			return AuthResult{}, fmt.Errorf("failed to create account key: %w", err)
		}

		authUser = domain.AuthUser{
			UserID:       userID,
			EmailLower:   emailLower,
			GoogleSub:    googleSub,
			Providers:    []string{"google"},
			Status:       AccountStatusActive,
			Version:      1,
			PublicKey:    key.PublicKey,
			KeyAlgorithm: key.Algorithm,
			VaultKeyPath: key.VaultPath,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := s.authUsers.Save(ctx, authUser); err != nil {
			s.cleanupAccountKey(ctx, key.VaultPath)
			return AuthResult{}, err
		}
		if err := s.users.Save(ctx, domain.User{
			UserID:      userID,
			Email:       emailLower,
			DisplayName: "",
			Role:        role,
			Status:      AccountStatusActive,
			Version:     1,
			CreatedAt:   now,
			UpdatedAt:   now,
		}); err != nil {
			s.cleanupAccountKey(ctx, key.VaultPath)
			return AuthResult{}, err
		}
		if err := s.logAccountCreated(ctx, authUser); err != nil {
			return AuthResult{}, err
		}
	}
	if authUser.Status != AccountStatusActive {
		return AuthResult{}, ErrAccountDeleted
	}

	user, err := s.users.GetByID(ctx, authUser.UserID)
	if err != nil || user.UserID == "" {
		return AuthResult{}, ErrInvalidCredentials
	}
	if user.Status != AccountStatusActive {
		return AuthResult{}, ErrAccountDeleted
	}

	principal := Principal{UserID: authUser.UserID, Email: emailLower, Role: user.Role}
	return s.issueAuthResult(ctx, principal, authUser.PublicKey)
}

func normalizeBootstrapAdmins(emails []string) map[string]struct{} {
	if len(emails) == 0 {
		return nil
	}
	result := make(map[string]struct{}, len(emails))
	for _, email := range emails {
		normalized := strings.ToLower(strings.TrimSpace(email))
		if normalized == "" {
			continue
		}
		result[normalized] = struct{}{}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func (s *AccountService) roleForNewUser(email string) string {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized != "" && s.bootstrapAdmins != nil {
		if _, ok := s.bootstrapAdmins[normalized]; ok {
			return "admin"
		}
	}
	return "user"
}

func (s *AccountService) Refresh(ctx context.Context, refreshToken string) (AuthResult, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return AuthResult{}, ErrInvalidRefreshToken
	}
	if s.authUsers == nil || s.users == nil || s.refreshTokens == nil || s.jwt == nil {
		return AuthResult{}, fmt.Errorf("auth service is not configured")
	}

	stored, err := s.refreshTokens.GetByTokenHash(ctx, hashRefreshToken(refreshToken))
	if err != nil || stored.TokenID == "" {
		return AuthResult{}, ErrInvalidRefreshToken
	}
	if stored.Status != RefreshTokenStatusActive || time.Now().UTC().After(stored.ExpiresAt) {
		return AuthResult{}, ErrInvalidRefreshToken
	}
	authUser, err := s.authUsers.GetByID(ctx, stored.UserID)
	if err != nil || authUser.UserID == "" {
		return AuthResult{}, ErrInvalidRefreshToken
	}
	if authUser.Status != AccountStatusActive {
		return AuthResult{}, ErrAccountDeleted
	}
	user, err := s.users.GetByID(ctx, stored.UserID)
	if err != nil || user.UserID == "" {
		return AuthResult{}, ErrInvalidRefreshToken
	}
	if user.Status != AccountStatusActive {
		return AuthResult{}, ErrAccountDeleted
	}

	return s.issueAuthResult(ctx, Principal{UserID: authUser.UserID, Email: authUser.EmailLower, Role: user.Role}, authUser.PublicKey)
}

func (s *AccountService) Logout(ctx context.Context, refreshToken string) error {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return ErrInvalidRefreshToken
	}
	if s.refreshTokens == nil {
		return fmt.Errorf("auth service is not configured")
	}

	stored, err := s.refreshTokens.GetByTokenHash(ctx, hashRefreshToken(refreshToken))
	if err != nil || stored.TokenID == "" {
		return ErrInvalidRefreshToken
	}
	if stored.Status == RefreshTokenStatusRevoked {
		return nil
	}

	now := time.Now().UTC()
	stored.Status = RefreshTokenStatusRevoked
	stored.RevokedAt = &now
	stored.UpdatedAt = now
	return s.refreshTokens.Save(ctx, stored)
}

func (s *AccountService) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" || currentPassword == "" || newPassword == "" {
		return fmt.Errorf("%w: userId, currentPassword, and newPassword are required", ErrInvalidCredentials)
	}
	if len(newPassword) < 8 {
		return fmt.Errorf("%w: newPassword must be at least 8 characters", ErrInvalidCredentials)
	}
	if s.authUsers == nil {
		return fmt.Errorf("auth service is not configured")
	}

	authUser, err := s.authUsers.GetByID(ctx, userID)
	if err != nil || authUser.UserID == "" {
		return ErrInvalidCredentials
	}
	if authUser.Status != AccountStatusActive {
		return ErrAccountDeleted
	}
	if authUser.PasswordHash == "" {
		return ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(authUser.PasswordHash), []byte(currentPassword)); err != nil {
		return ErrInvalidCredentials
	}

	return s.updatePassword(ctx, authUser, newPassword, "account.password_changed")
}

func (s *AccountService) RequestPasswordReset(ctx context.Context, email string) (PasswordResetResult, error) {
	emailLower := strings.ToLower(strings.TrimSpace(email))
	if emailLower == "" {
		return PasswordResetResult{}, fmt.Errorf("%w: email is required", ErrInvalidCredentials)
	}
	if s.authUsers == nil || s.passwordResets == nil {
		return PasswordResetResult{}, fmt.Errorf("auth service is not configured")
	}

	authUser, err := s.authUsers.GetByEmail(ctx, emailLower)
	if err != nil || authUser.UserID == "" || authUser.Status != AccountStatusActive {
		return PasswordResetResult{}, ErrInvalidCredentials
	}

	resetToken, err := newRefreshToken()
	if err != nil {
		return PasswordResetResult{}, err
	}
	now := time.Now().UTC()
	if err := s.passwordResets.Save(ctx, domain.PasswordResetToken{
		TokenID:   uuid.NewString(),
		UserID:    authUser.UserID,
		TokenHash: hashSecret(resetToken),
		Status:    PasswordResetTokenStatusActive,
		ExpiresAt: now.Add(s.passwordResetTTL),
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return PasswordResetResult{}, err
	}

	return PasswordResetResult{ResetToken: resetToken}, nil
}

func (s *AccountService) ResetPassword(ctx context.Context, resetToken, newPassword string) error {
	resetToken = strings.TrimSpace(resetToken)
	if resetToken == "" || newPassword == "" {
		return fmt.Errorf("%w: resetToken and newPassword are required", ErrInvalidResetToken)
	}
	if len(newPassword) < 8 {
		return fmt.Errorf("%w: newPassword must be at least 8 characters", ErrInvalidCredentials)
	}
	if s.authUsers == nil || s.passwordResets == nil {
		return fmt.Errorf("auth service is not configured")
	}

	stored, err := s.passwordResets.GetByTokenHash(ctx, hashSecret(resetToken))
	if err != nil || stored.TokenID == "" {
		return ErrInvalidResetToken
	}
	if stored.Status != PasswordResetTokenStatusActive || time.Now().UTC().After(stored.ExpiresAt) {
		return ErrInvalidResetToken
	}
	authUser, err := s.authUsers.GetByID(ctx, stored.UserID)
	if err != nil || authUser.UserID == "" {
		return ErrInvalidResetToken
	}
	if authUser.Status != AccountStatusActive {
		return ErrAccountDeleted
	}

	if err := s.updatePassword(ctx, authUser, newPassword, "account.password_reset"); err != nil {
		return err
	}

	now := time.Now().UTC()
	stored.Status = PasswordResetTokenStatusUsed
	stored.UsedAt = &now
	stored.UpdatedAt = now
	return s.passwordResets.Save(ctx, stored)
}

func (s *AccountService) issueAuthResult(ctx context.Context, principal Principal, publicKey string) (AuthResult, error) {
	accessToken, err := s.jwt.IssueToken(principal, s.jwtTTL)
	if err != nil {
		return AuthResult{}, err
	}

	refreshToken, err := newRefreshToken()
	if err != nil {
		return AuthResult{}, err
	}
	now := time.Now().UTC()
	if err := s.refreshTokens.Save(ctx, domain.RefreshToken{
		TokenID:   uuid.NewString(),
		UserID:    principal.UserID,
		TokenHash: hashSecret(refreshToken),
		Status:    RefreshTokenStatusActive,
		ExpiresAt: now.Add(s.refreshTTL),
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Principal:    principal,
		PublicKey:    publicKey,
	}, nil
}

func (s *AccountService) Delete(ctx context.Context, userID string, expectedVersion int) (DeleteResult, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return DeleteResult{}, fmt.Errorf("%w: userId is required", ErrInvalidCredentials)
	}
	if expectedVersion <= 0 {
		return DeleteResult{}, fmt.Errorf("%w: expectedVersion must be positive", ErrInvalidCredentials)
	}
	if s.authUsers == nil || s.users == nil {
		return DeleteResult{}, fmt.Errorf("auth service is not configured")
	}

	authUser, err := s.authUsers.GetByID(ctx, userID)
	if err != nil {
		return DeleteResult{}, err
	}
	if authUser.Version != expectedVersion {
		return DeleteResult{}, ErrVersionConflict
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return DeleteResult{}, err
	}
	if authUser.Status == AccountStatusDeleted && user.Status == AccountStatusDeleted {
		return DeleteResult{UserID: userID, Status: AccountStatusDeleted}, nil
	}

	now := time.Now().UTC()
	authUser.Status = AccountStatusDeleted
	authUser.Version++
	authUser.UpdatedAt = now
	if err := s.authUsers.Save(ctx, authUser); err != nil {
		return DeleteResult{}, err
	}

	user.Status = AccountStatusDeleted
	user.Version++
	user.UpdatedAt = now
	if err := s.users.Save(ctx, user); err != nil {
		return DeleteResult{}, err
	}

	if s.audit != nil {
		if err := s.audit.Log(ctx, audit.Input{
			ActorUserID:     userID,
			ResourceType:    "account",
			ResourceID:      userID,
			ResourceVersion: authUser.Version,
			Action:          "account.deleted",
			Status:          "deleted",
			PublicKey:       authUser.PublicKey,
			KeyAlgorithm:    authUser.KeyAlgorithm,
			SignerVaultKey:  authUser.VaultKeyPath,
			Payload: audit.MutationPayload{
				After: map[string]any{
					"userId":  userID,
					"status":  AccountStatusDeleted,
					"version": authUser.Version,
				},
			},
		}); err != nil {
			return DeleteResult{}, err
		}
	}

	return DeleteResult{UserID: userID, Status: AccountStatusDeleted}, nil
}

func (s *AccountService) cleanupAccountKey(ctx context.Context, vaultPath string) {
	if s.keys == nil || strings.TrimSpace(vaultPath) == "" {
		return
	}
	_ = s.keys.DeleteAccountKey(ctx, vaultPath)
}

func newRefreshToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func hashRefreshToken(token string) string {
	return hashSecret(token)
}

func hashSecret(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func (s *AccountService) updatePassword(ctx context.Context, authUser domain.AuthUser, newPassword, action string) error {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	authUser.PasswordHash = string(passwordHash)
	authUser.Version++
	authUser.UpdatedAt = time.Now().UTC()
	if err := s.authUsers.Save(ctx, authUser); err != nil {
		return err
	}
	if s.audit == nil {
		return nil
	}
	return s.audit.Log(ctx, audit.Input{
		ActorUserID:     authUser.UserID,
		ResourceType:    "account",
		ResourceID:      authUser.UserID,
		ResourceVersion: authUser.Version,
		Action:          action,
		Status:          "updated",
		PublicKey:       authUser.PublicKey,
		KeyAlgorithm:    authUser.KeyAlgorithm,
		SignerVaultKey:  authUser.VaultKeyPath,
		Payload: audit.MutationPayload{
			After: map[string]any{
				"userId":  authUser.UserID,
				"status":  authUser.Status,
				"version": authUser.Version,
			},
		},
	})
}

func (s *AccountService) logAccountCreated(ctx context.Context, authUser domain.AuthUser) error {
	if s.audit == nil {
		return nil
	}
	return s.audit.Log(ctx, audit.Input{
		ActorUserID:     authUser.UserID,
		ResourceType:    "account",
		ResourceID:      authUser.UserID,
		ResourceVersion: authUser.Version,
		Action:          "account.created",
		Status:          "created",
		PublicKey:       authUser.PublicKey,
		KeyAlgorithm:    authUser.KeyAlgorithm,
		SignerVaultKey:  authUser.VaultKeyPath,
		Payload: audit.MutationPayload{
			After: map[string]any{
				"userId":       authUser.UserID,
				"emailLower":   authUser.EmailLower,
				"providers":    authUser.Providers,
				"status":       authUser.Status,
				"version":      authUser.Version,
				"publicKey":    authUser.PublicKey,
				"keyAlgorithm": authUser.KeyAlgorithm,
			},
		},
	})
}
