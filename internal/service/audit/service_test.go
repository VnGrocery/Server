package audit

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"vngrocery/internal/domain"
)

type eventLogRepoStub struct {
	save      func(ctx context.Context, event domain.EventLog) error
	getLatest func(ctx context.Context, resourceType, resourceID string) (domain.EventLog, error)
}

func (s eventLogRepoStub) Save(ctx context.Context, event domain.EventLog) error {
	if s.save != nil {
		return s.save(ctx, event)
	}
	return nil
}

func (s eventLogRepoStub) GetLatestByResource(ctx context.Context, resourceType, resourceID string) (domain.EventLog, error) {
	if s.getLatest != nil {
		return s.getLatest(ctx, resourceType, resourceID)
	}
	return domain.EventLog{}, nil
}

type authUserRepoStub struct {
	getByID func(ctx context.Context, userID string) (domain.AuthUser, error)
}

func (s authUserRepoStub) NewUserID() string                                    { return "" }
func (s authUserRepoStub) Save(ctx context.Context, user domain.AuthUser) error { return nil }
func (s authUserRepoStub) GetByID(ctx context.Context, userID string) (domain.AuthUser, error) {
	if s.getByID != nil {
		return s.getByID(ctx, userID)
	}
	return domain.AuthUser{}, errors.New("not found")
}
func (s authUserRepoStub) GetByEmail(ctx context.Context, emailLower string) (domain.AuthUser, error) {
	return domain.AuthUser{}, errors.New("not implemented")
}
func (s authUserRepoStub) GetByGoogleSub(ctx context.Context, googleSub string) (domain.AuthUser, error) {
	return domain.AuthUser{}, errors.New("not implemented")
}

type signerStub struct {
	sign func(ctx context.Context, vaultPath string, message []byte) (string, error)
}

func (s signerStub) SignAccountEvent(ctx context.Context, vaultPath string, message []byte) (string, error) {
	return s.sign(ctx, vaultPath, message)
}

func TestLogSignsAndSavesEvent(t *testing.T) {
	var saved domain.EventLog
	service := NewService(
		eventLogRepoStub{
			save: func(ctx context.Context, event domain.EventLog) error {
				saved = event
				return nil
			},
			getLatest: func(ctx context.Context, resourceType, resourceID string) (domain.EventLog, error) {
				return domain.EventLog{EventID: "event-0", Sequence: 2}, nil
			},
		},
		authUserRepoStub{
			getByID: func(ctx context.Context, userID string) (domain.AuthUser, error) {
				return domain.AuthUser{
					UserID:       userID,
					PublicKey:    "pub-key",
					KeyAlgorithm: "Ed25519",
					VaultKeyPath: "account-keys/user-1",
				}, nil
			},
		},
		signerStub{
			sign: func(ctx context.Context, vaultPath string, message []byte) (string, error) {
				if vaultPath != "account-keys/user-1" {
					t.Fatalf("unexpected vault path: %s", vaultPath)
				}
				var envelope map[string]any
				if err := json.Unmarshal(message, &envelope); err != nil {
					t.Fatalf("failed to decode envelope: %v", err)
				}
				if envelope["action"] != "shop.updated" {
					t.Fatalf("unexpected action: %#v", envelope["action"])
				}
				if envelope["sequence"] != float64(3) {
					t.Fatalf("unexpected sequence: %#v", envelope["sequence"])
				}
				if envelope["previousEventId"] != "event-0" {
					t.Fatalf("unexpected previousEventId: %#v", envelope["previousEventId"])
				}
				return "signature", nil
			},
		},
	)

	err := service.Log(context.Background(), Input{
		ActorUserID:     "user-1",
		ResourceType:    "shop",
		ResourceID:      "shop-1",
		ResourceVersion: 4,
		Action:          "shop.updated",
		Status:          "updated",
		Payload: map[string]any{
			"after": map[string]any{"name": "Green Shop"},
		},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if saved.ActorUserID != "user-1" || saved.ResourceID != "shop-1" || saved.Signature != "signature" {
		t.Fatalf("unexpected saved event: %#v", saved)
	}
	if saved.Status != "updated" {
		t.Fatalf("unexpected status: %s", saved.Status)
	}
	if saved.ResourceVersion != 4 || saved.Sequence != 3 || saved.PreviousEventID != "event-0" {
		t.Fatalf("unexpected event chain fields: %#v", saved)
	}
	if saved.PublicKey != "pub-key" || saved.KeyAlgorithm != "Ed25519" {
		t.Fatalf("unexpected signer metadata: %#v", saved)
	}
	if saved.PayloadJSON == "" || saved.ContentSHA256 == "" {
		t.Fatalf("expected payload and hash to be persisted: %#v", saved)
	}
}
