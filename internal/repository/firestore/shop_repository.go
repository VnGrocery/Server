package firestore

import (
	"context"
	"fmt"

	gofirestore "cloud.google.com/go/firestore"

	"vngrocery/internal/domain"
)

type ShopRepository struct {
	client *gofirestore.Client
}

func NewShopRepository(client *gofirestore.Client) *ShopRepository {
	return &ShopRepository{client: client}
}

func (r *ShopRepository) Save(ctx context.Context, shop domain.Shop) error {
	_, err := r.client.Collection(ShopsCollection).Doc(shop.ShopID).Set(ctx, shop)
	if err != nil {
		return fmt.Errorf("failed to save shop: %w", err)
	}

	return nil
}

func (r *ShopRepository) GetByID(ctx context.Context, shopID string) (domain.Shop, error) {
	doc, err := r.client.Collection(ShopsCollection).Doc(shopID).Get(ctx)
	if err != nil {
		return domain.Shop{}, fmt.Errorf("failed to get shop: %w", err)
	}

	var shop domain.Shop
	if err := doc.DataTo(&shop); err != nil {
		return domain.Shop{}, fmt.Errorf("failed to decode shop document: %w", err)
	}

	return shop, nil
}

func (r *ShopRepository) ListActive(ctx context.Context) ([]domain.Shop, error) {
	docs, err := r.client.Collection(ShopsCollection).Where("status", "==", "active").Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list shops: %w", err)
	}

	shops := make([]domain.Shop, 0, len(docs))
	for _, doc := range docs {
		var shop domain.Shop
		if err := doc.DataTo(&shop); err != nil {
			return nil, fmt.Errorf("failed to decode shop document: %w", err)
		}
		shops = append(shops, shop)
	}

	return shops, nil
}
