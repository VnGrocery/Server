package voucher

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"
	"time"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
)

type voucherRepoStub struct{ items map[string]domain.Voucher }

func (r *voucherRepoStub) Save(_ context.Context, item domain.Voucher) error {
	r.items[item.VoucherID] = item
	return nil
}
func (r *voucherRepoStub) GetByID(_ context.Context, id string) (domain.Voucher, error) {
	return r.items[id], nil
}
func (r *voucherRepoStub) GetByCode(_ context.Context, code string) (domain.Voucher, error) {
	for _, item := range r.items {
		if item.Code == code {
			return item, nil
		}
	}
	return domain.Voucher{}, nil
}
func (r *voucherRepoStub) ListByShopID(_ context.Context, shopID string) ([]domain.Voucher, error) {
	var result []domain.Voucher
	for _, item := range r.items {
		if item.ShopID == shopID {
			result = append(result, item)
		}
	}
	return result, nil
}

func (r *voucherRepoStub) ListActive(_ context.Context) ([]domain.Voucher, error) {
	var result []domain.Voucher
	for _, item := range r.items {
		if item.Active {
			result = append(result, item)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Code < result[j].Code })
	return result, nil
}

// Mirrors the atomic update in Mongo: an unrationed voucher always yields a
// slot, a rationed one only while claims remain.
//
// Go zero values stand in for absent Mongo fields here, which is exactly the
// case that got the real query wrong - see TestClaimingWorksOnVouchersOlderThanQuantity
// and the $ifNull in the repository.
func (r *voucherRepoStub) ClaimSlot(_ context.Context, voucherID string) (bool, error) {
	item, ok := r.items[voucherID]
	if !ok {
		return false, nil
	}
	if item.TotalQuantity > 0 && item.ClaimedCount >= item.TotalQuantity {
		return false, nil
	}
	item.ClaimedCount++
	r.items[voucherID] = item
	return true, nil
}

type walletRepoStub struct{ items map[string]domain.UserVoucher }

func (r *walletRepoStub) Save(_ context.Context, item domain.UserVoucher) error {
	r.items[item.UserVoucherID] = item
	return nil
}
func (r *walletRepoStub) GetByID(_ context.Context, id string) (domain.UserVoucher, error) {
	return r.items[id], nil
}
func (r *walletRepoStub) GetByUserAndVoucher(_ context.Context, userID, voucherID string) (domain.UserVoucher, error) {
	for _, item := range r.items {
		if item.UserID == userID && item.VoucherID == voucherID {
			return item, nil
		}
	}
	return domain.UserVoucher{}, nil
}
func (r *walletRepoStub) ListByUserID(_ context.Context, userID string) ([]domain.UserVoucher, error) {
	var result []domain.UserVoucher
	for _, item := range r.items {
		if item.UserID == userID {
			result = append(result, item)
		}
	}
	return result, nil
}

type shopRepoStub struct{ shop domain.Shop }

func (r shopRepoStub) Save(context.Context, domain.Shop) error { return nil }
func (r shopRepoStub) GetByID(_ context.Context, id string) (domain.Shop, error) {
	if r.shop.ShopID == id {
		return r.shop, nil
	}
	return domain.Shop{}, nil
}
func (r shopRepoStub) List(context.Context, repository.ShopListFilter) ([]domain.Shop, error) {
	return []domain.Shop{r.shop}, nil
}

func newTestService() (*Service, *voucherRepoStub, *walletRepoStub) {
	vouchers := &voucherRepoStub{items: map[string]domain.Voucher{}}
	wallets := &walletRepoStub{items: map[string]domain.UserVoucher{}}
	service := NewService(vouchers, wallets, shopRepoStub{shop: domain.Shop{ShopID: "shop-1", OwnerUserID: "seller-1", Name: "Rau Sạch Cô Ba"}})
	service.now = func() time.Time { return time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC) }
	return service, vouchers, wallets
}

func TestCreateCheckAndWalletLifecycle(t *testing.T) {
	service, _, _ := newTestService()
	voucher, err := service.Create(context.Background(), CreateInput{ShopID: "shop-1", OwnerUserID: "seller-1", Code: " fresh20 ", Title: "Fresh", DiscountValue: 20, IsPercent: true, MinSpend: 100000, ExpiresAt: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	checked, err := service.Check(context.Background(), CheckInput{Code: "FRESH20", ShopID: "shop-1", OrderValue: 200000})
	if err != nil || !checked.Valid || checked.DiscountAmount != 40000 || checked.FinalPrice != 160000 {
		t.Fatalf("unexpected check: %+v err=%v", checked, err)
	}
	item, err := service.SaveToWallet(context.Background(), "buyer-1", voucher.VoucherID)
	if err != nil || item.UserVoucher.Used {
		t.Fatalf("unexpected wallet item: %+v err=%v", item, err)
	}
	used, err := service.Use(context.Background(), "buyer-1", item.UserVoucher.UserVoucherID)
	if err != nil || !used.UserVoucher.Used || used.UserVoucher.UsedAt == nil {
		t.Fatalf("unexpected used item: %+v err=%v", used, err)
	}
}

func TestCreateRejectsNonOwner(t *testing.T) {
	service, _, _ := newTestService()
	_, err := service.Create(context.Background(), CreateInput{ShopID: "shop-1", OwnerUserID: "other", Code: "X", Title: "X", ExpiresAt: time.Now().Add(time.Hour)})
	if err != ErrForbidden {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestManualVoucherBelongsOnlyToAuthenticatedWallet(t *testing.T) {
	service, _, _ := newTestService()
	item, err := service.AddManual(context.Background(), "buyer-1", CreateInput{ShopID: "shop-1", Code: "receipt-1", Title: "Receipt", ExpiresAt: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	if item.UserVoucher.UserID != "buyer-1" || !item.Voucher.Manual || item.Voucher.Code != "RECEIPT-1" {
		t.Fatalf("unexpected manual item: %+v", item)
	}
	if _, err := service.Use(context.Background(), "buyer-2", item.UserVoucher.UserVoucherID); err != ErrForbidden {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestFeaturedOffersLeaveOutWhatNobodyCouldRedeem(t *testing.T) {
	service, vouchers, _ := newTestService()
	now := service.now()

	vouchers.items["live"] = domain.Voucher{
		VoucherID: "live", ShopID: "shop-1", Code: "AAA", Title: "Giảm 10%",
		Active: true, ExpiresAt: now.AddDate(0, 1, 0),
	}
	vouchers.items["expired"] = domain.Voucher{
		VoucherID: "expired", ShopID: "shop-1", Code: "BBB", Title: "Hết hạn",
		Active: true, ExpiresAt: now.AddDate(0, 0, -1),
	}
	vouchers.items["paused"] = domain.Voucher{
		VoucherID: "paused", ShopID: "shop-1", Code: "CCC", Title: "Tạm khóa",
		Active: false, ExpiresAt: now.AddDate(0, 1, 0),
	}
	vouchers.items["orphan"] = domain.Voucher{
		VoucherID: "orphan", ShopID: "shop-gone", Code: "DDD", Title: "Cửa hàng đã đóng",
		Active: true, ExpiresAt: now.AddDate(0, 1, 0),
	}

	featured, err := service.ListFeatured(context.Background(), 10)
	if err != nil {
		t.Fatalf("list featured: %v", err)
	}
	// Advertising an offer that cannot be used is worse than advertising none.
	if len(featured) != 1 {
		t.Fatalf("expected only the live offer, got %d", len(featured))
	}
	if featured[0].Voucher.VoucherID != "live" {
		t.Fatalf("wrong offer featured: %s", featured[0].Voucher.VoucherID)
	}
	if featured[0].ShopName == "" {
		t.Fatal("an offer without a shop name says nothing the reader can act on")
	}
}

func TestFeaturedOffersRespectTheLimit(t *testing.T) {
	service, vouchers, _ := newTestService()
	now := service.now()
	for _, code := range []string{"A", "B", "C", "D"} {
		vouchers.items[code] = domain.Voucher{
			VoucherID: code, ShopID: "shop-1", Code: code, Title: code,
			Active: true, ExpiresAt: now.AddDate(0, 1, 0),
		}
	}

	featured, err := service.ListFeatured(context.Background(), 2)
	if err != nil {
		t.Fatalf("list featured: %v", err)
	}
	if len(featured) != 2 {
		t.Fatalf("expected 2, got %d", len(featured))
	}
}

func liveVoucher(service *Service, quantity int) domain.Voucher {
	return domain.Voucher{
		VoucherID: "v-1", ShopID: "shop-1", Code: "RAUSACH10", Title: "Giảm 10%",
		DiscountValue: 10, IsPercent: true, Active: true,
		TotalQuantity: quantity,
		ExpiresAt:     service.now().AddDate(0, 1, 0),
	}
}

func TestClaimingStopsAtTheQuantityTheShopOffered(t *testing.T) {
	service, vouchers, _ := newTestService()
	vouchers.items["v-1"] = liveVoucher(service, 2)

	for _, buyer := range []string{"buyer-1", "buyer-2"} {
		if _, err := service.SaveToWallet(context.Background(), buyer, "v-1"); err != nil {
			t.Fatalf("%s could not claim: %v", buyer, err)
		}
	}

	// The third buyer is told no rather than handed a voucher the shop never
	// offered.
	if _, err := service.SaveToWallet(context.Background(), "buyer-3", "v-1"); !errors.Is(err, ErrSoldOut) {
		t.Fatalf("expected sold out, got %v", err)
	}
	if vouchers.items["v-1"].ClaimedCount != 2 {
		t.Fatalf("claimed count drifted: %d", vouchers.items["v-1"].ClaimedCount)
	}
}

func TestClaimingTwiceCostsOnlyOneOfTheQuantity(t *testing.T) {
	service, vouchers, _ := newTestService()
	vouchers.items["v-1"] = liveVoucher(service, 1)

	first, err := service.SaveToWallet(context.Background(), "buyer-1", "v-1")
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}
	second, err := service.SaveToWallet(context.Background(), "buyer-1", "v-1")
	if err != nil {
		t.Fatalf("claiming what you already hold must not fail: %v", err)
	}

	if first.UserVoucher.UserVoucherID != second.UserVoucher.UserVoucherID {
		t.Fatal("a second claim minted a second wallet entry")
	}
	if vouchers.items["v-1"].ClaimedCount != 1 {
		t.Fatalf("a repeat claim ate another slot: %d", vouchers.items["v-1"].ClaimedCount)
	}
}

func TestAnUnrationedOfferNeverRunsOut(t *testing.T) {
	service, vouchers, _ := newTestService()
	// Zero is what every voucher created before quantity existed carries.
	vouchers.items["v-1"] = liveVoucher(service, 0)

	for i := range 5 {
		if _, err := service.SaveToWallet(context.Background(), fmt.Sprintf("buyer-%d", i), "v-1"); err != nil {
			t.Fatalf("claim %d: %v", i, err)
		}
	}
	// It still counts, because the shop wants to know how many went out.
	if vouchers.items["v-1"].ClaimedCount != 5 {
		t.Fatalf("expected 5 claims counted, got %d", vouchers.items["v-1"].ClaimedCount)
	}
}

func TestAnExpiredOrPausedOfferCannotBeClaimed(t *testing.T) {
	service, vouchers, _ := newTestService()

	expired := liveVoucher(service, 0)
	expired.ExpiresAt = service.now().AddDate(0, 0, -1)
	vouchers.items["v-1"] = expired
	if _, err := service.SaveToWallet(context.Background(), "buyer-1", "v-1"); !errors.Is(err, ErrExpired) {
		t.Fatalf("expected expired, got %v", err)
	}

	paused := liveVoucher(service, 0)
	paused.Active = false
	vouchers.items["v-1"] = paused
	if _, err := service.SaveToWallet(context.Background(), "buyer-2", "v-1"); !errors.Is(err, ErrExpired) {
		t.Fatalf("expected paused to be refused, got %v", err)
	}

	// Neither attempt may spend a slot.
	if vouchers.items["v-1"].ClaimedCount != 0 {
		t.Fatalf("a refused claim still counted: %d", vouchers.items["v-1"].ClaimedCount)
	}
}

func TestFullyClaimedOffersLeaveTheAdvertSlot(t *testing.T) {
	service, vouchers, _ := newTestService()
	sold := liveVoucher(service, 1)
	sold.ClaimedCount = 1
	vouchers.items["v-1"] = sold

	featured, err := service.ListFeatured(context.Background(), 10)
	if err != nil {
		t.Fatalf("list featured: %v", err)
	}
	// Advertising an offer nobody can still claim is the same mistake as
	// advertising an expired one.
	if len(featured) != 0 {
		t.Fatalf("a sold-out offer was still advertised: %d", len(featured))
	}
}

func TestCreateRefusesAQuantityThatMakesNoSense(t *testing.T) {
	service, _, _ := newTestService()
	base := CreateInput{
		ShopID: "shop-1", OwnerUserID: "seller-1", Code: "NEW10", Title: "Giảm 10%",
		DiscountValue: 10, IsPercent: true, ExpiresAt: service.now().AddDate(0, 1, 0),
	}

	negative := base
	negative.TotalQuantity = -1
	if _, err := service.Create(context.Background(), negative); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected invalid for a negative quantity, got %v", err)
	}

	past := base
	past.ExpiresAt = service.now().AddDate(0, 0, -1)
	if _, err := service.Create(context.Background(), past); !errors.Is(err, ErrInvalid) {
		t.Fatalf("an offer that has already expired is not an offer: %v", err)
	}

	ok := base
	ok.TotalQuantity = 50
	created, err := service.Create(context.Background(), ok)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.TotalQuantity != 50 || created.ClaimedCount != 0 {
		t.Fatalf("quantity not stored: %#v", created)
	}
}

func TestClaimingWorksOnVouchersOlderThanQuantity(t *testing.T) {
	// Every voucher seeded before the quantity fields existed carries neither
	// of them. In Mongo that is an absent field, not a zero, and a plain
	// comparison never matches one - which refused every claim on every offer
	// already in the database. The repository reads both through $ifNull; this
	// is the service-level guard that the zero case stays claimable.
	service, vouchers, _ := newTestService()
	legacy := liveVoucher(service, 0)
	legacy.ClaimedCount = 0
	vouchers.items["v-1"] = legacy

	if _, err := service.SaveToWallet(context.Background(), "buyer-1", "v-1"); err != nil {
		t.Fatalf("a voucher with no quantity set must still be claimable: %v", err)
	}
}
