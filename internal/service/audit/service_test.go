package audit

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
)

type eventLogRepoStub struct {
	save      func(ctx context.Context, event domain.EventLog) error
	getByID   func(ctx context.Context, eventID string) (domain.EventLog, error)
	getLatest func(ctx context.Context, resourceType, resourceID string) (domain.EventLog, error)
	list      func(ctx context.Context, filter repository.EventLogListFilter) ([]domain.EventLog, error)
}

func (s eventLogRepoStub) Save(ctx context.Context, event domain.EventLog) error {
	if s.save != nil {
		return s.save(ctx, event)
	}
	return nil
}

func (s eventLogRepoStub) GetByID(ctx context.Context, eventID string) (domain.EventLog, error) {
	if s.getByID != nil {
		return s.getByID(ctx, eventID)
	}
	return domain.EventLog{}, errors.New("not found")
}

func (s eventLogRepoStub) GetLatestByResource(ctx context.Context, resourceType, resourceID string) (domain.EventLog, error) {
	if s.getLatest != nil {
		return s.getLatest(ctx, resourceType, resourceID)
	}
	return domain.EventLog{}, nil
}

func (s eventLogRepoStub) List(ctx context.Context, filter repository.EventLogListFilter) ([]domain.EventLog, error) {
	if s.list != nil {
		return s.list(ctx, filter)
	}
	return []domain.EventLog{}, nil
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

func TestListAppliesFiltersAndPagination(t *testing.T) {
	now := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	service := NewService(
		eventLogRepoStub{
			list: func(ctx context.Context, filter repository.EventLogListFilter) ([]domain.EventLog, error) {
				if filter.ResourceType != "shop" || filter.ResourceID != "shop-1" || filter.ActorUserID != "user-1" {
					t.Fatalf("unexpected base filters: %+v", filter)
				}
				if filter.Action != "shop.updated" || filter.Status != "updated" || filter.MinSequence != 2 || filter.MaxSequence != 5 {
					t.Fatalf("unexpected extended filters: %+v", filter)
				}
				if !filter.CreatedAfter.Equal(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)) ||
					!filter.CreatedBefore.Equal(time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)) {
					t.Fatalf("unexpected time filters: %+v", filter)
				}
				return []domain.EventLog{
					{EventID: "event-1", CreatedAt: now},
					{EventID: "event-2", CreatedAt: now.Add(-time.Minute)},
					{EventID: "event-3", CreatedAt: now.Add(-2 * time.Minute)},
				}, nil
			},
		},
		nil,
		nil,
	)

	result, err := service.List(context.Background(), ListInput{
		ResourceType:  "shop",
		ResourceID:    "shop-1",
		ActorUserID:   "user-1",
		Action:        "shop.updated",
		Status:        "updated",
		MinSequence:   2,
		MaxSequence:   5,
		CreatedAfter:  time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		CreatedBefore: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC),
		Page:          2,
		PageSize:      1,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.Total != 3 || result.Page != 2 || result.PageSize != 1 {
		t.Fatalf("unexpected pagination result: %+v", result)
	}
	if len(result.Items) != 1 || result.Items[0].EventID != "event-2" {
		t.Fatalf("unexpected paginated items: %#v", result.Items)
	}
}

func TestListRejectsInvalidRanges(t *testing.T) {
	service := NewService(eventLogRepoStub{}, nil, nil)

	_, err := service.List(context.Background(), ListInput{
		MinSequence: 5,
		MaxSequence: 2,
	})
	if err == nil || err.Error() != "minSequence must be less than or equal to maxSequence" {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = service.List(context.Background(), ListInput{
		CreatedAfter:  time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC),
		CreatedBefore: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	})
	if err == nil || err.Error() != "createdAfter must be before or equal to createdBefore" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListAllowsEmptyFilter(t *testing.T) {
	service := NewService(
		eventLogRepoStub{
			list: func(ctx context.Context, filter repository.EventLogListFilter) ([]domain.EventLog, error) {
				return []domain.EventLog{{EventID: "event-1"}, {EventID: "event-2"}}, nil
			},
		},
		nil,
		nil,
	)

	result, err := service.List(context.Background(), ListInput{})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.Total != 2 || len(result.Items) != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestVerifyEventAndResourceChain(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate keypair: %v", err)
	}
	now := time.Date(2026, 4, 10, 1, 2, 3, 0, time.UTC)
	makeEvent := func(id string, sequence int, previousID string) domain.EventLog {
		event := domain.EventLog{
			EventID:         id,
			ActorUserID:     "user-1",
			ResourceType:    "shop",
			ResourceID:      "shop-1",
			ResourceVersion: sequence,
			Action:          "shop.updated",
			Status:          "updated",
			Sequence:        sequence,
			PreviousEventID: previousID,
			PayloadJSON:     `{"after":{"name":"Green Shop"}}`,
			PublicKey:       base64.StdEncoding.EncodeToString(publicKey),
			KeyAlgorithm:    "Ed25519",
			CreatedAt:       now.Add(time.Duration(sequence) * time.Minute),
		}
		envelopeBytes, err := signedEnvelopeBytes(event)
		if err != nil {
			t.Fatalf("failed to build envelope: %v", err)
		}
		signature := ed25519.Sign(privateKey, envelopeBytes)
		hash := sha256.Sum256(envelopeBytes)
		event.Signature = base64.StdEncoding.EncodeToString(signature)
		event.ContentSHA256 = base64.StdEncoding.EncodeToString(hash[:])
		return event
	}

	event1 := makeEvent("event-1", 1, "")
	event2 := makeEvent("event-2", 2, "event-1")
	service := NewService(
		eventLogRepoStub{
			getByID: func(ctx context.Context, eventID string) (domain.EventLog, error) {
				switch eventID {
				case "event-1":
					return event1, nil
				case "event-2":
					return event2, nil
				default:
					return domain.EventLog{}, errors.New("not found")
				}
			},
			list: func(ctx context.Context, filter repository.EventLogListFilter) ([]domain.EventLog, error) {
				return []domain.EventLog{event2, event1}, nil
			},
		},
		nil,
		nil,
	)

	verifiedEvent, err := service.VerifyEvent(context.Background(), VerifyEventInput{EventID: "event-2"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !verifiedEvent.Verified || !verifiedEvent.SignatureValid || !verifiedEvent.ContentHashValid || !verifiedEvent.ChainLinkValid {
		t.Fatalf("unexpected event verification result: %+v", verifiedEvent)
	}

	verifiedResource, err := service.VerifyResource(context.Background(), VerifyResourceInput{
		ResourceType: "shop",
		ResourceID:   "shop-1",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !verifiedResource.Verified || verifiedResource.EventCount != 2 || len(verifiedResource.Events) != 2 {
		t.Fatalf("unexpected resource verification result: %+v", verifiedResource)
	}
}

// The reason is signed with the event, and adding the field must not disturb
// the 856 events written before it existed: their envelopes have to serialize
// byte-for-byte as they did, or every one of them stops verifying.
func TestSignedEnvelopeKeepsOldEventsByteIdentical(t *testing.T) {
	withReason, err := json.Marshal(signedEnvelope{
		Action:          "shop.updated",
		ActorUserID:     "user-1",
		OccurredAt:      "2026-08-24T00:00:00Z",
		Payload:         json.RawMessage(`{"after":1}`),
		Reason:          "Đổi địa chỉ sạp",
		ResourceID:      "shop-1",
		ResourceType:    "shop",
		ResourceVersion: 2,
		Sequence:        2,
		PreviousEventID: "event-1",
	})
	if err != nil {
		t.Fatalf("marshal with reason: %v", err)
	}
	if !strings.Contains(string(withReason), `"reason":"Đổi địa chỉ sạp"`) {
		t.Fatalf("the reason has to be inside the signed bytes: %s", withReason)
	}

	withoutReason, err := json.Marshal(signedEnvelope{
		Action:          "shop.updated",
		ActorUserID:     "user-1",
		OccurredAt:      "2026-08-24T00:00:00Z",
		Payload:         json.RawMessage(`{"after":1}`),
		ResourceID:      "shop-1",
		ResourceType:    "shop",
		ResourceVersion: 2,
		Sequence:        2,
		PreviousEventID: "event-1",
	})
	if err != nil {
		t.Fatalf("marshal without reason: %v", err)
	}
	if strings.Contains(string(withoutReason), "reason") {
		t.Fatalf("an event with no reason must serialize exactly as before: %s", withoutReason)
	}
}
