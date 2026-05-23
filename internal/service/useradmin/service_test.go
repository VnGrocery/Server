package useradmin

import (
	"context"
	"errors"
	"testing"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	"vngrocery/internal/service/audit"
	authservice "vngrocery/internal/service/auth"
)

type userRepositoryStub struct {
	save    func(ctx context.Context, user domain.User) error
	getByID func(ctx context.Context, userID string) (domain.User, error)
	list    func(ctx context.Context, filter repository.UserListFilter) ([]domain.User, error)
}

func (s userRepositoryStub) Save(ctx context.Context, user domain.User) error {
	if s.save != nil {
		return s.save(ctx, user)
	}
	return nil
}

func (s userRepositoryStub) GetByID(ctx context.Context, userID string) (domain.User, error) {
	if s.getByID != nil {
		return s.getByID(ctx, userID)
	}
	return domain.User{}, errors.New("not found")
}

func (s userRepositoryStub) List(ctx context.Context, filter repository.UserListFilter) ([]domain.User, error) {
	if s.list != nil {
		return s.list(ctx, filter)
	}
	return nil, errors.New("not implemented")
}

type auditLoggerStub struct {
	logHits int
}

func (s *auditLoggerStub) Log(ctx context.Context, input audit.Input) error {
	s.logHits++
	return nil
}

var _ repository.UserRepository = userRepositoryStub{}

type accountKeyStoreStub struct {
	create func(ctx context.Context, userID string) (authservice.AccountKey, error)
}

func (s accountKeyStoreStub) CreateAccountKey(ctx context.Context, userID string) (authservice.AccountKey, error) {
	return s.create(ctx, userID)
}

func (s accountKeyStoreStub) DeleteAccountKey(ctx context.Context, vaultPath string) error {
	return nil
}

type authUserRepositoryStub struct {
	save    func(ctx context.Context, user domain.AuthUser) error
	getByID func(ctx context.Context, userID string) (domain.AuthUser, error)
}

func (s authUserRepositoryStub) NewUserID() string { return "" }
func (s authUserRepositoryStub) Save(ctx context.Context, user domain.AuthUser) error {
	if s.save != nil {
		return s.save(ctx, user)
	}
	return nil
}
func (s authUserRepositoryStub) GetByID(ctx context.Context, userID string) (domain.AuthUser, error) {
	if s.getByID != nil {
		return s.getByID(ctx, userID)
	}
	return domain.AuthUser{}, errors.New("not found")
}
func (s authUserRepositoryStub) GetByEmail(ctx context.Context, emailLower string) (domain.AuthUser, error) {
	return domain.AuthUser{}, errors.New("not implemented")
}
func (s authUserRepositoryStub) GetByGoogleSub(ctx context.Context, googleSub string) (domain.AuthUser, error) {
	return domain.AuthUser{}, errors.New("not implemented")
}

func TestUpdateRole(t *testing.T) {
	auditLogger := &auditLoggerStub{}
	var saved domain.User
	service := NewService(userRepositoryStub{
		getByID: func(ctx context.Context, userID string) (domain.User, error) {
			if userID == "admin-1" {
				return domain.User{UserID: userID, Role: RoleAdmin, Status: "active", Version: 1}, nil
			}
			return domain.User{UserID: userID, Role: RoleUser, Status: "active", Version: 2}, nil
		},
		save: func(ctx context.Context, user domain.User) error {
			saved = user
			return nil
		},
	}, authUserRepositoryStub{}, nil, auditLogger)

	user, err := service.UpdateRole(context.Background(), UpdateRoleInput{
		ActorUserID:     "admin-1",
		TargetUserID:    "user-1",
		ExpectedVersion: 2,
		Role:            RoleAdmin,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if user.Role != RoleAdmin || saved.Role != RoleAdmin || user.Version != 3 {
		t.Fatalf("unexpected updated user: user=%#v saved=%#v", user, saved)
	}
	if auditLogger.logHits != 1 {
		t.Fatalf("expected one audit call, got %d", auditLogger.logHits)
	}
}

func TestUpdateRoleRejectsNonAdmin(t *testing.T) {
	service := NewService(userRepositoryStub{
		getByID: func(ctx context.Context, userID string) (domain.User, error) {
			return domain.User{UserID: userID, Role: RoleUser, Status: "active", Version: 1}, nil
		},
	}, authUserRepositoryStub{}, nil, nil)

	_, err := service.UpdateRole(context.Background(), UpdateRoleInput{
		ActorUserID:     "user-1",
		TargetUserID:    "user-2",
		ExpectedVersion: 1,
		Role:            RoleAdmin,
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestUpdateRoleRejectsDemotingLastActiveAdmin(t *testing.T) {
	service := NewService(userRepositoryStub{
		getByID: func(ctx context.Context, userID string) (domain.User, error) {
			return domain.User{UserID: userID, Role: RoleAdmin, Status: StatusActive, Version: 1}, nil
		},
		list: func(ctx context.Context, filter repository.UserListFilter) ([]domain.User, error) {
			if filter.Role != RoleAdmin || filter.Status != StatusActive {
				t.Fatalf("unexpected filter: %+v", filter)
			}
			return []domain.User{{UserID: "admin-1", Role: RoleAdmin, Status: StatusActive}}, nil
		},
	}, authUserRepositoryStub{}, nil, nil)

	_, err := service.UpdateRole(context.Background(), UpdateRoleInput{
		ActorUserID:     "admin-1",
		TargetUserID:    "admin-1",
		ExpectedVersion: 1,
		Role:            RoleUser,
	})
	if !errors.Is(err, ErrInvalidUser) {
		t.Fatalf("expected ErrInvalidUser, got %v", err)
	}
}

func TestUpdateRoleAllowsDemotingAdminWhenAnotherActiveAdminExists(t *testing.T) {
	var saved domain.User
	service := NewService(userRepositoryStub{
		getByID: func(ctx context.Context, userID string) (domain.User, error) {
			if userID == "admin-2" {
				return domain.User{UserID: userID, Role: RoleAdmin, Status: StatusActive, Version: 1}, nil
			}
			return domain.User{UserID: userID, Role: RoleAdmin, Status: StatusActive, Version: 2}, nil
		},
		list: func(ctx context.Context, filter repository.UserListFilter) ([]domain.User, error) {
			return []domain.User{
				{UserID: "admin-1", Role: RoleAdmin, Status: StatusActive},
				{UserID: "admin-2", Role: RoleAdmin, Status: StatusActive},
			}, nil
		},
		save: func(ctx context.Context, user domain.User) error {
			saved = user
			return nil
		},
	}, authUserRepositoryStub{}, nil, nil)

	user, err := service.UpdateRole(context.Background(), UpdateRoleInput{
		ActorUserID:     "admin-2",
		TargetUserID:    "admin-1",
		ExpectedVersion: 2,
		Role:            RoleUser,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if user.Role != RoleUser || saved.Role != RoleUser {
		t.Fatalf("expected user role, user=%#v saved=%#v", user, saved)
	}
}

func TestListUsers(t *testing.T) {
	service := NewService(userRepositoryStub{
		getByID: func(ctx context.Context, userID string) (domain.User, error) {
			return domain.User{UserID: userID, Role: RoleAdmin, Status: StatusActive}, nil
		},
		list: func(ctx context.Context, filter repository.UserListFilter) ([]domain.User, error) {
			if filter.Status != StatusActive || filter.Role != RoleUser {
				t.Fatalf("unexpected filter: %+v", filter)
			}
			return []domain.User{{UserID: "user-1", Role: RoleUser, Status: StatusActive}}, nil
		},
	}, authUserRepositoryStub{}, nil, nil)

	users, err := service.List(context.Background(), ListInput{ActorUserID: "admin-1", Status: StatusActive, Role: RoleUser})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected one user, got %d", len(users))
	}
}

func TestListUsersDefaultsEmptyStatusToActive(t *testing.T) {
	service := NewService(userRepositoryStub{
		getByID: func(ctx context.Context, userID string) (domain.User, error) {
			return domain.User{UserID: userID, Role: RoleAdmin, Status: StatusActive}, nil
		},
		list: func(ctx context.Context, filter repository.UserListFilter) ([]domain.User, error) {
			return []domain.User{
				{UserID: "user-1", Role: RoleUser, Status: ""},
			}, nil
		},
	}, authUserRepositoryStub{}, nil, nil)

	users, err := service.List(context.Background(), ListInput{ActorUserID: "admin-1"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected one user, got %d", len(users))
	}
	if users[0].Status != StatusActive {
		t.Fatalf("expected default status active, got %q", users[0].Status)
	}
}

func TestUpdateStatusUpdatesUserAndAuthUser(t *testing.T) {
	auditLogger := &auditLoggerStub{}
	var savedUser domain.User
	var savedAuthUser domain.AuthUser
	service := NewService(userRepositoryStub{
		getByID: func(ctx context.Context, userID string) (domain.User, error) {
			if userID == "admin-1" {
				return domain.User{UserID: userID, Role: RoleAdmin, Status: StatusActive, Version: 1}, nil
			}
			return domain.User{UserID: userID, Role: RoleUser, Status: StatusActive, Version: 2}, nil
		},
		save: func(ctx context.Context, user domain.User) error {
			savedUser = user
			return nil
		},
	}, authUserRepositoryStub{
		getByID: func(ctx context.Context, userID string) (domain.AuthUser, error) {
			return domain.AuthUser{UserID: userID, Status: StatusActive, Version: 2}, nil
		},
		save: func(ctx context.Context, user domain.AuthUser) error {
			savedAuthUser = user
			return nil
		},
	}, nil, auditLogger)

	user, err := service.UpdateStatus(context.Background(), UpdateStatusInput{
		ActorUserID:     "admin-1",
		TargetUserID:    "user-1",
		ExpectedVersion: 2,
		Status:          StatusSuspended,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if user.Status != StatusSuspended || savedUser.Status != StatusSuspended || savedAuthUser.Status != StatusSuspended {
		t.Fatalf("unexpected statuses user=%#v saved=%#v auth=%#v", user, savedUser, savedAuthUser)
	}
	if savedUser.Version != 3 || savedAuthUser.Version != 3 {
		t.Fatalf("unexpected versions user=%d auth=%d", savedUser.Version, savedAuthUser.Version)
	}
	if auditLogger.logHits != 1 {
		t.Fatalf("expected one audit call, got %d", auditLogger.logHits)
	}
}

func TestUpdateStatusRejectsSuspendingLastActiveAdmin(t *testing.T) {
	service := NewService(userRepositoryStub{
		getByID: func(ctx context.Context, userID string) (domain.User, error) {
			return domain.User{UserID: userID, Role: RoleAdmin, Status: StatusActive, Version: 1}, nil
		},
		list: func(ctx context.Context, filter repository.UserListFilter) ([]domain.User, error) {
			if filter.Role != RoleAdmin || filter.Status != StatusActive {
				t.Fatalf("unexpected filter: %+v", filter)
			}
			return []domain.User{{UserID: "admin-1", Role: RoleAdmin, Status: StatusActive}}, nil
		},
	}, authUserRepositoryStub{
		getByID: func(ctx context.Context, userID string) (domain.AuthUser, error) {
			t.Fatal("auth user lookup should not happen after last-admin rejection")
			return domain.AuthUser{}, nil
		},
	}, nil, nil)

	_, err := service.UpdateStatus(context.Background(), UpdateStatusInput{
		ActorUserID:     "admin-1",
		TargetUserID:    "admin-1",
		ExpectedVersion: 1,
		Status:          StatusSuspended,
	})
	if !errors.Is(err, ErrInvalidUser) {
		t.Fatalf("expected ErrInvalidUser, got %v", err)
	}
}

func TestRotateAccountKey(t *testing.T) {
	auditLogger := &auditLoggerStub{}
	var saved domain.AuthUser
	service := NewService(userRepositoryStub{
		getByID: func(ctx context.Context, userID string) (domain.User, error) {
			return domain.User{UserID: userID, Role: RoleAdmin, Status: StatusActive, Version: 1}, nil
		},
	}, authUserRepositoryStub{
		getByID: func(ctx context.Context, userID string) (domain.AuthUser, error) {
			return domain.AuthUser{UserID: userID, PublicKey: "old-pub", KeyAlgorithm: "Ed25519", VaultKeyPath: "account-keys/user-1", Version: 4}, nil
		},
		save: func(ctx context.Context, user domain.AuthUser) error {
			saved = user
			return nil
		},
	}, accountKeyStoreStub{
		create: func(ctx context.Context, userID string) (authservice.AccountKey, error) {
			if userID != "user-1" {
				t.Fatalf("unexpected userID: %s", userID)
			}
			return authservice.AccountKey{PublicKey: "new-pub", Algorithm: "Ed25519", VaultPath: "account-keys/user-1"}, nil
		},
	}, auditLogger)

	result, err := service.RotateAccountKey(context.Background(), AccountKeyInput{
		ActorUserID:     "admin-1",
		TargetUserID:    "user-1",
		ExpectedVersion: 4,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.PublicKey != "new-pub" || saved.PublicKey != "new-pub" || result.Version != 5 {
		t.Fatalf("unexpected rotation result=%#v saved=%#v", result, saved)
	}
	if auditLogger.logHits != 1 {
		t.Fatalf("expected one audit call, got %d", auditLogger.logHits)
	}
}

func TestBackfillAccountKeyRejectsExistingMetadata(t *testing.T) {
	service := NewService(userRepositoryStub{
		getByID: func(ctx context.Context, userID string) (domain.User, error) {
			return domain.User{UserID: userID, Role: RoleAdmin, Status: StatusActive, Version: 1}, nil
		},
	}, authUserRepositoryStub{
		getByID: func(ctx context.Context, userID string) (domain.AuthUser, error) {
			return domain.AuthUser{UserID: userID, PublicKey: "pub", KeyAlgorithm: "Ed25519", VaultKeyPath: "account-keys/user-1", Version: 1}, nil
		},
	}, accountKeyStoreStub{
		create: func(ctx context.Context, userID string) (authservice.AccountKey, error) {
			t.Fatal("key creation should not be called")
			return authservice.AccountKey{}, nil
		},
	}, nil)

	_, err := service.BackfillAccountKey(context.Background(), AccountKeyInput{
		ActorUserID:     "admin-1",
		TargetUserID:    "user-1",
		ExpectedVersion: 1,
	})
	if !errors.Is(err, ErrInvalidUser) {
		t.Fatalf("expected ErrInvalidUser, got %v", err)
	}
}
