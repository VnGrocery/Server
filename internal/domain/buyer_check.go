package domain

import "time"

type BuyerCheck struct {
	CheckID           string     `firestore:"checkId"`
	ShopID            string     `firestore:"shopId"`
	ProductID         string     `firestore:"productId"`
	PledgeID          string     `firestore:"pledgeId"`
	BuyerUserID       string     `firestore:"buyerUserId"`
	Status            string     `firestore:"status"`
	Version           int        `firestore:"version"`
	PolicyVersion     string     `firestore:"policyVersion"`
	Trusted           bool       `firestore:"trusted"`
	Verdict           string     `firestore:"verdict"`
	PledgedScore      float64    `firestore:"pledgedScore"`
	ActualScore       float64    `firestore:"actualScore"`
	ScoreDelta        float64    `firestore:"scoreDelta"`
	ScoreDeltaAbs     float64    `firestore:"scoreDeltaAbs"`
	PledgedCategory   string     `firestore:"pledgedCategory"`
	ActualCategory    string     `firestore:"actualCategory"`
	ActualConfidence  float64    `firestore:"actualConfidence"`
	CategoryMatch     bool       `firestore:"categoryMatch"`
	ImageHash         string     `firestore:"imageHash"`
	Reasons           []string   `firestore:"reasons"`
	ModeratedByUserID string     `firestore:"moderatedByUserId"`
	ModerationNote    string     `firestore:"moderationNote"`
	ModeratedAt       *time.Time `firestore:"moderatedAt"`
	CreatedAt         time.Time  `firestore:"createdAt"`
	UpdatedAt         time.Time  `firestore:"updatedAt"`
}
