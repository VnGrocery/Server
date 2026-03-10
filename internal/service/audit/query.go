package audit

import (
	"context"
	"fmt"
	"strings"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
)

type ListInput struct {
	ResourceType string
	ResourceID   string
	ActorUserID  string
}

func (s *Service) List(ctx context.Context, input ListInput) ([]domain.EventLog, error) {
	if s.events == nil {
		return nil, fmt.Errorf("event log repository is not configured")
	}
	if strings.TrimSpace(input.ResourceID) != "" && strings.TrimSpace(input.ResourceType) == "" {
		return nil, fmt.Errorf("resourceType is required when resourceId is provided")
	}

	return s.events.List(ctx, repository.EventLogListFilter{
		ResourceType: strings.TrimSpace(input.ResourceType),
		ResourceID:   strings.TrimSpace(input.ResourceID),
		ActorUserID:  strings.TrimSpace(input.ActorUserID),
	})
}
