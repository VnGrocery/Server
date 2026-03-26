package audit

import (
	"context"
	"fmt"
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
