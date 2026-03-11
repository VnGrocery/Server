package useradmin

import (
	"context"
	"errors"
	"testing"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	"vngrocery/internal/service/audit"
)

type userRepositoryStub struct {
	save    func(ctx context.Context, user domain.User) error
	getByID func(ctx context.Context, userID string) (domain.User, error)
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

type auditLoggerStub struct {
	logHits int
}

func (s *auditLoggerStub) Log(ctx context.Context, input audit.Input) error {
	s.logHits++
	return nil
}

var _ repository.UserRepository = userRepositoryStub{}

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
	}, auditLogger)

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
	}, nil)

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
