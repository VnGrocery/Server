package audit

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
)

type ListInput struct {
	ResourceType  string
	ResourceID    string
	ActorUserID   string
	Action        string
	Status        string
	MinSequence   int
	MaxSequence   int
	CreatedAfter  time.Time
	CreatedBefore time.Time
	Page          int
	PageSize      int
}

type ListResult struct {
	Items    []domain.EventLog
	Total    int
	Page     int
	PageSize int
}

type VerifyEventInput struct {
	EventID string
}

type VerifyResourceInput struct {
	ResourceType string
	ResourceID   string
}

type EventVerificationResult struct {
	EventID              string
	ResourceType         string
	ResourceID           string
	Sequence             int
	PreviousEventID      string
	ContentHashValid     bool
	SignatureValid       bool
	ChainLinkValid       bool
	PreviousEventPresent bool
	Verified             bool
}

type VerifyResourceResult struct {
	ResourceType string
	ResourceID   string
	EventCount   int
	Verified     bool
	Events       []EventVerificationResult
}

func (s *Service) List(ctx context.Context, input ListInput) (ListResult, error) {
	if s.events == nil {
		return ListResult{}, fmt.Errorf("event log repository is not configured")
	}
	if strings.TrimSpace(input.ResourceID) != "" && strings.TrimSpace(input.ResourceType) == "" {
		return ListResult{}, fmt.Errorf("resourceType is required when resourceId is provided")
	}
	if input.MinSequence > 0 && input.MaxSequence > 0 && input.MinSequence > input.MaxSequence {
		return ListResult{}, fmt.Errorf("minSequence must be less than or equal to maxSequence")
	}
	if !input.CreatedAfter.IsZero() && !input.CreatedBefore.IsZero() && input.CreatedAfter.After(input.CreatedBefore) {
		return ListResult{}, fmt.Errorf("createdAfter must be before or equal to createdBefore")
	}

	page := input.Page
	if page <= 0 {
		page = 1
	}
	pageSize := input.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}

	items, err := s.events.List(ctx, repository.EventLogListFilter{
		ResourceType:  strings.TrimSpace(input.ResourceType),
		ResourceID:    strings.TrimSpace(input.ResourceID),
		ActorUserID:   strings.TrimSpace(input.ActorUserID),
		Action:        strings.TrimSpace(input.Action),
		Status:        strings.TrimSpace(input.Status),
		MinSequence:   input.MinSequence,
		MaxSequence:   input.MaxSequence,
		CreatedAfter:  input.CreatedAfter,
		CreatedBefore: input.CreatedBefore,
	})
	if err != nil {
		return ListResult{}, err
	}

	total := len(items)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	return ListResult{
		Items:    items[start:end],
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Service) VerifyEvent(ctx context.Context, input VerifyEventInput) (EventVerificationResult, error) {
	if s.events == nil {
		return EventVerificationResult{}, fmt.Errorf("event log repository is not configured")
	}
	eventID := strings.TrimSpace(input.EventID)
	if eventID == "" {
		return EventVerificationResult{}, fmt.Errorf("eventId is required")
	}

	event, err := s.events.GetByID(ctx, eventID)
	if err != nil {
		return EventVerificationResult{}, err
	}

	var previous *domain.EventLog
	if strings.TrimSpace(event.PreviousEventID) != "" {
		prev, err := s.events.GetByID(ctx, event.PreviousEventID)
		if err == nil {
			previous = &prev
		}
	}

	return verifyEvent(event, previous)
}

func (s *Service) VerifyResource(ctx context.Context, input VerifyResourceInput) (VerifyResourceResult, error) {
	if s.events == nil {
		return VerifyResourceResult{}, fmt.Errorf("event log repository is not configured")
	}

	resourceType := strings.TrimSpace(input.ResourceType)
	resourceID := strings.TrimSpace(input.ResourceID)
	if resourceType == "" {
		return VerifyResourceResult{}, fmt.Errorf("resourceType is required")
	}
	if resourceID == "" {
		return VerifyResourceResult{}, fmt.Errorf("resourceId is required")
	}

	events, err := s.events.List(ctx, repository.EventLogListFilter{
		ResourceType: resourceType,
		ResourceID:   resourceID,
	})
	if err != nil {
		return VerifyResourceResult{}, err
	}

	sort.Slice(events, func(i, j int) bool {
		if events[i].Sequence == events[j].Sequence {
			return events[i].CreatedAt.Before(events[j].CreatedAt)
		}
		return events[i].Sequence < events[j].Sequence
	})

	results := make([]EventVerificationResult, 0, len(events))
	allValid := true
	for i := range events {
		var previous *domain.EventLog
		if i > 0 {
			previous = &events[i-1]
		}
		result, err := verifyEvent(events[i], previous)
		if err != nil {
			return VerifyResourceResult{}, err
		}
		if !result.Verified {
			allValid = false
		}
		results = append(results, result)
	}

	return VerifyResourceResult{
		ResourceType: resourceType,
		ResourceID:   resourceID,
		EventCount:   len(events),
		Verified:     allValid,
		Events:       results,
	}, nil
}

func verifyEvent(event domain.EventLog, previous *domain.EventLog) (EventVerificationResult, error) {
	envelopeBytes, err := signedEnvelopeBytes(event)
	if err != nil {
		return EventVerificationResult{}, err
	}

	contentHash := sha256.Sum256(envelopeBytes)
	contentHashValid := base64.StdEncoding.EncodeToString(contentHash[:]) == strings.TrimSpace(event.ContentSHA256)

	signatureValid := false
	if strings.EqualFold(strings.TrimSpace(event.KeyAlgorithm), "Ed25519") {
		publicKey, err := base64.StdEncoding.DecodeString(strings.TrimSpace(event.PublicKey))
		if err == nil {
			signature, err := base64.StdEncoding.DecodeString(strings.TrimSpace(event.Signature))
			if err == nil {
				signatureValid = ed25519.Verify(ed25519.PublicKey(publicKey), envelopeBytes, signature)
			}
		}
	}

	previousPresent := previous != nil
	chainLinkValid := false
	switch {
	case event.Sequence == 1 && strings.TrimSpace(event.PreviousEventID) == "":
		chainLinkValid = true
	case previous != nil:
		chainLinkValid = previous.EventID == strings.TrimSpace(event.PreviousEventID) &&
			previous.ResourceType == event.ResourceType &&
			previous.ResourceID == event.ResourceID &&
			previous.Sequence+1 == event.Sequence
	}

	return EventVerificationResult{
		EventID:              event.EventID,
		ResourceType:         event.ResourceType,
		ResourceID:           event.ResourceID,
		Sequence:             event.Sequence,
		PreviousEventID:      event.PreviousEventID,
		ContentHashValid:     contentHashValid,
		SignatureValid:       signatureValid,
		ChainLinkValid:       chainLinkValid,
		PreviousEventPresent: previousPresent,
		Verified:             contentHashValid && signatureValid && chainLinkValid,
	}, nil
}

func signedEnvelopeBytes(event domain.EventLog) ([]byte, error) {
	payload := json.RawMessage(strings.TrimSpace(event.PayloadJSON))
	if len(payload) == 0 {
		payload = json.RawMessage([]byte("null"))
	}
	occurredAt := strings.TrimSpace(event.OccurredAt)
	if occurredAt == "" {
		occurredAt = event.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	return json.Marshal(signedEnvelope{
		Action:          strings.TrimSpace(event.Action),
		ActorUserID:     strings.TrimSpace(event.ActorUserID),
		OccurredAt:      occurredAt,
		Payload:         payload,
		Reason:          strings.TrimSpace(event.Reason),
		ResourceID:      strings.TrimSpace(event.ResourceID),
		ResourceType:    strings.TrimSpace(event.ResourceType),
		ResourceVersion: event.ResourceVersion,
		Sequence:        event.Sequence,
		PreviousEventID: strings.TrimSpace(event.PreviousEventID),
	})
}
