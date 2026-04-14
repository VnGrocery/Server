package cache

import (
	"context"
	"fmt"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	cachepkg "vngrocery/pkg/cache"
)

const eventLogNamespace = "event_logs"

type EventLogRepository struct {
	base  repository.EventLogRepository
	cache *cachepkg.Store
}

func NewEventLogRepository(base repository.EventLogRepository, cache *cachepkg.Store) *EventLogRepository {
	return &EventLogRepository{base: base, cache: cache}
}

func (r *EventLogRepository) Save(ctx context.Context, event domain.EventLog) error {
	if err := r.base.Save(ctx, event); err != nil {
		return err
	}
	if !r.cacheEnabled() {
		return nil
	}
	if err := r.cache.BumpVersion(ctx, eventLogNamespace); err != nil {
		return fmt.Errorf("saved event log but failed to invalidate cache: %w", err)
	}
	return nil
}

func (r *EventLogRepository) GetByID(ctx context.Context, eventID string) (domain.EventLog, error) {
	version, key, err := r.keyWithVersion(ctx, "id:"+eventID)
	if err != nil {
		return domain.EventLog{}, err
	}
	var cached domain.EventLog
	if hit, err := r.cache.GetJSON(ctx, key, &cached); err != nil {
		return domain.EventLog{}, err
	} else if hit {
		return cached, nil
	}

	item, err := r.base.GetByID(ctx, eventID)
	if err != nil {
		return domain.EventLog{}, err
	}
	if !r.cacheEnabled() {
		return item, nil
	}
	if err := r.cache.SetJSON(ctx, r.cache.BuildKey(eventLogNamespace, version, "id:"+eventID), item); err != nil {
		return domain.EventLog{}, err
	}
	return item, nil
}

func (r *EventLogRepository) GetLatestByResource(ctx context.Context, resourceType, resourceID string) (domain.EventLog, error) {
	suffix := "latest:" + resourceType + ":" + resourceID
	version, key, err := r.keyWithVersion(ctx, suffix)
	if err != nil {
		return domain.EventLog{}, err
	}
	var cached domain.EventLog
	if hit, err := r.cache.GetJSON(ctx, key, &cached); err != nil {
		return domain.EventLog{}, err
	} else if hit {
		return cached, nil
	}

	item, err := r.base.GetLatestByResource(ctx, resourceType, resourceID)
	if err != nil {
		return domain.EventLog{}, err
	}
	if !r.cacheEnabled() {
		return item, nil
	}
	if err := r.cache.SetJSON(ctx, r.cache.BuildKey(eventLogNamespace, version, suffix), item); err != nil {
		return domain.EventLog{}, err
	}
	return item, nil
}

func (r *EventLogRepository) List(ctx context.Context, filter repository.EventLogListFilter) ([]domain.EventLog, error) {
	suffix, err := buildSuffix(filter)
	if err != nil {
		return nil, err
	}
	version, key, err := r.keyWithVersion(ctx, "list:"+suffix)
	if err != nil {
		return nil, err
	}
	var cached []domain.EventLog
	if hit, err := r.cache.GetJSON(ctx, key, &cached); err != nil {
		return nil, err
	} else if hit {
		return cached, nil
	}

	items, err := r.base.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	if !r.cacheEnabled() {
		return items, nil
	}
	if err := r.cache.SetJSON(ctx, r.cache.BuildKey(eventLogNamespace, version, "list:"+suffix), items); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *EventLogRepository) keyWithVersion(ctx context.Context, suffix string) (int64, string, error) {
	if !r.cacheEnabled() {
		return 0, "", nil
	}
	version, err := r.cache.Version(ctx, eventLogNamespace)
	if err != nil {
		return 0, "", err
	}
	return version, r.cache.BuildKey(eventLogNamespace, version, suffix), nil
}

func (r *EventLogRepository) cacheEnabled() bool {
	return r.cache != nil && r.cache.IsEnabled()
}
