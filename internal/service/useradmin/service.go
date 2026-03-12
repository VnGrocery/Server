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
)

var (
	ErrInvalidUser     = errors.New("invalid user request")
	ErrForbidden       = errors.New("forbidden")
	ErrNotFound        = errors.New("user not found")
	ErrVersionConflict = errors.New("version conflict")
)

const (
	RoleAdmin       = "admin"
	RoleUser        = "user"
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

type AuditLogger interface {
	Log(ctx context.Context, input audit.Input) error
}

type Service struct {
	users     repository.UserRepository
	authUsers repository.AuthUserRepository
	audit     AuditLogger
	now       func() time.Time
}

func NewService(users repository.UserRepository, authUsers repository.AuthUserRepository, auditLogger AuditLogger) *Service {
	return &Service{
		users:     users,
		authUsers: authUsers,
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
	if role != RoleAdmin && role != RoleUser {
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
