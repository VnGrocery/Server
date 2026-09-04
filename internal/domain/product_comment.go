package domain

import "time"

// ProductComment is what a buyer wrote about a product after checking it at the
// stall.
//
// CheckID is the buyer check that entitled the comment. It is kept on the
// record rather than looked up later so the comment can still say "this person
// stood in front of the goods" even if the check is archived, and so the trust
// score never has to guess which comments are backed by evidence.
type ProductComment struct {
	CommentID    string `firestore:"commentId"`
	ShopID       string `firestore:"shopId"`
	ProductID    string `firestore:"productId"`
	AuthorUserID string `firestore:"authorUserId"`
	Body         string `firestore:"body"`
	Status       string `firestore:"status"`

	CheckID string `firestore:"checkId"`
	Verdict string `firestore:"verdict"`

	ModeratedByUserID string     `firestore:"moderatedByUserId"`
	ModerationReason  string     `firestore:"moderationReason"`
	ModeratedAt       *time.Time `firestore:"moderatedAt"`

	// The shop's public reply, one per comment. Rewriting it replaces the old
	// text rather than stacking a thread - the same one-slot rule the comment
	// itself already follows for its author.
	ShopReplyBody string     `firestore:"shopReplyBody"`
	ShopRepliedAt *time.Time `firestore:"shopRepliedAt"`

	Version   int       `firestore:"version"`
	CreatedAt time.Time `firestore:"createdAt"`
	UpdatedAt time.Time `firestore:"updatedAt"`
}
