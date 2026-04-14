package cache

import (
	"context"
	"encoding/json"
	"fmt"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	cachepkg "vngrocery/pkg/cache"
)

const shopNamespace = "shops"

type ShopRepository struct {
	base  repository.ShopRepository
	cache *cachepkg.Store
}

func NewShopRepository(base repository.ShopRepository, cache *cachepkg.Store) *ShopRepository {
	return &ShopRepository{base: base, cache: cache}
}

func (r *ShopRepository) Save(ctx context.Context, shop domain.Shop) error {
	if err := r.base.Save(ctx, shop); err != nil {
		return err
	}
	if !r.cacheEnabled() {
		return nil
	}
	if err := r.cache.BumpVersion(ctx, shopNamespace); err != nil {
		return fmt.Errorf("saved shop but failed to invalidate cache: %w", err)
	}
	return nil
}

func (r *ShopRepository) GetByID(ctx context.Context, shopID string) (domain.Shop, error) {
	if cachepkg.ShouldBypass(ctx) {
		return r.base.GetByID(ctx, shopID)
	}
	version, key, err := r.keyWithVersion(ctx, "id:"+shopID)
	if err != nil {
		return domain.Shop{}, err
	}
	var cached domain.Shop
	if hit, err := r.cache.GetJSON(ctx, key, &cached); err != nil {
		return domain.Shop{}, err
	} else if hit {
		return cached, nil
	}

	shop, err := r.base.GetByID(ctx, shopID)
	if err != nil {
		return domain.Shop{}, err
	}
	if !r.cacheEnabled() {
		return shop, nil
	}
	if err := r.cache.SetJSON(ctx, r.cache.BuildKey(shopNamespace, version, "id:"+shopID), shop); err != nil {
		return domain.Shop{}, err
	}
	return shop, nil
}

func (r *ShopRepository) List(ctx context.Context, filter repository.ShopListFilter) ([]domain.Shop, error) {
	if cachepkg.ShouldBypass(ctx) {
		return r.base.List(ctx, filter)
	}
	suffix, err := buildSuffix(filter)
	if err != nil {
		return nil, err
	}
	version, key, err := r.keyWithVersion(ctx, "list:"+suffix)
	if err != nil {
		return nil, err
	}
	var cached []domain.Shop
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
	if err := r.cache.SetJSON(ctx, r.cache.BuildKey(shopNamespace, version, "list:"+suffix), items); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *ShopRepository) keyWithVersion(ctx context.Context, suffix string) (int64, string, error) {
	if !r.cacheEnabled() {
		return 0, "", nil
	}
	version, err := r.cache.Version(ctx, shopNamespace)
	if err != nil {
		return 0, "", err
	}
	return version, r.cache.BuildKey(shopNamespace, version, suffix), nil
}

func (r *ShopRepository) cacheEnabled() bool {
	return r.cache != nil && r.cache.IsEnabled()
}

func buildSuffix(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("marshal cache key suffix: %w", err)
	}
	return string(b), nil
}
