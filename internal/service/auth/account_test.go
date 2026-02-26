package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"vngrocery/internal/domain"
)

type authUserRepoStub struct {
	newUserID      string
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
	save func(ctx context.Context, user domain.User) error
}

func (s userRepoStub) Save(ctx context.Context, user domain.User) error {
	if s.save != nil {
		return s.save(ctx, user)
	}
	return nil
}

func (s userRepoStub) GetByID(ctx context.Context, userID string) (domain.User, error) {
	return domain.User{}, errors.New("not implemented")
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
		keys,
		nil,
		issuerStub{},
		time.Hour,
		"google-client-id",
	)

	token, principal, publicKey, err := service.Register(context.Background(), "USER@example.com", "password123", "Demo User")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if token != "token-for-user-1" {
		t.Fatalf("unexpected token: %s", token)
	}
	if principal.UserID != "user-1" {
		t.Fatalf("unexpected principal: %#v", principal)
	}
	if publicKey != "pub-key" {
		t.Fatalf("unexpected public key: %s", publicKey)
	}
	if savedAuthUser.PublicKey != "pub-key" || savedAuthUser.KeyAlgorithm != "Ed25519" || savedAuthUser.VaultKeyPath != "account-keys/user-1" {
		t.Fatalf("auth user key metadata not persisted: %#v", savedAuthUser)
	}
	if savedUser.Email != "user@example.com" {
		t.Fatalf("unexpected saved user email: %s", savedUser.Email)
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
		keys,
		nil,
		issuerStub{},
		time.Hour,
		"google-client-id",
	)

	_, _, _, err := service.Register(context.Background(), "user@example.com", "password123", "Demo User")
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
		keys,
		nil,
		issuerStub{},
		time.Hour,
		"google-client-id",
	)

	_, _, _, err := service.Register(context.Background(), "user@example.com", "password123", "Demo User")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if keys.deleteHits != 1 {
		t.Fatalf("expected cleanup delete call, got %d", keys.deleteHits)
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
		nil,
		nil,
		issuerStub{},
		time.Hour,
		"google-client-id",
	)

	token, principal, publicKey, err := service.Login(context.Background(), "user@example.com", "password123")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if token != "token-for-user-1" || principal.UserID != "user-1" || publicKey != "pub-key" {
		t.Fatalf("unexpected login response: token=%s principal=%#v publicKey=%s", token, principal, publicKey)
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
		keys,
		googleTokensStub{
			validate: func(ctx context.Context, idToken, audience string) (GoogleIdentity, error) {
				return GoogleIdentity{Subject: "google-sub", Email: "user@example.com"}, nil
			},
		},
		issuerStub{},
		time.Hour,
		"google-client-id",
	)

	token, principal, publicKey, err := service.GoogleLogin(context.Background(), "google-token")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if token != "token-for-user-2" || principal.UserID != "user-2" || publicKey != "pub-key" {
		t.Fatalf("unexpected google login response: token=%s principal=%#v publicKey=%s", token, principal, publicKey)
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
		keys,
		googleTokensStub{
			validate: func(ctx context.Context, idToken, audience string) (GoogleIdentity, error) {
				return GoogleIdentity{Subject: "google-sub", Email: "user@example.com"}, nil
			},
		},
		issuerStub{},
		time.Hour,
		"google-client-id",
	)

	_, _, publicKey, err := service.GoogleLogin(context.Background(), "google-token")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if publicKey != "existing-pub" {
		t.Fatalf("unexpected public key: %s", publicKey)
	}
	if keys.createHits != 0 {
		t.Fatalf("expected no key recreation, got %d", keys.createHits)
	}
}
