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
	if PledgesCollection != "pledges" {
		t.Fatalf("expected pledges collection, got %s", PledgesCollection)
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

	assertFirestoreTag(t, user, "UserID", "userId")
	assertFirestoreTag(t, shop, "OwnerUserID", "ownerUserId")
	assertFirestoreTag(t, pledge, "CreatedByUserID", "createdByUserId")
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
