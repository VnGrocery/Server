package voucher

import (
	"context"
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
	service := NewService(vouchers, wallets, shopRepoStub{shop: domain.Shop{ShopID: "shop-1", OwnerUserID: "seller-1"}})
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
