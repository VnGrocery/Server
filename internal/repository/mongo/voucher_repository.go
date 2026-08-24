package mongo

import (
	"context"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"vngrocery/internal/domain"
)

type VoucherRepository struct{ collection *mongo.Collection }

func NewVoucherRepository(db *mongo.Database) *VoucherRepository {
	return &VoucherRepository{collection: db.Collection(vouchersCollection)}
}

func (r *VoucherRepository) Save(ctx context.Context, voucher domain.Voucher) error {
	return saveByID(ctx, r.collection, voucher.VoucherID, voucher)
}

func (r *VoucherRepository) GetByID(ctx context.Context, voucherID string) (domain.Voucher, error) {
	return getByID[domain.Voucher](ctx, r.collection, voucherID)
}

func (r *VoucherRepository) GetByCode(ctx context.Context, code string) (domain.Voucher, error) {
	items, err := listDocuments[domain.Voucher](ctx, r.collection, bson.M{"code": strings.ToUpper(strings.TrimSpace(code))})
	if err != nil || len(items) == 0 {
		return domain.Voucher{}, err
	}
	return items[0], nil
}

func (r *VoucherRepository) ListByShopID(ctx context.Context, shopID string) ([]domain.Voucher, error) {
	items, err := listDocuments[domain.Voucher](ctx, r.collection, bson.M{"shopId": shopID})
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt.After(items[j].UpdatedAt) })
	return items, nil
}

func (r *VoucherRepository) ListActive(ctx context.Context) ([]domain.Voucher, error) {
	items, err := listDocuments[domain.Voucher](ctx, r.collection, bson.M{"active": true})
	if err != nil {
		return nil, err
	}
	// Newest first. Expiry is left to the caller, which owns the clock.
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

type UserVoucherRepository struct{ collection *mongo.Collection }

func NewUserVoucherRepository(db *mongo.Database) *UserVoucherRepository {
	return &UserVoucherRepository{collection: db.Collection(userVouchersCollection)}
}

func (r *UserVoucherRepository) Save(ctx context.Context, voucher domain.UserVoucher) error {
	return saveByID(ctx, r.collection, voucher.UserVoucherID, voucher)
}

func (r *UserVoucherRepository) GetByID(ctx context.Context, userVoucherID string) (domain.UserVoucher, error) {
	return getByID[domain.UserVoucher](ctx, r.collection, userVoucherID)
}

func (r *UserVoucherRepository) GetByUserAndVoucher(ctx context.Context, userID, voucherID string) (domain.UserVoucher, error) {
	items, err := listDocuments[domain.UserVoucher](ctx, r.collection, bson.M{"userId": userID, "voucherId": voucherID})
	if err != nil || len(items) == 0 {
		return domain.UserVoucher{}, err
	}
	return items[0], nil
}

func (r *UserVoucherRepository) ListByUserID(ctx context.Context, userID string) ([]domain.UserVoucher, error) {
	items, err := listDocuments[domain.UserVoucher](ctx, r.collection, bson.M{"userId": userID})
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}
