package domain

import "time"

type ProductBatch struct {
	BatchID          string     `firestore:"batchId"`
	ProductID        string     `firestore:"productId"`
	ShopID           string     `firestore:"shopId"`
	OwnerUserID      string     `firestore:"ownerUserId"`
	BatchCode        string     `firestore:"batchCode"`
	OriginName       string     `firestore:"originName"`
	OriginAddress    string     `firestore:"originAddress"`
	SupplierName     string     `firestore:"supplierName"`
	SlaughteredAt    *time.Time `firestore:"slaughteredAt"`
	PackedAt         *time.Time `firestore:"packedAt"`
	ReceivedAt       *time.Time `firestore:"receivedAt"`
	ExpiresAt        *time.Time `firestore:"expiresAt"`
	Quantity         float64    `firestore:"quantity"`
	QuantityUnit     string     `firestore:"quantityUnit"`
	StorageTempMin   float64    `firestore:"storageTempMin"`
	StorageTempMax   float64    `firestore:"storageTempMax"`
	CurrentFreshness float64    `firestore:"currentFreshness"`
	CurrentCategory  string     `firestore:"currentCategory"`
	Status           string     `firestore:"status"`
	Version          int        `firestore:"version"`
	CreatedAt        time.Time  `firestore:"createdAt"`
	UpdatedAt        time.Time  `firestore:"updatedAt"`
}
