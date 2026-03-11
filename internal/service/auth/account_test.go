package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"vngrocery/internal/domain"
	"vngrocery/internal/service/audit"
)

type authUserRepoStub struct {
	newUserID      string
	getByID        func(ctx context.Context, userID string) (domain.AuthUser, error)
	getByEmail     func(ctx context.Context, emailLower string) (domain.AuthUser, error)
	getByGoogleSub func(ctx context.Context, googleSub string) (domain.AuthUser, error)
	save           func(ctx context.Context, user domain.AuthUser) error
}

func (s authUserRepoStub) NewUserID() string { return s.newUserID }
func (s authUserRepoStub) Save(ctx context.Context, user domain.AuthUser) error {
	if s.save != nil {
		return s.save(ctx, user)
	}
	return nil
}
func (s authUserRepoStub) GetByID(ctx context.Context, userID string) (domain.AuthUser, error) {
	if s.getByID != nil {
		return s.getByID(ctx, userID)
	}
	return domain.AuthUser{}, errors.New("not found")
}
func (s authUserRepoStub) GetByEmail(ctx context.Context, emailLower string) (domain.AuthUser, error) {
	if s.getByEmail != nil {
		return s.getByEmail(ctx, emailLower)
	}
	return domain.AuthUser{}, errors.New("not found")
}
func (s authUserRepoStub) GetByGoogleSub(ctx context.Context, googleSub string) (domain.AuthUser, error) {
	if s.getByGoogleSub != nil {
		return s.getByGoogleSub(ctx, googleSub)
	}
	return domain.AuthUser{}, errors.New("not found")
}

type userRepoStub struct {
	save    func(ctx context.Context, user domain.User) error
	getByID func(ctx context.Context, userID string) (domain.User, error)
}

func (s userRepoStub) Save(ctx context.Context, user domain.User) error {
	if s.save != nil {
		return s.save(ctx, user)
	}
	return nil
}

func (s userRepoStub) GetByID(ctx context.Context, userID string) (domain.User, error) {
	if s.getByID != nil {
		return s.getByID(ctx, userID)
	}
	return domain.User{}, errors.New("not implemented")
}

type refreshTokenRepoStub struct {
	save           func(ctx context.Context, token domain.RefreshToken) error
	getByTokenHash func(ctx context.Context, tokenHash string) (domain.RefreshToken, error)
	saveHits       int
}

func (s *refreshTokenRepoStub) Save(ctx context.Context, token domain.RefreshToken) error {
	s.saveHits++
	if s.save != nil {
		return s.save(ctx, token)
	}
	return nil
}

func (s *refreshTokenRepoStub) GetByTokenHash(ctx context.Context, tokenHash string) (domain.RefreshToken, error) {
	if s.getByTokenHash != nil {
		return s.getByTokenHash(ctx, tokenHash)
	}
	return domain.RefreshToken{}, errors.New("not found")
}

type passwordResetTokenRepoStub struct {
	save           func(ctx context.Context, token domain.PasswordResetToken) error
	getByTokenHash func(ctx context.Context, tokenHash string) (domain.PasswordResetToken, error)
	saveHits       int
}

func (s *passwordResetTokenRepoStub) Save(ctx context.Context, token domain.PasswordResetToken) error {
	s.saveHits++
	if s.save != nil {
		return s.save(ctx, token)
	}
	return nil
}

func (s *passwordResetTokenRepoStub) GetByTokenHash(ctx context.Context, tokenHash string) (domain.PasswordResetToken, error) {
	if s.getByTokenHash != nil {
		return s.getByTokenHash(ctx, tokenHash)
	}
	return domain.PasswordResetToken{}, errors.New("not found")
}

type accountKeyStoreStub struct {
	create     func(ctx context.Context, userID string) (AccountKey, error)
	delete     func(ctx context.Context, vaultPath string) error
	createHits int
	deleteHits int
}

func (s *accountKeyStoreStub) CreateAccountKey(ctx context.Context, userID string) (AccountKey, error) {
	s.createHits++
	if s.create != nil {
		return s.create(ctx, userID)
	}
	return AccountKey{}, nil
}

func (s *accountKeyStoreStub) DeleteAccountKey(ctx context.Context, vaultPath string) error {
	s.deleteHits++
	if s.delete != nil {
		return s.delete(ctx, vaultPath)
	}
	return nil
}

type issuerStub struct{}

func (issuerStub) IssueToken(principal Principal, ttl time.Duration) (string, error) {
	return "token-for-" + principal.UserID, nil
}

func (issuerStub) Verify(ctx context.Context, token string) (Principal, error) {
	return Principal{}, nil
}

type googleTokensStub struct {
	validate func(ctx context.Context, idToken, audience string) (GoogleIdentity, error)
}

func (s googleTokensStub) Validate(ctx context.Context, idToken, audience string) (GoogleIdentity, error) {
	return s.validate(ctx, idToken, audience)
}

type auditLoggerStub struct {
	log     func(ctx context.Context, input auditInput) error
	logHits int
}

type auditInput struct {
	Action       string
	ActorUserID  string
	ResourceType string
	ResourceID   string
}

func (s *auditLoggerStub) Log(ctx context.Context, input audit.Input) error {
	s.logHits++
	if s.log != nil {
		return s.log(ctx, auditInput{
			Action:       input.Action,
			ActorUserID:  input.ActorUserID,
			ResourceType: input.ResourceType,
			ResourceID:   input.ResourceID,
		})
	}
	return nil
}

func TestAccountServiceRegisterCreatesVaultKeyAndReturnsPublicKey(t *testing.T) {
	keys := &accountKeyStoreStub{
		create: func(ctx context.Context, userID string) (AccountKey, error) {
			if userID != "user-1" {
				t.Fatalf("unexpected userID: %s", userID)
			}
			return AccountKey{
				PublicKey: "pub-key",
				Algorithm: "Ed25519",
				VaultPath: "account-keys/user-1",
			}, nil
		},
	}

	var savedAuthUser domain.AuthUser
	var savedUser domain.User
	service := NewAccountService(
		authUserRepoStub{
			newUserID: "user-1",
			save: func(ctx context.Context, user domain.AuthUser) error {
				savedAuthUser = user
				return nil
			},
		},
		userRepoStub{
			save: func(ctx context.Context, user domain.User) error {
				savedUser = user
				return nil
			},
		},
		&refreshTokenRepoStub{},
		&passwordResetTokenRepoStub{},
		keys,
		nil,
		nil,
		issuerStub{},
		time.Hour,
		30*24*time.Hour,
		"google-client-id",
	)

	result, err := service.Register(context.Background(), "USER@example.com", "password123", "Demo User")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.AccessToken != "token-for-user-1" || result.RefreshToken == "" {
		t.Fatalf("unexpected tokens: %#v", result)
	}
	if result.Principal.UserID != "user-1" {
		t.Fatalf("unexpected principal: %#v", result.Principal)
	}
	if result.PublicKey != "pub-key" {
		t.Fatalf("unexpected public key: %s", result.PublicKey)
	}
	if savedAuthUser.PublicKey != "pub-key" || savedAuthUser.KeyAlgorithm != "Ed25519" || savedAuthUser.VaultKeyPath != "account-keys/user-1" {
		t.Fatalf("auth user key metadata not persisted: %#v", savedAuthUser)
	}
	if savedAuthUser.Status != AccountStatusActive {
		t.Fatalf("unexpected auth user status: %s", savedAuthUser.Status)
	}
	if savedAuthUser.Version != 1 {
		t.Fatalf("unexpected auth user version: %d", savedAuthUser.Version)
	}
	if savedUser.Email != "user@example.com" {
		t.Fatalf("unexpected saved user email: %s", savedUser.Email)
	}
	if savedUser.Status != AccountStatusActive {
		t.Fatalf("unexpected saved user status: %s", savedUser.Status)
	}
	if savedUser.Version != 1 {
		t.Fatalf("unexpected saved user version: %d", savedUser.Version)
	}
	if keys.createHits != 1 || keys.deleteHits != 0 {
		t.Fatalf("unexpected key store calls: create=%d delete=%d", keys.createHits, keys.deleteHits)
	}
}

func TestAccountServiceRegisterDoesNotCreateKeyWhenEmailExists(t *testing.T) {
	keys := &accountKeyStoreStub{}
	service := NewAccountService(
		authUserRepoStub{
			newUserID: "user-1",
			getByEmail: func(ctx context.Context, emailLower string) (domain.AuthUser, error) {
				return domain.AuthUser{UserID: "existing-user"}, nil
			},
		},
		userRepoStub{},
		&refreshTokenRepoStub{},
		&passwordResetTokenRepoStub{},
		keys,
		nil,
		nil,
		issuerStub{},
		time.Hour,
		30*24*time.Hour,
		"google-client-id",
	)

	_, err := service.Register(context.Background(), "user@example.com", "password123", "Demo User")
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}
	if keys.createHits != 0 {
		t.Fatalf("expected no key generation, got %d", keys.createHits)
	}
}

func TestAccountServiceRegisterCleansVaultKeyWhenUserSaveFails(t *testing.T) {
	keys := &accountKeyStoreStub{
		create: func(ctx context.Context, userID string) (AccountKey, error) {
			return AccountKey{
				PublicKey: "pub-key",
				Algorithm: "Ed25519",
				VaultPath: "account-keys/user-1",
			}, nil
		},
	}
	service := NewAccountService(
		authUserRepoStub{newUserID: "user-1"},
		userRepoStub{
			save: func(ctx context.Context, user domain.User) error {
				return errors.New("user save failed")
			},
		},
		&refreshTokenRepoStub{},
		&passwordResetTokenRepoStub{},
		keys,
		nil,
		nil,
		issuerStub{},
		time.Hour,
		30*24*time.Hour,
		"google-client-id",
	)

	_, err := service.Register(context.Background(), "user@example.com", "password123", "Demo User")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if keys.deleteHits != 1 {
		t.Fatalf("expected cleanup delete call, got %d", keys.deleteHits)
	}
}

func TestAccountServiceRegisterWritesAuditLog(t *testing.T) {
	keys := &accountKeyStoreStub{
		create: func(ctx context.Context, userID string) (AccountKey, error) {
			return AccountKey{
				PublicKey: "pub-key",
				Algorithm: "Ed25519",
				VaultPath: "account-keys/user-1",
			}, nil
		},
	}
	auditLogger := &auditLoggerStub{
		log: func(ctx context.Context, input auditInput) error {
			if input.Action != "account.created" || input.ActorUserID != "user-1" || input.ResourceID != "user-1" {
				t.Fatalf("unexpected audit input: %#v", input)
			}
			return nil
		},
	}

	service := NewAccountService(
		authUserRepoStub{newUserID: "user-1"},
		userRepoStub{},
		&refreshTokenRepoStub{},
		&passwordResetTokenRepoStub{},
		keys,
		auditLogger,
		nil,
		issuerStub{},
		time.Hour,
		30*24*time.Hour,
		"google-client-id",
	)

	if _, err := service.Register(context.Background(), "user@example.com", "password123", "Demo User"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if auditLogger.logHits != 1 {
		t.Fatalf("expected one audit call, got %d", auditLogger.logHits)
	}
}

func TestAccountServiceLoginReturnsStoredPublicKey(t *testing.T) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	service := NewAccountService(
		authUserRepoStub{
			getByEmail: func(ctx context.Context, emailLower string) (domain.AuthUser, error) {
				return domain.AuthUser{
					UserID:       "user-1",
					EmailLower:   emailLower,
					PasswordHash: string(passwordHash),
					PublicKey:    "pub-key",
				}, nil
			},
		},
		userRepoStub{},
		&refreshTokenRepoStub{},
		&passwordResetTokenRepoStub{},
		nil,
		nil,
		nil,
		issuerStub{},
		time.Hour,
		30*24*time.Hour,
		"google-client-id",
	)

	result, err := service.Login(context.Background(), "user@example.com", "password123")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.AccessToken != "token-for-user-1" || result.RefreshToken == "" || result.Principal.UserID != "user-1" || result.PublicKey != "pub-key" {
		t.Fatalf("unexpected login response: %#v", result)
	}
}

func TestAccountServiceGoogleLoginCreatesVaultKeyForNewUser(t *testing.T) {
	keys := &accountKeyStoreStub{
		create: func(ctx context.Context, userID string) (AccountKey, error) {
			return AccountKey{
				PublicKey: "pub-key",
				Algorithm: "Ed25519",
				VaultPath: "account-keys/user-2",
			}, nil
		},
	}

	service := NewAccountService(
		authUserRepoStub{
			newUserID: "user-2",
			getByGoogleSub: func(ctx context.Context, googleSub string) (domain.AuthUser, error) {
				return domain.AuthUser{}, errors.New("not found")
			},
		},
		userRepoStub{},
		&refreshTokenRepoStub{},
		&passwordResetTokenRepoStub{},
		keys,
		nil,
		googleTokensStub{
			validate: func(ctx context.Context, idToken, audience string) (GoogleIdentity, error) {
				return GoogleIdentity{Subject: "google-sub", Email: "user@example.com"}, nil
			},
		},
		issuerStub{},
		time.Hour,
		30*24*time.Hour,
		"google-client-id",
	)

	result, err := service.GoogleLogin(context.Background(), "google-token")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.AccessToken != "token-for-user-2" || result.RefreshToken == "" || result.Principal.UserID != "user-2" || result.PublicKey != "pub-key" {
		t.Fatalf("unexpected google login response: %#v", result)
	}
	if keys.createHits != 1 {
		t.Fatalf("expected key generation once, got %d", keys.createHits)
	}
}

func TestAccountServiceGoogleLoginDoesNotRecreateKeyForExistingUser(t *testing.T) {
	keys := &accountKeyStoreStub{}
	service := NewAccountService(
		authUserRepoStub{
			getByGoogleSub: func(ctx context.Context, googleSub string) (domain.AuthUser, error) {
				return domain.AuthUser{
					UserID:     "user-3",
					EmailLower: "user@example.com",
					PublicKey:  "existing-pub",
				}, nil
			},
		},
		userRepoStub{},
		&refreshTokenRepoStub{},
		&passwordResetTokenRepoStub{},
		keys,
		nil,
		googleTokensStub{
			validate: func(ctx context.Context, idToken, audience string) (GoogleIdentity, error) {
				return GoogleIdentity{Subject: "google-sub", Email: "user@example.com"}, nil
			},
		},
		issuerStub{},
		time.Hour,
		30*24*time.Hour,
		"google-client-id",
	)

	result, err := service.GoogleLogin(context.Background(), "google-token")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.PublicKey != "existing-pub" {
		t.Fatalf("unexpected public key: %s", result.PublicKey)
	}
	if keys.createHits != 0 {
		t.Fatalf("expected no key recreation, got %d", keys.createHits)
	}
}

func TestAccountServiceLoginRejectsDeletedAccount(t *testing.T) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	service := NewAccountService(
		authUserRepoStub{
			getByEmail: func(ctx context.Context, emailLower string) (domain.AuthUser, error) {
				return domain.AuthUser{
					UserID:       "user-1",
					EmailLower:   emailLower,
					PasswordHash: string(passwordHash),
					Status:       AccountStatusDeleted,
				}, nil
			},
		},
		userRepoStub{},
		&refreshTokenRepoStub{},
		&passwordResetTokenRepoStub{},
		nil,
		nil,
		nil,
		issuerStub{},
		time.Hour,
		30*24*time.Hour,
		"google-client-id",
	)

	_, err = service.Login(context.Background(), "user@example.com", "password123")
	if !errors.Is(err, ErrAccountDeleted) {
		t.Fatalf("expected ErrAccountDeleted, got %v", err)
	}
}

func TestAccountServiceDeleteMarksAccountDeleted(t *testing.T) {
	auditLogger := &auditLoggerStub{}
	var savedAuthUser domain.AuthUser
	var savedUser domain.User
	service := NewAccountService(
		authUserRepoStub{
			getByID: func(ctx context.Context, userID string) (domain.AuthUser, error) {
				return domain.AuthUser{
					UserID:       userID,
					Status:       AccountStatusActive,
					Version:      1,
					PublicKey:    "pub-key",
					KeyAlgorithm: "Ed25519",
					VaultKeyPath: "account-keys/user-1",
				}, nil
			},
			save: func(ctx context.Context, user domain.AuthUser) error {
				savedAuthUser = user
				return nil
			},
		},
		userRepoStub{
			getByID: func(ctx context.Context, userID string) (domain.User, error) {
				return domain.User{UserID: userID, Status: AccountStatusActive, Version: 1}, nil
			},
			save: func(ctx context.Context, user domain.User) error {
				savedUser = user
				return nil
			},
		},
		&refreshTokenRepoStub{},
		&passwordResetTokenRepoStub{},
		nil,
		auditLogger,
		nil,
		issuerStub{},
		time.Hour,
		30*24*time.Hour,
		"google-client-id",
	)

	result, err := service.Delete(context.Background(), "user-1", 1)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.Status != AccountStatusDeleted {
		t.Fatalf("unexpected result: %#v", result)
	}
	if savedAuthUser.Status != AccountStatusDeleted || savedUser.Status != AccountStatusDeleted {
		t.Fatalf("expected deleted statuses, got auth=%s user=%s", savedAuthUser.Status, savedUser.Status)
	}
	if savedAuthUser.Version != 2 || savedUser.Version != 2 {
		t.Fatalf("expected incremented versions, got auth=%d user=%d", savedAuthUser.Version, savedUser.Version)
	}
	if auditLogger.logHits != 1 {
		t.Fatalf("expected one audit call, got %d", auditLogger.logHits)
	}
}

func TestAccountServiceDeleteRejectsVersionConflict(t *testing.T) {
	service := NewAccountService(
		authUserRepoStub{
			getByID: func(ctx context.Context, userID string) (domain.AuthUser, error) {
				return domain.AuthUser{UserID: userID, Status: AccountStatusActive, Version: 3}, nil
			},
		},
		userRepoStub{
			getByID: func(ctx context.Context, userID string) (domain.User, error) {
				return domain.User{UserID: userID, Status: AccountStatusActive, Version: 3}, nil
			},
		},
		&refreshTokenRepoStub{},
		&passwordResetTokenRepoStub{},
		nil,
		nil,
		nil,
		issuerStub{},
		time.Hour,
		30*24*time.Hour,
		"google-client-id",
	)

	_, err := service.Delete(context.Background(), "user-1", 2)
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
	}
}

func TestAccountServiceChangePassword(t *testing.T) {
	oldHash, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	var saved domain.AuthUser
	service := NewAccountService(
		authUserRepoStub{
			getByID: func(ctx context.Context, userID string) (domain.AuthUser, error) {
				return domain.AuthUser{
					UserID:       userID,
					EmailLower:   "user@example.com",
					PasswordHash: string(oldHash),
					Status:       AccountStatusActive,
					Version:      1,
				}, nil
			},
			save: func(ctx context.Context, user domain.AuthUser) error {
				saved = user
				return nil
			},
		},
		userRepoStub{},
		&refreshTokenRepoStub{},
		&passwordResetTokenRepoStub{},
		nil,
		nil,
		nil,
		issuerStub{},
		time.Hour,
		30*24*time.Hour,
		"google-client-id",
	)

	if err := service.ChangePassword(context.Background(), "user-1", "old-password", "new-password"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if saved.Version != 2 {
		t.Fatalf("expected version 2, got %d", saved.Version)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(saved.PasswordHash), []byte("new-password")); err != nil {
		t.Fatalf("saved password hash did not match new password: %v", err)
	}
}

func TestAccountServiceRequestAndResetPassword(t *testing.T) {
	resetTokens := &passwordResetTokenRepoStub{}
	var savedReset domain.PasswordResetToken
	resetTokens.save = func(ctx context.Context, token domain.PasswordResetToken) error {
		savedReset = token
		return nil
	}
	service := NewAccountService(
		authUserRepoStub{
			getByEmail: func(ctx context.Context, emailLower string) (domain.AuthUser, error) {
				return domain.AuthUser{UserID: "user-1", EmailLower: emailLower, Status: AccountStatusActive, Version: 1}, nil
			},
		},
		userRepoStub{},
		&refreshTokenRepoStub{},
		resetTokens,
		nil,
		nil,
		nil,
		issuerStub{},
		time.Hour,
		30*24*time.Hour,
		"google-client-id",
	)

	result, err := service.RequestPasswordReset(context.Background(), "user@example.com")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.ResetToken == "" || savedReset.TokenHash != hashSecret(result.ResetToken) {
		t.Fatalf("unexpected reset token result=%#v saved=%#v", result, savedReset)
	}

	var savedAuth domain.AuthUser
	resetTokens.getByTokenHash = func(ctx context.Context, tokenHash string) (domain.PasswordResetToken, error) {
		if tokenHash != savedReset.TokenHash {
			t.Fatalf("unexpected reset token hash: %s", tokenHash)
		}
		return savedReset, nil
	}
	service.authUsers = authUserRepoStub{
		getByID: func(ctx context.Context, userID string) (domain.AuthUser, error) {
			return domain.AuthUser{UserID: userID, PasswordHash: "old-hash", Status: AccountStatusActive, Version: 1}, nil
		},
		save: func(ctx context.Context, user domain.AuthUser) error {
			savedAuth = user
			return nil
		},
	}

	if err := service.ResetPassword(context.Background(), result.ResetToken, "new-password"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if savedAuth.Version != 2 {
		t.Fatalf("expected auth version 2, got %d", savedAuth.Version)
	}
	if savedReset.Status != PasswordResetTokenStatusUsed {
		t.Fatalf("expected reset token used, got %s", savedReset.Status)
	}
}
