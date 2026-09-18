package domain

import "time"

type ProductFreshnessReport struct {
	ReportID       string `firestore:"reportId"`
	ProductID      string `firestore:"productId"`
	ShopID         string `firestore:"shopId"`
	ReporterUserID string `firestore:"reporterUserId"`
	Status         string `firestore:"status"`

	// Who produced Score. Recorded from the first report rather than left for
	// later: once AI scoring exists it becomes another value here, and the rows
	// written before it stay readable instead of being an unlabelled backlog a
	// migration has to guess at.
	ReviewStatus string `firestore:"reviewStatus"`

	Version           int        `firestore:"version"`
	Score             float64    `firestore:"score"`
	Category          string     `firestore:"category"`
	Confidence        float64    `firestore:"confidence"`
	Comment           string     `firestore:"comment"`
	ImageHash         string     `firestore:"imageHash"`
	ImageCID          string     `firestore:"imageCid"`
	ModeratedByUserID string     `firestore:"moderatedByUserId"`
	ModerationNote    string     `firestore:"moderationNote"`
	ModeratedAt       *time.Time `firestore:"moderatedAt"`
	CreatedAt         time.Time  `firestore:"createdAt"`
	UpdatedAt         time.Time  `firestore:"updatedAt"`
}
