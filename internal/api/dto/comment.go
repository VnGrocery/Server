package dto

import "time"

type CreateProductCommentRequest struct {
	Body string `json:"body"`
}

type ModerateProductCommentRequest struct {
	ExpectedVersion int  `json:"expectedVersion"`
	Approve         bool `json:"approve"`

	// Why the shop is publishing or hiding this comment. Required, and signed
	// into the event log so the decision cannot be quietly rewritten.
	Reason string `json:"reason"`
}

type ReplyProductCommentRequest struct {
	ExpectedVersion int    `json:"expectedVersion"`
	Body            string `json:"body"`
}

type DeleteProductCommentRequest struct {
	ExpectedVersion int    `json:"expectedVersion"`
	Reason          string `json:"reason"`
}

type ProductCommentResponse struct {
	CommentID    string `json:"commentId"`
	ShopID       string `json:"shopId"`
	ProductID    string `json:"productId"`
	AuthorUserID string `json:"authorUserId"`
	// Empty when the account has no display name; the client shows a generic
	// label rather than printing the raw user id.
	AuthorName string `json:"authorName"`
	// Only the shop-wide moderation queue sets this: that list crosses
	// products, so a row without a product name is a decision made blind.
	ProductName string `json:"productName,omitempty"`
	Body        string `json:"body"`
	Status      string `json:"status"`

	// The buyer check behind the comment. Its presence is what lets the client
	// say "checked this at the stall" instead of taking the words on faith.
	CheckID string `json:"checkId,omitempty"`
	Verdict string `json:"verdict,omitempty"`

	ModeratedByUserID string     `json:"moderatedByUserId,omitempty"`
	ModerationReason  string     `json:"moderationReason,omitempty"`
	ModeratedAt       *time.Time `json:"moderatedAt,omitempty"`

	ShopReplyBody string     `json:"shopReplyBody,omitempty"`
	ShopRepliedAt *time.Time `json:"shopRepliedAt,omitempty"`

	Version   int       `json:"version"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ProductCommentListResponse carries the counts as well as the comments,
// because what a moderated shop did not publish is part of what the reader
// needs to know.
type ProductCommentListResponse struct {
	Items         []ProductCommentResponse `json:"items"`
	Moderation    bool                     `json:"moderation"`
	ApprovedCount int                      `json:"approvedCount"`
	PendingCount  int                      `json:"pendingCount"`
	RejectedCount int                      `json:"rejectedCount"`

	// CanComment is false until the reader has checked this product, which is
	// what the write box keys off instead of guessing.
	CanComment bool `json:"canComment"`
}
