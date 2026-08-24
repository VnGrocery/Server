package comment

import (
	"context"
	"errors"
	"testing"
	"time"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	"vngrocery/internal/service/audit"
)

type commentRepositoryStub struct {
	items map[string]domain.ProductComment
}

func newCommentRepositoryStub() *commentRepositoryStub {
	return &commentRepositoryStub{items: map[string]domain.ProductComment{}}
}

func (c *commentRepositoryStub) Save(ctx context.Context, comment domain.ProductComment) error {
	c.items[comment.CommentID] = comment
	return nil
}

func (c *commentRepositoryStub) GetByID(ctx context.Context, commentID string) (domain.ProductComment, error) {
	item, ok := c.items[commentID]
	if !ok {
		return domain.ProductComment{}, errors.New("not found")
	}
	return item, nil
}

func (c *commentRepositoryStub) List(ctx context.Context, filter repository.ProductCommentListFilter) ([]domain.ProductComment, error) {
	out := make([]domain.ProductComment, 0, len(c.items))
	for _, item := range c.items {
		if filter.ShopID != "" && item.ShopID != filter.ShopID {
			continue
		}
		if filter.ProductID != "" && item.ProductID != filter.ProductID {
			continue
		}
		if filter.AuthorUserID != "" && item.AuthorUserID != filter.AuthorUserID {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}

type shopRepositoryStub struct{ shop domain.Shop }

func (s shopRepositoryStub) Save(ctx context.Context, shop domain.Shop) error { return nil }
func (s shopRepositoryStub) GetByID(ctx context.Context, shopID string) (domain.Shop, error) {
	if s.shop.ShopID != shopID {
		return domain.Shop{}, errors.New("not found")
	}
	return s.shop, nil
}
func (s shopRepositoryStub) List(ctx context.Context, filter repository.ShopListFilter) ([]domain.Shop, error) {
	return nil, errors.New("not implemented")
}

type productRepositoryStub struct{ product domain.Product }

func (p productRepositoryStub) Save(ctx context.Context, product domain.Product) error { return nil }
func (p productRepositoryStub) GetByID(ctx context.Context, productID string) (domain.Product, error) {
	if p.product.ProductID != productID {
		return domain.Product{}, errors.New("not found")
	}
	return p.product, nil
}
func (p productRepositoryStub) List(ctx context.Context, filter repository.ProductListFilter) ([]domain.Product, error) {
	return nil, errors.New("not implemented")
}

type checkRepositoryStub struct{ checks []domain.BuyerCheck }

func (c checkRepositoryStub) Save(ctx context.Context, check domain.BuyerCheck) error { return nil }
func (c checkRepositoryStub) GetByID(ctx context.Context, checkID string) (domain.BuyerCheck, error) {
	return domain.BuyerCheck{}, errors.New("not implemented")
}
func (c checkRepositoryStub) ListByShopID(ctx context.Context, shopID string) ([]domain.BuyerCheck, error) {
	return c.checks, nil
}
func (c checkRepositoryStub) ListByBuyerUserID(ctx context.Context, buyerUserID string) ([]domain.BuyerCheck, error) {
	return c.checks, nil
}
func (c checkRepositoryStub) List(ctx context.Context, filter repository.BuyerCheckListFilter) ([]domain.BuyerCheck, error) {
	out := make([]domain.BuyerCheck, 0, len(c.checks))
	for _, check := range c.checks {
		if filter.ShopID != "" && check.ShopID != filter.ShopID {
			continue
		}
		if filter.ProductID != "" && check.ProductID != filter.ProductID {
			continue
		}
		if filter.BuyerUserID != "" && check.BuyerUserID != filter.BuyerUserID {
			continue
		}
		out = append(out, check)
	}
	return out, nil
}

type userRepositoryStub struct{}

func (userRepositoryStub) Save(ctx context.Context, user domain.User) error { return nil }
func (userRepositoryStub) GetByID(ctx context.Context, userID string) (domain.User, error) {
	return domain.User{UserID: userID, DisplayName: "Khách hàng"}, nil
}
func (userRepositoryStub) List(ctx context.Context, filter repository.UserListFilter) ([]domain.User, error) {
	return nil, errors.New("not implemented")
}

type auditStub struct{ inputs []audit.Input }

func (a *auditStub) Log(ctx context.Context, input audit.Input) error {
	a.inputs = append(a.inputs, input)
	return nil
}

type fixture struct {
	service  *Service
	comments *commentRepositoryStub
	audit    *auditStub
}

func newFixture(moderationOn bool, checks []domain.BuyerCheck) fixture {
	comments := newCommentRepositoryStub()
	auditLog := &auditStub{}
	service := NewService(
		comments,
		shopRepositoryStub{shop: domain.Shop{ShopID: "s1", OwnerUserID: "owner", CommentModeration: moderationOn}},
		productRepositoryStub{product: domain.Product{ProductID: "p1", ShopID: "s1", Name: "Rau muống"}},
		checkRepositoryStub{checks: checks},
		userRepositoryStub{},
		auditLog,
	).WithClock(func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) })
	return fixture{service: service, comments: comments, audit: auditLog}
}

func checkedProduct() []domain.BuyerCheck {
	return []domain.BuyerCheck{{
		CheckID:     "chk1",
		ShopID:      "s1",
		ProductID:   "p1",
		BuyerUserID: "u1",
		Verdict:     "trusted",
	}}
}

func TestCreateRequiresACheckOnTheProduct(t *testing.T) {
	f := newFixture(false, nil)

	_, err := f.service.Create(context.Background(), CreateInput{
		ShopID:       "s1",
		ProductID:    "p1",
		AuthorUserID: "u1",
		Body:         "Rau còn tươi đúng như ghi nhận",
	})
	if !errors.Is(err, ErrCheckRequired) {
		t.Fatalf("expected ErrCheckRequired, got %v", err)
	}
	if len(f.comments.items) != 0 {
		t.Fatalf("a comment was stored without a check behind it")
	}
}

func TestCreatePublishesWhenModerationIsOff(t *testing.T) {
	f := newFixture(false, checkedProduct())

	comment, err := f.service.Create(context.Background(), CreateInput{
		ShopID:       "s1",
		ProductID:    "p1",
		AuthorUserID: "u1",
		Body:         "Rau còn tươi đúng như ghi nhận",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if comment.Status != StatusApproved {
		t.Fatalf("expected an approved comment, got %q", comment.Status)
	}
	if comment.CheckID != "chk1" || comment.Verdict != "trusted" {
		t.Fatalf("the check behind the comment was not recorded: %+v", comment)
	}
	if len(f.audit.inputs) != 1 || f.audit.inputs[0].Action != "product_comment.created" {
		t.Fatalf("the comment was not written to the signed log: %+v", f.audit.inputs)
	}
}

func TestCreateHoldsTheCommentWhenModerationIsOn(t *testing.T) {
	f := newFixture(true, checkedProduct())

	comment, err := f.service.Create(context.Background(), CreateInput{
		ShopID:       "s1",
		ProductID:    "p1",
		AuthorUserID: "u1",
		Body:         "Giá cao hơn niêm yết",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if comment.Status != StatusPending {
		t.Fatalf("expected a pending comment, got %q", comment.Status)
	}
}

func TestModerationDecisionsNeedAReasonAndAreSigned(t *testing.T) {
	f := newFixture(true, checkedProduct())
	comment, err := f.service.Create(context.Background(), CreateInput{
		ShopID: "s1", ProductID: "p1", AuthorUserID: "u1", Body: "Giá cao hơn niêm yết",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := f.service.Moderate(context.Background(), ModerateInput{
		ShopID: "s1", CommentID: comment.CommentID, OwnerUserID: "owner",
		ExpectedVersion: comment.Version, Approve: false, Reason: "ok",
	}); !errors.Is(err, ErrInvalidComment) {
		t.Fatalf("a four-character reason was accepted: %v", err)
	}

	rejected, err := f.service.Moderate(context.Background(), ModerateInput{
		ShopID: "s1", CommentID: comment.CommentID, OwnerUserID: "owner",
		ExpectedVersion: comment.Version, Approve: false, Reason: "Bình luận nhầm sản phẩm khác",
	})
	if err != nil {
		t.Fatalf("moderate: %v", err)
	}
	if rejected.Status != StatusRejected {
		t.Fatalf("expected rejected, got %q", rejected.Status)
	}
	last := f.audit.inputs[len(f.audit.inputs)-1]
	if last.Action != "product_comment.rejected" || last.Reason != "Bình luận nhầm sản phẩm khác" {
		t.Fatalf("the rejection was not signed with its reason: %+v", last)
	}
}

func TestOnlyTheOwnerModerates(t *testing.T) {
	f := newFixture(true, checkedProduct())
	comment, err := f.service.Create(context.Background(), CreateInput{
		ShopID: "s1", ProductID: "p1", AuthorUserID: "u1", Body: "Giá cao hơn niêm yết",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := f.service.Moderate(context.Background(), ModerateInput{
		ShopID: "s1", CommentID: comment.CommentID, OwnerUserID: "someone-else",
		ExpectedVersion: comment.Version, Approve: true, Reason: "Nội dung hợp lệ",
	}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("a stranger moderated the shop's comments: %v", err)
	}
}

func TestListHidesHeldCommentsFromStrangersButCountsThem(t *testing.T) {
	f := newFixture(true, checkedProduct())
	if _, err := f.service.Create(context.Background(), CreateInput{
		ShopID: "s1", ProductID: "p1", AuthorUserID: "u1", Body: "Giá cao hơn niêm yết",
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	views, summary, err := f.service.List(context.Background(), ListInput{
		ShopID: "s1", ProductID: "p1", ActorUserID: "stranger",
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(views) != 0 {
		t.Fatalf("a stranger read a comment the shop has not published: %+v", views)
	}
	if summary.PendingCount != 1 || !summary.ModerationOn {
		t.Fatalf("the reader was not told something is being held: %+v", summary)
	}
	if summary.CanComment {
		t.Fatalf("a reader with no check on this product was offered the write box")
	}

	// The author still sees their own, so it does not look like it vanished.
	own, _, err := f.service.List(context.Background(), ListInput{
		ShopID: "s1", ProductID: "p1", ActorUserID: "u1",
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(own) != 1 {
		t.Fatalf("the author cannot see their own held comment: %+v", own)
	}
}

func TestRewritingACommentGoesBackThroughModeration(t *testing.T) {
	f := newFixture(true, checkedProduct())
	comment, err := f.service.Create(context.Background(), CreateInput{
		ShopID: "s1", ProductID: "p1", AuthorUserID: "u1", Body: "Rau còn tươi đúng như ghi nhận",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	approved, err := f.service.Moderate(context.Background(), ModerateInput{
		ShopID: "s1", CommentID: comment.CommentID, OwnerUserID: "owner",
		ExpectedVersion: comment.Version, Approve: true, Reason: "Nội dung hợp lệ",
	})
	if err != nil {
		t.Fatalf("moderate: %v", err)
	}
	if approved.Status != StatusApproved {
		t.Fatalf("expected approved, got %q", approved.Status)
	}

	rewritten, err := f.service.Create(context.Background(), CreateInput{
		ShopID: "s1", ProductID: "p1", AuthorUserID: "u1", Body: "Hôm nay rau héo, không như ghi nhận",
	})
	if err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	if rewritten.CommentID != comment.CommentID {
		t.Fatalf("the rewrite created a second comment instead of replacing the first")
	}
	if rewritten.Status != StatusPending {
		t.Fatalf("a rewritten comment kept the approval of the old text: %q", rewritten.Status)
	}
	if rewritten.ModerationReason != "" || rewritten.ModeratedAt != nil {
		t.Fatalf("the old moderation decision survived the rewrite: %+v", rewritten)
	}
}

func TestTheShopCannotDeleteABuyersComment(t *testing.T) {
	f := newFixture(false, checkedProduct())
	comment, err := f.service.Create(context.Background(), CreateInput{
		ShopID: "s1", ProductID: "p1", AuthorUserID: "u1", Body: "Giá cao hơn niêm yết",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := f.service.Delete(context.Background(), DeleteInput{
		ShopID: "s1", CommentID: comment.CommentID, ActorUserID: "owner",
		ExpectedVersion: comment.Version, Reason: "Không thích bình luận này",
	}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("the shop deleted a buyer's comment outright: %v", err)
	}

	if _, err := f.service.Delete(context.Background(), DeleteInput{
		ShopID: "s1", CommentID: comment.CommentID, ActorUserID: "u1",
		ExpectedVersion: comment.Version, Reason: "Tôi nhầm sản phẩm",
	}); err != nil {
		t.Fatalf("the author could not withdraw their own comment: %v", err)
	}
}

func TestTheOwnerQueueNamesTheProductBeingJudged(t *testing.T) {
	// The queue crosses every product in the shop, and one of the rejection
	// reasons the app offers is "posted on the wrong product" - which nobody
	// can tell without the product's name on the row.
	f := newFixture(true, checkedProduct())
	if _, err := f.service.Create(context.Background(), CreateInput{
		ShopID: "s1", ProductID: "p1", AuthorUserID: "u1", Body: "Rau còn tươi đúng như ghi nhận",
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	views, summary, err := f.service.ListForShop(context.Background(), ShopQueueInput{
		ShopID: "s1", OwnerUserID: "owner", Status: StatusPending,
	})
	if err != nil {
		t.Fatalf("list for shop: %v", err)
	}
	if len(views) != 1 {
		t.Fatalf("expected one pending comment, got %d", len(views))
	}
	if views[0].ProductName != "Rau muống" {
		t.Fatalf("queue row does not name its product: %q", views[0].ProductName)
	}
	if summary.PendingCount != 1 {
		t.Fatalf("expected one pending, got %d", summary.PendingCount)
	}
}
