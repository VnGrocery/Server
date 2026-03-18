package domain

import "time"

type ProductFreshnessReport struct {
	ReportID       string    `firestore:"reportId"`
	ProductID      string    `firestore:"productId"`
	ShopID         string    `firestore:"shopId"`
	ReporterUserID string    `firestore:"reporterUserId"`
	Status         string    `firestore:"status"`
	Version        int       `firestore:"version"`
	Score          float64   `firestore:"score"`
	Category       string    `firestore:"category"`
	Confidence     float64   `firestore:"confidence"`
	Comment        string    `firestore:"comment"`
	ImageHash      string    `firestore:"imageHash"`
	CreatedAt      time.Time `firestore:"createdAt"`
	UpdatedAt      time.Time `firestore:"updatedAt"`
}
