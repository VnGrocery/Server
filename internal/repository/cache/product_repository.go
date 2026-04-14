package cache

import (
	"context"
	"fmt"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	cachepkg "vngrocery/pkg/cache"
)

const productNamespace = "products"

type ProductRepository struct {
	base  repository.ProductRepository
	cache *cachepkg.Store
}

func NewProductRepository(base repository.ProductRepository, cache *cachepkg.Store) *ProductRepository {
	return &ProductRepository{base: base, cache: cache}
}

func (r *ProductRepository) Save(ctx context.Context, product domain.Product) error {
	if err := r.base.Save(ctx, product); err != nil {
		return err
	}
	if !r.cacheEnabled() {
		return nil
	}
	if err := r.cache.BumpVersion(ctx, productNamespace); err != nil {
		return fmt.Errorf("saved product but failed to invalidate cache: %w", err)
	}
	return nil
}

func (r *ProductRepository) GetByID(ctx context.Context, productID string) (domain.Product, error) {
	version, key, err := r.keyWithVersion(ctx, "id:"+productID)
	if err != nil {
		return domain.Product{}, err
	}
	var cached domain.Product
	if hit, err := r.cache.GetJSON(ctx, key, &cached); err != nil {
		return domain.Product{}, err
	} else if hit {
		return cached, nil
	}

	item, err := r.base.GetByID(ctx, productID)
	if err != nil {
		return domain.Product{}, err
	}
	if !r.cacheEnabled() {
		return item, nil
	}
	if err := r.cache.SetJSON(ctx, r.cache.BuildKey(productNamespace, version, "id:"+productID), item); err != nil {
		return domain.Product{}, err
	}
	return item, nil
}

func (r *ProductRepository) List(ctx context.Context, filter repository.ProductListFilter) ([]domain.Product, error) {
	suffix, err := buildSuffix(filter)
	if err != nil {
		return nil, err
	}
	version, key, err := r.keyWithVersion(ctx, "list:"+suffix)
	if err != nil {
		return nil, err
	}
	var cached []domain.Product
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
	if err := r.cache.SetJSON(ctx, r.cache.BuildKey(productNamespace, version, "list:"+suffix), items); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *ProductRepository) keyWithVersion(ctx context.Context, suffix string) (int64, string, error) {
	if !r.cacheEnabled() {
		return 0, "", nil
	}
	version, err := r.cache.Version(ctx, productNamespace)
	if err != nil {
		return 0, "", err
	}
	return version, r.cache.BuildKey(productNamespace, version, suffix), nil
}

func (r *ProductRepository) cacheEnabled() bool {
	return r.cache != nil && r.cache.IsEnabled()
}
