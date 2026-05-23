package main

import (
	"context"
	"testing"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
)

type fakeUserRepo struct {
	users []domain.User
	saved []domain.User
}

func (r *fakeUserRepo) List(ctx context.Context, filter repository.UserListFilter) ([]domain.User, error) {
	return r.users, nil
}

func (r *fakeUserRepo) Save(ctx context.Context, user domain.User) error {
	r.saved = append(r.saved, user)
	return nil
}

func TestMigrateUserRolesDryRunMapsLegacyRoles(t *testing.T) {
	repo := &fakeUserRepo{users: []domain.User{
		{UserID: "admin-1", Role: domain.RoleAdmin},
		{UserID: "seller-1", Role: "seller"},
		{UserID: "buyer-1", Role: "buyer"},
		{UserID: "user-1", Role: domain.RoleUser},
	}}

	stats, err := migrateUserRoles(context.Background(), repo, false, false)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if stats.Total != 4 || stats.Changed != 2 || stats.Skipped != 2 || stats.Legacy["seller"] != 1 || stats.Legacy["buyer"] != 1 {
		t.Fatalf("unexpected stats: %#v", stats)
	}
	if len(repo.saved) != 0 {
		t.Fatalf("dry-run should not save, got %d saves", len(repo.saved))
	}
}

func TestMigrateUserRolesApplyUpdatesLegacyRoles(t *testing.T) {
	repo := &fakeUserRepo{users: []domain.User{
		{UserID: "seller-1", Role: "seller", Version: 3},
		{UserID: "buyer-1", Role: "buyer", Version: 4},
	}}

	stats, err := migrateUserRoles(context.Background(), repo, true, false)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if stats.Changed != 2 || len(repo.saved) != 2 {
		t.Fatalf("unexpected stats or save count: stats=%#v saved=%d", stats, len(repo.saved))
	}
	for _, saved := range repo.saved {
		if saved.Role != domain.RoleUser {
			t.Fatalf("expected saved role user, got %#v", saved)
		}
		if saved.UpdatedAt.IsZero() {
			t.Fatalf("expected UpdatedAt to be set: %#v", saved)
		}
	}
	if repo.saved[0].Version != 4 || repo.saved[1].Version != 5 {
		t.Fatalf("expected versions to increment, got %#v", repo.saved)
	}
}

func TestMigrateUserRolesSkipsUnknownUnlessRequested(t *testing.T) {
	repo := &fakeUserRepo{users: []domain.User{{UserID: "unknown-1", Role: "manager", Version: 1}}}
	stats, err := migrateUserRoles(context.Background(), repo, true, false)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if stats.Changed != 0 || len(repo.saved) != 0 || stats.Unknown["manager"] != 1 {
		t.Fatalf("unexpected unknown handling: stats=%#v saved=%d", stats, len(repo.saved))
	}

	stats, err = migrateUserRoles(context.Background(), repo, true, true)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if stats.Changed != 1 || len(repo.saved) != 1 || repo.saved[0].Role != domain.RoleUser {
		t.Fatalf("expected unknown role to map to user: stats=%#v saved=%#v", stats, repo.saved)
	}
}
