package firestore

import (
	"context"
	"fmt"

	gofirestore "cloud.google.com/go/firestore"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
)

type ProductRepository struct {
	client *gofirestore.Client
}

func NewProductRepository(client *gofirestore.Client) *ProductRepository {
	return &ProductRepository{client: client}
}

func (r *ProductRepository) Save(ctx context.Context, product domain.Product) error {
	_, err := r.client.Collection(ProductsCollection).Doc(product.ProductID).Set(ctx, product)
	if err != nil {
		return fmt.Errorf("failed to save product: %w", err)
	}
	return nil
}

func (r *ProductRepository) GetByID(ctx context.Context, productID string) (domain.Product, error) {
	doc, err := r.client.Collection(ProductsCollection).Doc(productID).Get(ctx)
	if err != nil {
		return domain.Product{}, fmt.Errorf("failed to get product: %w", err)
	}

	var product domain.Product
	if err := doc.DataTo(&product); err != nil {
		return domain.Product{}, fmt.Errorf("failed to decode product document: %w", err)
	}
	return product, nil
}

func (r *ProductRepository) List(ctx context.Context, filter repository.ProductListFilter) ([]domain.Product, error) {
	query := r.client.Collection(ProductsCollection).Query
	if filter.ShopID != "" {
		query = query.Where("shopId", "==", filter.ShopID)
	}
	if filter.Status != "" {
		query = query.Where("status", "==", filter.Status)
	}
	if filter.OwnerUserID != "" {
		query = query.Where("ownerUserId", "==", filter.OwnerUserID)
	}

	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to list products: %w", err)
	}

	products := make([]domain.Product, 0, len(docs))
	for _, doc := range docs {
		var product domain.Product
		if err := doc.DataTo(&product); err != nil {
			return nil, fmt.Errorf("failed to decode product document: %w", err)
		}
		products = append(products, product)
	}

	return products, nil
}
