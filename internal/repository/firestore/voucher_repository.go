package firestore

import (
	"context"
	"fmt"
	"sort"
	"strings"

	gofirestore "cloud.google.com/go/firestore"

	"vngrocery/internal/domain"
)

type VoucherRepository struct{ client *gofirestore.Client }

func NewVoucherRepository(client *gofirestore.Client) *VoucherRepository {
	return &VoucherRepository{client: client}
}

func (r *VoucherRepository) Save(ctx context.Context, voucher domain.Voucher) error {
	_, err := r.client.Collection(VouchersCollection).Doc(voucher.VoucherID).Set(ctx, voucher)
	return err
}

func (r *VoucherRepository) GetByID(ctx context.Context, voucherID string) (domain.Voucher, error) {
	doc, err := r.client.Collection(VouchersCollection).Doc(voucherID).Get(ctx)
	if err != nil {
		return domain.Voucher{}, err
	}
	var voucher domain.Voucher
	if err := doc.DataTo(&voucher); err != nil {
		return domain.Voucher{}, fmt.Errorf("decode voucher: %w", err)
	}
	return voucher, nil
}

func (r *VoucherRepository) GetByCode(ctx context.Context, code string) (domain.Voucher, error) {
	docs, err := r.client.Collection(VouchersCollection).Where("code", "==", strings.ToUpper(strings.TrimSpace(code))).Limit(1).Documents(ctx).GetAll()
	if err != nil || len(docs) == 0 {
		return domain.Voucher{}, err
	}
	var voucher domain.Voucher
	if err := docs[0].DataTo(&voucher); err != nil {
		return domain.Voucher{}, fmt.Errorf("decode voucher: %w", err)
	}
	return voucher, nil
}

func (r *VoucherRepository) ListByShopID(ctx context.Context, shopID string) ([]domain.Voucher, error) {
	docs, err := r.client.Collection(VouchersCollection).Where("shopId", "==", shopID).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	items := make([]domain.Voucher, 0, len(docs))
	for _, doc := range docs {
		var item domain.Voucher
		if err := doc.DataTo(&item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt.After(items[j].UpdatedAt) })
	return items, nil
}

type UserVoucherRepository struct{ client *gofirestore.Client }

func NewUserVoucherRepository(client *gofirestore.Client) *UserVoucherRepository {
	return &UserVoucherRepository{client: client}
}

func (r *UserVoucherRepository) Save(ctx context.Context, voucher domain.UserVoucher) error {
	_, err := r.client.Collection(UserVouchersCollection).Doc(voucher.UserVoucherID).Set(ctx, voucher)
	return err
}

func (r *UserVoucherRepository) GetByID(ctx context.Context, userVoucherID string) (domain.UserVoucher, error) {
	doc, err := r.client.Collection(UserVouchersCollection).Doc(userVoucherID).Get(ctx)
	if err != nil {
		return domain.UserVoucher{}, err
	}
	var voucher domain.UserVoucher
	err = doc.DataTo(&voucher)
	return voucher, err
}

func (r *UserVoucherRepository) GetByUserAndVoucher(ctx context.Context, userID, voucherID string) (domain.UserVoucher, error) {
	docs, err := r.client.Collection(UserVouchersCollection).Where("userId", "==", userID).Where("voucherId", "==", voucherID).Limit(1).Documents(ctx).GetAll()
	if err != nil || len(docs) == 0 {
		return domain.UserVoucher{}, err
	}
	var voucher domain.UserVoucher
	err = docs[0].DataTo(&voucher)
	return voucher, err
}

func (r *UserVoucherRepository) ListByUserID(ctx context.Context, userID string) ([]domain.UserVoucher, error) {
	docs, err := r.client.Collection(UserVouchersCollection).Where("userId", "==", userID).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	items := make([]domain.UserVoucher, 0, len(docs))
	for _, doc := range docs {
		var item domain.UserVoucher
		if err := doc.DataTo(&item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}
