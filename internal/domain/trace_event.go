package domain

import "time"

type TraceEvent struct {
	EventID      string    `firestore:"eventId"`
	BatchID      string    `firestore:"batchId"`
	ProductID    string    `firestore:"productId"`
	ShopID       string    `firestore:"shopId"`
	ActorUserID  string    `firestore:"actorUserId"`
	Type         string    `firestore:"type"`
	Title        string    `firestore:"title"`
	Description  string    `firestore:"description"`
	LocationName string    `firestore:"locationName"`
	Latitude     float64   `firestore:"latitude"`
	Longitude    float64   `firestore:"longitude"`
	Temperature  *float64  `firestore:"temperature"`
	Humidity     *float64  `firestore:"humidity"`
	ImageCID     string    `firestore:"imageCid"`
	ImageHash    string    `firestore:"imageHash"`
	DataHash     string    `firestore:"dataHash"`
	Status       string    `firestore:"status"`
	OccurredAt   time.Time `firestore:"occurredAt"`
	CreatedAt    time.Time `firestore:"createdAt"`
}
