package audit

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
)

type Signer interface {
	SignAccountEvent(ctx context.Context, vaultPath string, message []byte) (string, error)
}

type Input struct {
	ActorUserID    string
	ResourceType   string
	ResourceID     string
	Action         string
	Status         string
	Payload        any
	PublicKey      string
	KeyAlgorithm   string
	SignerVaultKey string
}

type Service struct {
	events    repository.EventLogRepository
	authUsers repository.AuthUserRepository
	signer    Signer
	now       func() time.Time
}

func NewService(events repository.EventLogRepository, authUsers repository.AuthUserRepository, signer Signer) *Service {
	return &Service{
		events:    events,
		authUsers: authUsers,
		signer:    signer,
		now:       time.Now,
	}
}

type signedEnvelope struct {
	Action       string          `json:"action"`
	ActorUserID  string          `json:"actorUserId"`
	OccurredAt   string          `json:"occurredAt"`
	Payload      json.RawMessage `json:"payload"`
	ResourceID   string          `json:"resourceId"`
	ResourceType string          `json:"resourceType"`
}

func (s *Service) Log(ctx context.Context, input Input) error {
	if strings.TrimSpace(input.ActorUserID) == "" {
		return fmt.Errorf("actorUserId is required")
	}
	if strings.TrimSpace(input.ResourceType) == "" {
		return fmt.Errorf("resourceType is required")
	}
	if strings.TrimSpace(input.ResourceID) == "" {
		return fmt.Errorf("resourceId is required")
	}
	if strings.TrimSpace(input.Action) == "" {
		return fmt.Errorf("action is required")
	}
	if strings.TrimSpace(input.Status) == "" {
		return fmt.Errorf("status is required")
	}
	if s.events == nil || s.signer == nil {
		return fmt.Errorf("audit dependencies are not configured")
	}

	publicKey := strings.TrimSpace(input.PublicKey)
	keyAlgorithm := strings.TrimSpace(input.KeyAlgorithm)
	vaultPath := strings.TrimSpace(input.SignerVaultKey)
	if publicKey == "" || keyAlgorithm == "" || vaultPath == "" {
		if s.authUsers == nil {
			return fmt.Errorf("auth user repository is not configured")
		}
		authUser, err := s.authUsers.GetByID(ctx, input.ActorUserID)
		if err != nil {
			return fmt.Errorf("failed to resolve signing key for actor: %w", err)
		}
		if publicKey == "" {
			publicKey = authUser.PublicKey
		}
		if keyAlgorithm == "" {
			keyAlgorithm = authUser.KeyAlgorithm
		}
		if vaultPath == "" {
			vaultPath = authUser.VaultKeyPath
		}
	}
	if publicKey == "" || keyAlgorithm == "" || vaultPath == "" {
		return fmt.Errorf("signing key metadata is incomplete")
	}

	payloadBytes, err := json.Marshal(input.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	createdAt := s.now().UTC()
	envelopeBytes, err := json.Marshal(signedEnvelope{
		Action:       strings.TrimSpace(input.Action),
		ActorUserID:  strings.TrimSpace(input.ActorUserID),
		OccurredAt:   createdAt.Format(time.RFC3339Nano),
		Payload:      json.RawMessage(payloadBytes),
		ResourceID:   strings.TrimSpace(input.ResourceID),
		ResourceType: strings.TrimSpace(input.ResourceType),
	})
	if err != nil {
		return fmt.Errorf("failed to marshal signed event envelope: %w", err)
	}

	signature, err := s.signer.SignAccountEvent(ctx, vaultPath, envelopeBytes)
	if err != nil {
		return fmt.Errorf("failed to sign event log: %w", err)
	}

	contentHash := sha256.Sum256(envelopeBytes)
	return s.events.Save(ctx, domain.EventLog{
		EventID:       uuid.NewString(),
		ActorUserID:   strings.TrimSpace(input.ActorUserID),
		ResourceType:  strings.TrimSpace(input.ResourceType),
		ResourceID:    strings.TrimSpace(input.ResourceID),
		Action:        strings.TrimSpace(input.Action),
		Status:        strings.TrimSpace(input.Status),
		PayloadJSON:   string(payloadBytes),
		PublicKey:     publicKey,
		KeyAlgorithm:  keyAlgorithm,
		Signature:     signature,
		ContentSHA256: base64.StdEncoding.EncodeToString(contentHash[:]),
		CreatedAt:     createdAt,
	})
}
