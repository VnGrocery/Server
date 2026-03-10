package firestore

import (
	"reflect"
	"testing"
	"time"

	"vngrocery/internal/domain"
)

func TestCollectionNames(t *testing.T) {
	if UsersCollection != "users" {
		t.Fatalf("expected users collection, got %s", UsersCollection)
	}
	if ShopsCollection != "shops" {
		t.Fatalf("expected shops collection, got %s", ShopsCollection)
	}
	if ProductsCollection != "products" {
		t.Fatalf("expected products collection, got %s", ProductsCollection)
	}
	if PledgesCollection != "pledges" {
		t.Fatalf("expected pledges collection, got %s", PledgesCollection)
	}
	if ShopReviewsCollection != "shop_reviews" {
		t.Fatalf("expected shop_reviews collection, got %s", ShopReviewsCollection)
	}
}

func TestDomainStructTags(t *testing.T) {
	now := time.Now()

	user := domain.User{
		UserID:      "user-1",
		Email:       "user@example.com",
		DisplayName: "User 1",
		Role:        "seller",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	shop := domain.Shop{
		ShopID:      "shop-1",
		OwnerUserID: "user-1",
		Name:        "Fresh Market",
		Description: "A nice store",
		Address:     "123 Main St",
		Latitude:    10.123,
		Longitude:   106.456,
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	pledge := domain.Pledge{
		PledgeID:        "pledge-1",
		ShopID:          "shop-1",
		CreatedByUserID: "user-1",
		Status:          "pending",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	review := domain.ShopReview{
		ReviewID:       "review-1",
		ShopID:         "shop-1",
		ReviewerUserID: "user-2",
		Rating:         5,
		Comment:        "Great",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	product := domain.Product{
		ProductID:   "product-1",
		ShopID:      "shop-1",
		OwnerUserID: "user-1",
		Name:        "Apple",
		Price:       10,
		Currency:    "VND",
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	assertFirestoreTag(t, user, "UserID", "userId")
	assertFirestoreTag(t, shop, "OwnerUserID", "ownerUserId")
	assertFirestoreTag(t, shop, "Latitude", "latitude")
	assertFirestoreTag(t, shop, "Longitude", "longitude")
	assertFirestoreTag(t, pledge, "CreatedByUserID", "createdByUserId")
	assertFirestoreTag(t, pledge, "Score", "score")
	assertFirestoreTag(t, pledge, "Category", "category")
	assertFirestoreTag(t, pledge, "Confidence", "confidence")
	assertFirestoreTag(t, review, "ReviewerUserID", "reviewerUserId")
	assertFirestoreTag(t, review, "Rating", "rating")
	assertFirestoreTag(t, product, "ProductID", "productId")
	assertFirestoreTag(t, product, "OwnerUserID", "ownerUserId")
	assertFirestoreTag(t, product, "Currency", "currency")
}

func assertFirestoreTag(t *testing.T, value any, fieldName, expected string) {
	t.Helper()

	field, ok := reflect.TypeOf(value).FieldByName(fieldName)
	if !ok {
		t.Fatalf("field %s not found", fieldName)
	}

	if actual := field.Tag.Get("firestore"); actual != expected {
		t.Fatalf("expected firestore tag %s, got %s", expected, actual)
	}
}
