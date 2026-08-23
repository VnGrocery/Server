// Package comment holds product comments written by buyers who checked the
// goods, and the shop-side moderation that can hold them back.
//
// A comment is only worth reading if the person writing it stood in front of
// the product, so writing one requires a buyer check on that product. That is
// also what keeps the comment count meaningful in the trust score: it cannot be
// inflated by accounts that never went to the stall.
package comment

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	"vngrocery/internal/service/audit"
)

var (
	ErrInvalidComment = errors.New("invalid comment request")
	ErrNotFound       = errors.New("comment not found")
	ErrForbidden      = errors.New("forbidden")
	ErrCheckRequired  = errors.New("check the product before commenting on it")
)

const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"
	StatusDeleted  = "deleted"

	minBody   = 5
	maxBody   = 1000
	minReason = 5
	maxReason = 200
)

// AuditLogger is the signed event log. Every comment and every moderation
// decision goes through it, so a shop that hides a comment leaves a record of
// the hiding that it cannot edit later.
type AuditLogger interface {
	Log(ctx context.Context, input audit.Input) error
}

type Service struct {
	comments repository.ProductCommentRepository
	shops    repository.ShopRepository
	products repository.ProductRepository
	checks   repository.BuyerCheckRepository
	users    repository.UserRepository
	audit    AuditLogger
	now      func() time.Time
}

func NewService(
	comments repository.ProductCommentRepository,
	shops repository.ShopRepository,
	products repository.ProductRepository,
	checks repository.BuyerCheckRepository,
	users repository.UserRepository,
	auditLogger AuditLogger,
) *Service {
	return &Service{
		comments: comments,
		shops:    shops,
		products: products,
		checks:   checks,
		users:    users,
		audit:    auditLogger,
		now:      time.Now,
	}
}

// WithClock replaces the clock, so tests can pin timestamps.
func (s *Service) WithClock(now func() time.Time) *Service {
	if now != nil {
		s.now = now
	}
	return s
}

type CreateInput struct {
	ShopID       string
	ProductID    string
	AuthorUserID string
	Body         string
}

type ListInput struct {
	ShopID    string
	ProductID string

	// ActorUserID decides how much of the queue is visible. The shop owner sees
	// held-back and rejected comments; everyone else sees what was published
	// plus their own, so a buyer can tell "waiting for the shop" from "gone".
	ActorUserID string
}

type ShopQueueInput struct {
	ShopID      string
	OwnerUserID string

	// Status narrows the queue, e.g. "pending". Empty returns everything the
	// owner can act on.
	Status string
}

type ModerateInput struct {
	ShopID          string
	CommentID       string
	OwnerUserID     string
	ExpectedVersion int

	// Approve or reject. Rejecting is the destructive one, and both are signed.
	Approve bool

	// Why. Required, and inside the signed envelope: a shop that hides a
	// comment has to say why in words a buyer can later read.
	Reason string
}

type DeleteInput struct {
	ShopID          string
	CommentID       string
	ActorUserID     string
	ExpectedVersion int
	Reason          string
}

// View pairs a comment with the display name of whoever wrote it, so the API
// does not make the client resolve user ids it is not allowed to read.
type View struct {
	Comment    domain.ProductComment
	AuthorName string
}

// Summary is what a product screen needs to be honest about the comment
// section: how many comments exist, and how many of them the reader is not
// being shown.
type Summary struct {
	ModerationOn  bool
	ApprovedCount int
	PendingCount  int
	RejectedCount int

	// CanComment is whether the reader has checked this product and may
	// therefore write. The screen asks rather than guessing, so the write box
	// and the "check it first" line are never both wrong.
	CanComment bool
}

func (s *Service) Create(ctx context.Context, input CreateInput) (domain.ProductComment, error) {
	shopID := strings.TrimSpace(input.ShopID)
	productID := strings.TrimSpace(input.ProductID)
	authorID := strings.TrimSpace(input.AuthorUserID)
	if shopID == "" {
		return domain.ProductComment{}, fmt.Errorf("%w: shopId is required", ErrInvalidComment)
	}
	if productID == "" {
		return domain.ProductComment{}, fmt.Errorf("%w: productId is required", ErrInvalidComment)
	}
	if authorID == "" {
		return domain.ProductComment{}, fmt.Errorf("%w: authorUserId is required", ErrInvalidComment)
	}
	body, err := validateBody(input.Body)
	if err != nil {
		return domain.ProductComment{}, err
	}
	if s.comments == nil || s.shops == nil || s.products == nil || s.checks == nil {
		return domain.ProductComment{}, fmt.Errorf("comment dependencies are not configured")
	}

	shop, err := s.shops.GetByID(ctx, shopID)
	if err != nil || shop.ShopID == "" {
		return domain.ProductComment{}, ErrNotFound
	}
	product, err := s.products.GetByID(ctx, productID)
	if err != nil || product.ProductID == "" || product.ShopID != shopID {
		return domain.ProductComment{}, ErrNotFound
	}

	// The entitlement. Without a check on this product by this account there is
	// no evidence behind the words, so there is no comment.
	check, err := s.latestCheck(ctx, shopID, productID, authorID)
	if err != nil {
		return domain.ProductComment{}, err
	}
	if check.CheckID == "" {
		return domain.ProductComment{}, ErrCheckRequired
	}

	// One comment per person per product. A second one replaces the first
	// rather than stacking, the same rule shop reviews already follow.
	existing, err := s.byAuthor(ctx, shopID, productID, authorID)
	if err != nil {
		return domain.ProductComment{}, err
	}

	now := s.now().UTC()
	status := StatusApproved
	action := "product_comment.created"
	if shop.CommentModeration {
		status = StatusPending
	}

	comment := existing
	if comment.CommentID == "" {
		comment = domain.ProductComment{
			CommentID:    uuid.NewString(),
			ShopID:       shopID,
			ProductID:    productID,
			AuthorUserID: authorID,
			CreatedAt:    now,
		}
	} else {
		action = "product_comment.updated"
		// A rewritten comment goes back through moderation, and loses the old
		// decision: approving one sentence must not approve a different one.
		comment.ModeratedByUserID = ""
		comment.ModerationReason = ""
		comment.ModeratedAt = nil
	}
	before := existing
	comment.Body = body
	comment.Status = status
	comment.CheckID = check.CheckID
	comment.Verdict = check.Verdict
	comment.Version++
	comment.UpdatedAt = now

	if err := s.comments.Save(ctx, comment); err != nil {
		return domain.ProductComment{}, err
	}
	if err := s.log(ctx, authorID, comment, action, status, "", audit.MutationPayload{
		Before: nonZero(before),
		After:  comment,
	}); err != nil {
		return domain.ProductComment{}, err
	}
	return comment, nil
}

// ListForShop is the owner's moderation queue: everything written across the
// shop, newest first, optionally narrowed to one status.
//
// Owner only. The per-product list is what buyers read; this one exists so the
// seller does not have to walk their own catalogue looking for what is waiting.
func (s *Service) ListForShop(ctx context.Context, input ShopQueueInput) ([]View, Summary, error) {
	shopID := strings.TrimSpace(input.ShopID)
	if shopID == "" {
		return nil, Summary{}, fmt.Errorf("%w: shopId is required", ErrInvalidComment)
	}
	if s.comments == nil || s.shops == nil {
		return nil, Summary{}, fmt.Errorf("comment dependencies are not configured")
	}
	shop, err := s.shops.GetByID(ctx, shopID)
	if err != nil || shop.ShopID == "" {
		return nil, Summary{}, ErrNotFound
	}
	if shop.OwnerUserID != strings.TrimSpace(input.OwnerUserID) {
		return nil, Summary{}, ErrForbidden
	}
	all, err := s.comments.List(ctx, repository.ProductCommentListFilter{ShopID: shopID})
	if err != nil {
		return nil, Summary{}, err
	}

	summary := Summary{ModerationOn: shop.CommentModeration}
	views := make([]View, 0, len(all))
	for _, item := range all {
		switch item.Status {
		case StatusApproved:
			summary.ApprovedCount++
		case StatusPending:
			summary.PendingCount++
		case StatusRejected:
			summary.RejectedCount++
		}
		if item.Status == StatusDeleted {
			continue
		}
		if status := strings.TrimSpace(input.Status); status != "" && item.Status != status {
			continue
		}
		views = append(views, View{Comment: item, AuthorName: s.displayName(ctx, item.AuthorUserID)})
	}
	return views, summary, nil
}

func (s *Service) List(ctx context.Context, input ListInput) ([]View, Summary, error) {
	shopID := strings.TrimSpace(input.ShopID)
	productID := strings.TrimSpace(input.ProductID)
	if shopID == "" || productID == "" {
		return nil, Summary{}, fmt.Errorf("%w: shopId and productId are required", ErrInvalidComment)
	}
	if s.comments == nil || s.shops == nil {
		return nil, Summary{}, fmt.Errorf("comment dependencies are not configured")
	}
	shop, err := s.shops.GetByID(ctx, shopID)
	if err != nil || shop.ShopID == "" {
		return nil, Summary{}, ErrNotFound
	}
	all, err := s.comments.List(ctx, repository.ProductCommentListFilter{
		ShopID:    shopID,
		ProductID: productID,
	})
	if err != nil {
		return nil, Summary{}, err
	}

	actorID := strings.TrimSpace(input.ActorUserID)
	isOwner := actorID != "" && actorID == shop.OwnerUserID

	summary := Summary{ModerationOn: shop.CommentModeration}
	if actorID := strings.TrimSpace(input.ActorUserID); actorID != "" && s.checks != nil {
		check, err := s.latestCheck(ctx, shopID, productID, actorID)
		if err != nil {
			return nil, Summary{}, err
		}
		summary.CanComment = check.CheckID != ""
	}
	views := make([]View, 0, len(all))
	for _, item := range all {
		switch item.Status {
		case StatusApproved:
			summary.ApprovedCount++
		case StatusPending:
			summary.PendingCount++
		case StatusRejected:
			summary.RejectedCount++
		}
		if !s.visible(item, actorID, isOwner) {
			continue
		}
		views = append(views, View{Comment: item, AuthorName: s.displayName(ctx, item.AuthorUserID)})
	}
	return views, summary, nil
}

// visible decides who gets to see a comment that is not published.
//
// The owner sees everything, because they are the one deciding. The author sees
// their own, because a comment that silently vanished would look like a bug.
// Nobody else sees anything but approved text - the counts are published
// instead, which is what makes the moderation visible without exposing what a
// stranger wrote.
func (s *Service) visible(comment domain.ProductComment, actorID string, isOwner bool) bool {
	if comment.Status == StatusDeleted {
		return isOwner
	}
	if comment.Status == StatusApproved {
		return true
	}
	if isOwner {
		return true
	}
	return actorID != "" && actorID == comment.AuthorUserID
}

func (s *Service) Moderate(ctx context.Context, input ModerateInput) (domain.ProductComment, error) {
	reason, err := validateReason(input.Reason)
	if err != nil {
		return domain.ProductComment{}, err
	}
	comment, shop, err := s.load(ctx, input.ShopID, input.CommentID)
	if err != nil {
		return domain.ProductComment{}, err
	}
	if shop.OwnerUserID != strings.TrimSpace(input.OwnerUserID) {
		return domain.ProductComment{}, ErrForbidden
	}
	if input.ExpectedVersion <= 0 || comment.Version != input.ExpectedVersion {
		return domain.ProductComment{}, ErrInvalidComment
	}
	if comment.Status == StatusDeleted {
		return domain.ProductComment{}, ErrNotFound
	}

	before := comment
	now := s.now().UTC()
	comment.Status = StatusRejected
	action := "product_comment.rejected"
	if input.Approve {
		comment.Status = StatusApproved
		action = "product_comment.approved"
	}
	comment.ModeratedByUserID = strings.TrimSpace(input.OwnerUserID)
	comment.ModerationReason = reason
	comment.ModeratedAt = &now
	comment.Version++
	comment.UpdatedAt = now

	if err := s.comments.Save(ctx, comment); err != nil {
		return domain.ProductComment{}, err
	}
	if err := s.log(ctx, input.OwnerUserID, comment, action, comment.Status, reason, audit.MutationPayload{
		Before: before,
		After:  comment,
	}); err != nil {
		return domain.ProductComment{}, err
	}
	return comment, nil
}

// Delete removes a comment. The author can take back their own words; the shop
// owner cannot delete them, only reject them, so a buyer's comment can never
// disappear from the record without a moderation decision attached.
func (s *Service) Delete(ctx context.Context, input DeleteInput) (domain.ProductComment, error) {
	reason, err := validateReason(input.Reason)
	if err != nil {
		return domain.ProductComment{}, err
	}
	comment, _, err := s.load(ctx, input.ShopID, input.CommentID)
	if err != nil {
		return domain.ProductComment{}, err
	}
	if comment.AuthorUserID != strings.TrimSpace(input.ActorUserID) {
		return domain.ProductComment{}, ErrForbidden
	}
	if input.ExpectedVersion <= 0 || comment.Version != input.ExpectedVersion {
		return domain.ProductComment{}, ErrInvalidComment
	}
	if comment.Status == StatusDeleted {
		return domain.ProductComment{}, ErrNotFound
	}

	before := comment
	comment.Status = StatusDeleted
	comment.Version++
	comment.UpdatedAt = s.now().UTC()

	if err := s.comments.Save(ctx, comment); err != nil {
		return domain.ProductComment{}, err
	}
	if err := s.log(ctx, input.ActorUserID, comment, "product_comment.deleted", StatusDeleted, reason, audit.MutationPayload{
		Before: before,
		After:  comment,
	}); err != nil {
		return domain.ProductComment{}, err
	}
	return comment, nil
}

func (s *Service) load(ctx context.Context, shopID, commentID string) (domain.ProductComment, domain.Shop, error) {
	shopID = strings.TrimSpace(shopID)
	commentID = strings.TrimSpace(commentID)
	if shopID == "" || commentID == "" {
		return domain.ProductComment{}, domain.Shop{}, fmt.Errorf("%w: shopId and commentId are required", ErrInvalidComment)
	}
	if s.comments == nil || s.shops == nil {
		return domain.ProductComment{}, domain.Shop{}, fmt.Errorf("comment dependencies are not configured")
	}
	shop, err := s.shops.GetByID(ctx, shopID)
	if err != nil || shop.ShopID == "" {
		return domain.ProductComment{}, domain.Shop{}, ErrNotFound
	}
	comment, err := s.comments.GetByID(ctx, commentID)
	if err != nil || comment.CommentID == "" || comment.ShopID != shopID {
		return domain.ProductComment{}, domain.Shop{}, ErrNotFound
	}
	return comment, shop, nil
}

func (s *Service) latestCheck(ctx context.Context, shopID, productID, buyerUserID string) (domain.BuyerCheck, error) {
	checks, err := s.checks.List(ctx, repository.BuyerCheckListFilter{
		ShopID:      shopID,
		ProductID:   productID,
		BuyerUserID: buyerUserID,
	})
	if err != nil {
		return domain.BuyerCheck{}, err
	}
	if len(checks) == 0 {
		return domain.BuyerCheck{}, nil
	}
	// The repository sorts newest first.
	return checks[0], nil
}

func (s *Service) byAuthor(ctx context.Context, shopID, productID, authorID string) (domain.ProductComment, error) {
	items, err := s.comments.List(ctx, repository.ProductCommentListFilter{
		ShopID:       shopID,
		ProductID:    productID,
		AuthorUserID: authorID,
	})
	if err != nil {
		return domain.ProductComment{}, err
	}
	for _, item := range items {
		if item.Status != StatusDeleted {
			return item, nil
		}
	}
	return domain.ProductComment{}, nil
}

func (s *Service) displayName(ctx context.Context, userID string) string {
	if s.users == nil || strings.TrimSpace(userID) == "" {
		return ""
	}
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return ""
	}
	return user.DisplayName
}

func (s *Service) log(ctx context.Context, actorUserID string, comment domain.ProductComment, action, status, reason string, payload any) error {
	if s.audit == nil {
		return nil
	}
	return s.audit.Log(ctx, audit.Input{
		ActorUserID:     strings.TrimSpace(actorUserID),
		ResourceType:    "product_comment",
		ResourceID:      comment.CommentID,
		ResourceVersion: comment.Version,
		Action:          action,
		Status:          status,
		Reason:          strings.TrimSpace(reason),
		Payload:         payload,
	})
}

func nonZero(comment domain.ProductComment) any {
	if comment.CommentID == "" {
		return nil
	}
	return comment
}

func validateBody(body string) (string, error) {
	trimmed := strings.TrimSpace(body)
	if len([]rune(trimmed)) < minBody {
		return "", fmt.Errorf("%w: body is required", ErrInvalidComment)
	}
	if len([]rune(trimmed)) > maxBody {
		return "", fmt.Errorf("%w: body is too long", ErrInvalidComment)
	}
	return trimmed, nil
}

func validateReason(reason string) (string, error) {
	trimmed := strings.TrimSpace(reason)
	if len([]rune(trimmed)) < minReason {
		return "", fmt.Errorf("%w: reason is required", ErrInvalidComment)
	}
	if len([]rune(trimmed)) > maxReason {
		return "", fmt.Errorf("%w: reason is too long", ErrInvalidComment)
	}
	return trimmed, nil
}
