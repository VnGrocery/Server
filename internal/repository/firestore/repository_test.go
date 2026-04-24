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
	if ProductFreshnessReportsCollection != "product_freshness_reports" {
		t.Fatalf("expected product_freshness_reports collection, got %s", ProductFreshnessReportsCollection)
	}
	if RefreshTokensCollection != "refresh_tokens" {
		t.Fatalf("expected refresh_tokens collection, got %s", RefreshTokensCollection)
	}
	if PasswordResetTokensCollection != "password_reset_tokens" {
		t.Fatalf("expected password_reset_tokens collection, got %s", PasswordResetTokensCollection)
	}
	if PledgesCollection != "pledges" {
		t.Fatalf("expected pledges collection, got %s", PledgesCollection)
	}
	if BuyerChecksCollection != "buyer_checks" {
		t.Fatalf("expected buyer_checks collection, got %s", BuyerChecksCollection)
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
		ProductID:       "product-1",
		CreatedByUserID: "user-1",
		Status:          "pending",
		ImageHash:       "hash-1",
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
		ProductID:      "product-1",
		ShopID:         "shop-1",
		OwnerUserID:    "user-1",
		Name:           "Apple",
		Category:       "fruit",
		Tags:           []string{"fresh"},
		ImageURLs:      []string{"https://example.com/apple.jpg"},
		FreshnessNote:  "Fresh today",
		FreshnessScore: 8.5,
		Price:          10,
		Currency:       "VND",
		Status:         "active",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	productFreshnessReport := domain.ProductFreshnessReport{
		ReportID:       "report-1",
		ProductID:      "product-1",
		ShopID:         "shop-1",
		ReporterUserID: "user-2",
		Status:         "active",
		ImageHash:      "hash",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	refreshToken := domain.RefreshToken{
		TokenID:   "refresh-1",
		UserID:    "user-1",
		TokenHash: "hash",
		Status:    "active",
		ExpiresAt: now,
		CreatedAt: now,
		UpdatedAt: now,
	}
	passwordResetToken := domain.PasswordResetToken{
		TokenID:   "reset-1",
		UserID:    "user-1",
		TokenHash: "hash",
		Status:    "active",
		ExpiresAt: now,
		CreatedAt: now,
		UpdatedAt: now,
	}

	assertFirestoreTag(t, user, "UserID", "userId")
	assertFirestoreTag(t, shop, "OwnerUserID", "ownerUserId")
	assertFirestoreTag(t, shop, "Latitude", "latitude")
	assertFirestoreTag(t, shop, "Longitude", "longitude")
	assertFirestoreTag(t, shop, "DataHash", "dataHash")
	assertFirestoreTag(t, shop, "ChainTxHash", "chainTxHash")
	assertFirestoreTag(t, shop, "ChainBlockNumber", "chainBlockNumber")
	assertFirestoreTag(t, shop, "ChainAnchorStatus", "chainAnchorStatus")
	assertFirestoreTag(t, shop, "ChainAnchorTime", "chainAnchorTime")
	assertFirestoreTag(t, shop, "IntegrityStatus", "integrityStatus")
	assertFirestoreTag(t, pledge, "CreatedByUserID", "createdByUserId")
	assertFirestoreTag(t, pledge, "ProductID", "productId")
	assertFirestoreTag(t, pledge, "Score", "score")
	assertFirestoreTag(t, pledge, "Category", "category")
	assertFirestoreTag(t, pledge, "Confidence", "confidence")
	assertFirestoreTag(t, pledge, "DataHash", "dataHash")
	assertFirestoreTag(t, pledge, "ChainTxHash", "chainTxHash")
	assertFirestoreTag(t, pledge, "ChainBlockNumber", "chainBlockNumber")
	assertFirestoreTag(t, pledge, "ChainAnchorStatus", "chainAnchorStatus")
	assertFirestoreTag(t, pledge, "ChainAnchorTime", "chainAnchorTime")
	assertFirestoreTag(t, pledge, "IntegrityStatus", "integrityStatus")
	assertFirestoreTag(t, review, "ReviewerUserID", "reviewerUserId")
	assertFirestoreTag(t, review, "Rating", "rating")
	assertFirestoreTag(t, product, "ProductID", "productId")
	assertFirestoreTag(t, product, "OwnerUserID", "ownerUserId")
	assertFirestoreTag(t, product, "Category", "category")
	assertFirestoreTag(t, product, "Tags", "tags")
	assertFirestoreTag(t, product, "ImageURLs", "imageUrls")
	assertFirestoreTag(t, product, "FreshnessNote", "freshnessNote")
	assertFirestoreTag(t, product, "FreshnessScore", "freshnessScore")
	assertFirestoreTag(t, product, "Currency", "currency")
	assertFirestoreTag(t, productFreshnessReport, "ReportID", "reportId")
	assertFirestoreTag(t, productFreshnessReport, "ReporterUserID", "reporterUserId")
	assertFirestoreTag(t, productFreshnessReport, "ImageHash", "imageHash")
	assertFirestoreTag(t, refreshToken, "TokenHash", "tokenHash")
	assertFirestoreTag(t, refreshToken, "ExpiresAt", "expiresAt")
	assertFirestoreTag(t, passwordResetToken, "TokenHash", "tokenHash")
	assertFirestoreTag(t, passwordResetToken, "UsedAt", "usedAt")
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
