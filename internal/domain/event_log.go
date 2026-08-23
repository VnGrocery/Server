package domain

import "time"

type EventLog struct {
	EventID         string `firestore:"eventId"`
	ActorUserID     string `firestore:"actorUserId"`
	ResourceType    string `firestore:"resourceType"`
	ResourceID      string `firestore:"resourceId"`
	ResourceVersion int    `firestore:"resourceVersion"`
	Action          string `firestore:"action"`
	Status          string `firestore:"status"`
	Sequence        int    `firestore:"sequence"`
	PreviousEventID string `firestore:"previousEventId"`
	OccurredAt      string `firestore:"occurredAt"`

	// The actor's own words for why this change was made. Signed with the rest
	// of the event; empty on events written before the field existed.
	Reason        string    `firestore:"reason"`
	PayloadJSON   string    `firestore:"payloadJson"`
	PublicKey     string    `firestore:"publicKey"`
	KeyAlgorithm  string    `firestore:"keyAlgorithm"`
	Signature     string    `firestore:"signature"`
	ContentSHA256 string    `firestore:"contentSha256"`
	CreatedAt     time.Time `firestore:"createdAt"`
}
