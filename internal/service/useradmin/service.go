package useradmin

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	"vngrocery/internal/service/audit"
	authservice "vngrocery/internal/service/auth"
)

var (
	ErrInvalidUser     = errors.New("invalid user request")
	ErrForbidden       = errors.New("forbidden")
	ErrNotFound        = errors.New("user not found")
	ErrVersionConflict = errors.New("version conflict")
)

const (
	RoleAdmin = "admin"
	RoleUser  = "user"

	// A seller is a user an admin has cleared to open a shop. Opening one is
	// not self-service: the shop is what the whole signed record hangs off, so
	// somebody has to vouch for the account behind it first.
	RoleSeller      = "seller"
	StatusActive    = "active"
	StatusSuspended = "suspended"
	StatusDeleted   = "deleted"
)

type UpdateRoleInput struct {
	ActorUserID     string
	TargetUserID    string
	ExpectedVersion int
	Role            string
}

type ListInput struct {
	ActorUserID string
	Status      string
	Role        string
}

type UpdateStatusInput struct {
	ActorUserID     string
	TargetUserID    string
	ExpectedVersion int
	Status          string
}

type AccountKeyInput struct {
	ActorUserID     string
	TargetUserID    string
	ExpectedVersion int
	Mode            string
}

type AccountKeyResult struct {
	UserID       string
	PublicKey    string
	KeyAlgorithm string
	VaultKeyPath string
	Version      int
}

type AuditLogger interface {
	Log(ctx context.Context, input audit.Input) error
}

type Service struct {
	users     repository.UserRepository
	authUsers repository.AuthUserRepository
	keys      authservice.AccountKeyStore
	audit     AuditLogger
	now       func() time.Time
}

func NewService(users repository.UserRepository, authUsers repository.AuthUserRepository, keys authservice.AccountKeyStore, auditLogger AuditLogger) *Service {
	return &Service{
		users:     users,
		authUsers: authUsers,
		keys:      keys,
		audit:     auditLogger,
		now:       time.Now,
	}
}

func (s *Service) List(ctx context.Context, input ListInput) ([]domain.User, error) {
	if strings.TrimSpace(input.ActorUserID) == "" {
		return nil, fmt.Errorf("%w: actorUserId is required", ErrInvalidUser)
	}
	if s.users == nil {
		return nil, fmt.Errorf("user repository is not configured")
	}
	if err := s.ensureAdmin(ctx, input.ActorUserID); err != nil {
		return nil, err
	}

	users, err := s.users.List(ctx, repository.UserListFilter{
		Status: strings.TrimSpace(input.Status),
		Role:   strings.TrimSpace(input.Role),
	})
	if err != nil {
		return nil, err
	}
	for i := range users {
		if strings.TrimSpace(users[i].Status) == "" {
			users[i].Status = StatusActive
		}
	}
	sort.Slice(users, func(i, j int) bool {
		return users[i].CreatedAt.Before(users[j].CreatedAt)
	})
	return users, nil
}

func (s *Service) UpdateRole(ctx context.Context, input UpdateRoleInput) (domain.User, error) {
	if strings.TrimSpace(input.ActorUserID) == "" || strings.TrimSpace(input.TargetUserID) == "" {
		return domain.User{}, fmt.Errorf("%w: actorUserId and targetUserId are required", ErrInvalidUser)
	}
	role := strings.TrimSpace(input.Role)
	if role != RoleAdmin && role != RoleUser && role != RoleSeller {
		return domain.User{}, fmt.Errorf("%w: unsupported role", ErrInvalidUser)
	}
	if input.ExpectedVersion <= 0 {
		return domain.User{}, ErrVersionConflict
	}
	if s.users == nil {
		return domain.User{}, fmt.Errorf("user repository is not configured")
	}
	if err := s.ensureAdmin(ctx, input.ActorUserID); err != nil {
		return domain.User{}, err
	}

	user, err := s.users.GetByID(ctx, strings.TrimSpace(input.TargetUserID))
	if err != nil || user.UserID == "" {
		return domain.User{}, ErrNotFound
	}
	if user.Version != input.ExpectedVersion {
		return domain.User{}, ErrVersionConflict
	}

	before := user
	user.Role = role
	user.Version++
	user.UpdatedAt = s.now().UTC()
	if err := s.users.Save(ctx, user); err != nil {
		return domain.User{}, err
	}
	if err := s.logMutation(ctx, strings.TrimSpace(input.ActorUserID), user, "user.role_updated", before); err != nil {
		return domain.User{}, err
	}
	return user, nil
}

func (s *Service) UpdateStatus(ctx context.Context, input UpdateStatusInput) (domain.User, error) {
	if strings.TrimSpace(input.ActorUserID) == "" || strings.TrimSpace(input.TargetUserID) == "" {
		return domain.User{}, fmt.Errorf("%w: actorUserId and targetUserId are required", ErrInvalidUser)
	}
	status := strings.TrimSpace(input.Status)
	if status != StatusActive && status != StatusSuspended && status != StatusDeleted {
		return domain.User{}, fmt.Errorf("%w: unsupported status", ErrInvalidUser)
	}
	if input.ExpectedVersion <= 0 {
		return domain.User{}, ErrVersionConflict
	}
	if s.users == nil || s.authUsers == nil {
		return domain.User{}, fmt.Errorf("user moderation dependencies are not configured")
	}
	if err := s.ensureAdmin(ctx, input.ActorUserID); err != nil {
		return domain.User{}, err
	}

	user, err := s.users.GetByID(ctx, strings.TrimSpace(input.TargetUserID))
	if err != nil || user.UserID == "" {
		return domain.User{}, ErrNotFound
	}
	if user.Version != input.ExpectedVersion {
		return domain.User{}, ErrVersionConflict
	}
	authUser, err := s.authUsers.GetByID(ctx, user.UserID)
	if err != nil || authUser.UserID == "" {
		return domain.User{}, ErrNotFound
	}

	before := user
	now := s.now().UTC()
	user.Status = status
	user.Version++
	user.UpdatedAt = now
	if err := s.users.Save(ctx, user); err != nil {
		return domain.User{}, err
	}

	authUser.Status = status
	authUser.Version++
	authUser.UpdatedAt = now
	if err := s.authUsers.Save(ctx, authUser); err != nil {
		return domain.User{}, err
	}

	if err := s.logMutation(ctx, strings.TrimSpace(input.ActorUserID), user, "user.status_updated", before); err != nil {
		return domain.User{}, err
	}
	return user, nil
}

func (s *Service) RotateAccountKey(ctx context.Context, input AccountKeyInput) (AccountKeyResult, error) {
	return s.createAccountKey(ctx, input, "account.key_rotated", false)
}

func (s *Service) RecoverAccountKey(ctx context.Context, input AccountKeyInput) (AccountKeyResult, error) {
	return s.createAccountKey(ctx, input, "account.key_recovered", false)
}

func (s *Service) BackfillAccountKey(ctx context.Context, input AccountKeyInput) (AccountKeyResult, error) {
	return s.createAccountKey(ctx, input, "account.key_backfilled", true)
}

func (s *Service) ensureAdmin(ctx context.Context, userID string) error {
	actor, err := s.users.GetByID(ctx, strings.TrimSpace(userID))
	if err != nil || actor.UserID == "" {
		return ErrForbidden
	}
	if actor.Role != RoleAdmin {
		return ErrForbidden
	}
	return nil
}

func (s *Service) logMutation(ctx context.Context, actorUserID string, user domain.User, action string, before domain.User) error {
	if s.audit == nil {
		return nil
	}
	return s.audit.Log(ctx, audit.Input{
		ActorUserID:     actorUserID,
		ResourceType:    "user",
		ResourceID:      user.UserID,
		ResourceVersion: user.Version,
		Action:          action,
		Status:          "updated",
		Payload: audit.MutationPayload{
			Before: before,
			After:  user,
		},
	})
}

func (s *Service) createAccountKey(ctx context.Context, input AccountKeyInput, action string, requireMissing bool) (AccountKeyResult, error) {
	if strings.TrimSpace(input.ActorUserID) == "" || strings.TrimSpace(input.TargetUserID) == "" {
		return AccountKeyResult{}, fmt.Errorf("%w: actorUserId and targetUserId are required", ErrInvalidUser)
	}
	if input.ExpectedVersion <= 0 {
		return AccountKeyResult{}, ErrVersionConflict
	}
	if s.users == nil || s.authUsers == nil || s.keys == nil {
		return AccountKeyResult{}, fmt.Errorf("account key management dependencies are not configured")
	}
	if err := s.ensureAdmin(ctx, input.ActorUserID); err != nil {
		return AccountKeyResult{}, err
	}

	authUser, err := s.authUsers.GetByID(ctx, strings.TrimSpace(input.TargetUserID))
	if err != nil || authUser.UserID == "" {
		return AccountKeyResult{}, ErrNotFound
	}
	if authUser.Version != input.ExpectedVersion {
		return AccountKeyResult{}, ErrVersionConflict
	}
	if requireMissing && authUser.PublicKey != "" && authUser.KeyAlgorithm != "" && authUser.VaultKeyPath != "" {
		return AccountKeyResult{}, fmt.Errorf("%w: account key metadata already exists", ErrInvalidUser)
	}

	before := authUser
	key, err := s.keys.CreateAccountKey(ctx, authUser.UserID)
	if err != nil {
		return AccountKeyResult{}, err
	}

	authUser.PublicKey = key.PublicKey
	authUser.KeyAlgorithm = key.Algorithm
	authUser.VaultKeyPath = key.VaultPath
	authUser.Version++
	authUser.UpdatedAt = s.now().UTC()
	if err := s.authUsers.Save(ctx, authUser); err != nil {
		return AccountKeyResult{}, err
	}
	if err := s.logAuthUserMutation(ctx, strings.TrimSpace(input.ActorUserID), authUser, action, before); err != nil {
		return AccountKeyResult{}, err
	}

	return AccountKeyResult{
		UserID:       authUser.UserID,
		PublicKey:    authUser.PublicKey,
		KeyAlgorithm: authUser.KeyAlgorithm,
		VaultKeyPath: authUser.VaultKeyPath,
		Version:      authUser.Version,
	}, nil
}

func (s *Service) logAuthUserMutation(ctx context.Context, actorUserID string, authUser domain.AuthUser, action string, before domain.AuthUser) error {
	if s.audit == nil {
		return nil
	}
	return s.audit.Log(ctx, audit.Input{
		ActorUserID:     actorUserID,
		ResourceType:    "account",
		ResourceID:      authUser.UserID,
		ResourceVersion: authUser.Version,
		Action:          action,
		Status:          "updated",
		Payload: audit.MutationPayload{
			Before: map[string]any{
				"userId":       before.UserID,
				"publicKey":    before.PublicKey,
				"keyAlgorithm": before.KeyAlgorithm,
				"vaultKeyPath": before.VaultKeyPath,
				"version":      before.Version,
			},
			After: map[string]any{
				"userId":       authUser.UserID,
				"publicKey":    authUser.PublicKey,
				"keyAlgorithm": authUser.KeyAlgorithm,
				"vaultKeyPath": authUser.VaultKeyPath,
				"version":      authUser.Version,
			},
		},
	})
}
