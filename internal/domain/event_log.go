package domain

import "time"

type EventLog struct {
	EventID       string    `firestore:"eventId"`
	ActorUserID   string    `firestore:"actorUserId"`
	ResourceType  string    `firestore:"resourceType"`
	ResourceID    string    `firestore:"resourceId"`
	Action        string    `firestore:"action"`
	Status        string    `firestore:"status"`
	PayloadJSON   string    `firestore:"payloadJson"`
	PublicKey     string    `firestore:"publicKey"`
	KeyAlgorithm  string    `firestore:"keyAlgorithm"`
	Signature     string    `firestore:"signature"`
	ContentSHA256 string    `firestore:"contentSha256"`
	CreatedAt     time.Time `firestore:"createdAt"`
}
