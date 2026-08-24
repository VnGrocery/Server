package voucher

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
)

var (
	ErrInvalid   = errors.New("invalid voucher request")
	ErrNotFound  = errors.New("voucher not found")
	ErrForbidden = errors.New("forbidden")
	ErrUsed      = errors.New("voucher has already been used")
	ErrSoldOut   = errors.New("voucher has been fully claimed")
	ErrExpired   = errors.New("voucher is no longer available")
)

type Service struct {
	vouchers repository.VoucherRepository
	wallets  repository.UserVoucherRepository
	shops    repository.ShopRepository
	now      func() time.Time
}

type CheckInput struct {
	Code       string
	ShopID     string
	OrderValue int
}

type CheckResult struct {
	Voucher        *domain.Voucher
	Valid          bool
	Message        string
	DiscountAmount int
	FinalPrice     int
}

type CreateInput struct {
	ShopID        string
	OwnerUserID   string
	Code          string
	Title         string
	DiscountValue int
	IsPercent     bool
	MinSpend      int
	ExpiresAt     time.Time
	Note          string
	CodeFormat    string
	Manual        bool

	// TotalQuantity caps how many buyers can claim it. Zero means uncapped,
	// which is what the shop gets when it leaves the field blank.
	TotalQuantity int
}

type WalletItem struct {
	UserVoucher domain.UserVoucher
	Voucher     domain.Voucher
}

func NewService(vouchers repository.VoucherRepository, wallets repository.UserVoucherRepository, shops repository.ShopRepository) *Service {
	return &Service{vouchers: vouchers, wallets: wallets, shops: shops, now: time.Now}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (domain.Voucher, error) {
	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	input.Title = strings.TrimSpace(input.Title)
	input.CodeFormat = strings.ToUpper(strings.TrimSpace(input.CodeFormat))
	if input.ShopID == "" || input.OwnerUserID == "" || input.Code == "" || input.Title == "" || input.ExpiresAt.IsZero() || input.DiscountValue < 0 || input.MinSpend < 0 || (input.IsPercent && input.DiscountValue > 100) {
		return domain.Voucher{}, ErrInvalid
	}
	if input.TotalQuantity < 0 {
		return domain.Voucher{}, ErrInvalid
	}
	// An offer that has already expired is not an offer.
	if !s.now().Before(input.ExpiresAt) {
		return domain.Voucher{}, ErrInvalid
	}
	shop, err := s.shops.GetByID(ctx, input.ShopID)
	if err != nil || shop.ShopID == "" {
		return domain.Voucher{}, ErrNotFound
	}
	if shop.OwnerUserID != input.OwnerUserID {
		return domain.Voucher{}, ErrForbidden
	}
	if existing, _ := s.vouchers.GetByCode(ctx, input.Code); existing.VoucherID != "" {
		return domain.Voucher{}, ErrInvalid
	}
	now := s.now().UTC()
	voucher := domain.Voucher{
		VoucherID: uuid.NewString(), ShopID: input.ShopID, Code: input.Code, Title: input.Title,
		DiscountValue: input.DiscountValue, IsPercent: input.IsPercent, MinSpend: input.MinSpend,
		ExpiresAt: input.ExpiresAt.UTC(), Active: true, Manual: input.Manual,
		TotalQuantity: input.TotalQuantity,
		Note:          strings.TrimSpace(input.Note), CodeFormat: input.CodeFormat, CreatedAt: now, UpdatedAt: now,
	}
	if voucher.CodeFormat == "" {
		voucher.CodeFormat = "QR"
	}
	if err := s.vouchers.Save(ctx, voucher); err != nil {
		return domain.Voucher{}, err
	}
	return voucher, nil
}

func (s *Service) Get(ctx context.Context, voucherID string) (domain.Voucher, error) {
	voucher, err := s.vouchers.GetByID(ctx, strings.TrimSpace(voucherID))
	if err != nil || voucher.VoucherID == "" {
		return domain.Voucher{}, ErrNotFound
	}
	return voucher, nil
}

func (s *Service) ListShop(ctx context.Context, shopID string) ([]domain.Voucher, error) {
	if strings.TrimSpace(shopID) == "" {
		return nil, ErrInvalid
	}
	return s.vouchers.ListByShopID(ctx, shopID)
}

// Featured is one live offer plus the shop it belongs to, ready to advertise.
type Featured struct {
	Voucher  domain.Voucher
	ShopName string
}

// ListFeatured returns offers that a reader could actually use right now:
// active, not expired, and belonging to a shop that still exists. An offer the
// reader cannot redeem is worse than no offer at all, so none of those filters
// is optional.
func (s *Service) ListFeatured(ctx context.Context, limit int) ([]Featured, error) {
	if limit <= 0 {
		limit = 5
	}
	vouchers, err := s.vouchers.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	now := s.now()
	featured := make([]Featured, 0, limit)
	names := make(map[string]string)
	for _, voucher := range vouchers {
		if len(featured) >= limit {
			break
		}
		if now.After(voucher.ExpiresAt) {
			continue
		}
		// Advertising an offer nobody can still claim is the same mistake as
		// advertising an expired one.
		if voucher.TotalQuantity > 0 && voucher.ClaimedCount >= voucher.TotalQuantity {
			continue
		}
		name, ok := names[voucher.ShopID]
		if !ok {
			shop, err := s.shops.GetByID(ctx, voucher.ShopID)
			if err != nil || shop.ShopID == "" {
				// A voucher whose shop is gone cannot be redeemed anywhere.
				names[voucher.ShopID] = ""
				continue
			}
			name = shop.Name
			names[voucher.ShopID] = name
		}
		if name == "" {
			continue
		}
		featured = append(featured, Featured{Voucher: voucher, ShopName: name})
	}
	return featured, nil
}

func (s *Service) Check(ctx context.Context, input CheckInput) (CheckResult, error) {
	if input.OrderValue < 0 || strings.TrimSpace(input.Code) == "" || strings.TrimSpace(input.ShopID) == "" {
		return CheckResult{}, ErrInvalid
	}
	voucher, err := s.vouchers.GetByCode(ctx, input.Code)
	result := CheckResult{FinalPrice: input.OrderValue}
	if err != nil || voucher.VoucherID == "" {
		result.Message = "Không tìm thấy voucher này"
		return result, nil
	}
	result.Voucher = &voucher
	switch {
	case !voucher.Active:
		result.Message = "Voucher đang tạm khóa"
	case voucher.ShopID != input.ShopID:
		result.Message = "Voucher không áp dụng cho cửa hàng này"
	case s.now().After(voucher.ExpiresAt):
		result.Message = "Voucher đã hết hạn"
	case input.OrderValue < voucher.MinSpend:
		result.Message = "Đơn hàng chưa đạt giá trị tối thiểu"
	default:
		discount := voucher.DiscountValue
		if voucher.IsPercent {
			discount = input.OrderValue * voucher.DiscountValue / 100
		}
		if discount > input.OrderValue {
			discount = input.OrderValue
		}
		result.Valid = true
		result.Message = "Voucher hợp lệ"
		result.DiscountAmount = discount
		result.FinalPrice = input.OrderValue - discount
	}
	return result, nil
}

// SaveToWallet is a buyer claiming an offer.
//
// It used to hand over anything with an id: an expired voucher, a paused one,
// or the two-hundredth claim on an offer of fifty. Now the offer has to still
// be on, and a rationed one has to have a claim left - taken atomically, so
// two buyers reaching for the last one cannot both be told yes.
//
// Claiming twice is not an error and does not consume a second slot: the
// reader already has it, and the button that led here is idempotent by design.
func (s *Service) SaveToWallet(ctx context.Context, userID, voucherID string) (WalletItem, error) {
	if strings.TrimSpace(userID) == "" {
		return WalletItem{}, ErrInvalid
	}
	voucher, err := s.Get(ctx, voucherID)
	if err != nil {
		return WalletItem{}, err
	}
	if existing, _ := s.wallets.GetByUserAndVoucher(ctx, userID, voucherID); existing.UserVoucherID != "" {
		return WalletItem{UserVoucher: existing, Voucher: voucher}, nil
	}
	if !voucher.Active || !s.now().Before(voucher.ExpiresAt) {
		return WalletItem{}, ErrExpired
	}

	claimed, err := s.vouchers.ClaimSlot(ctx, voucherID)
	if err != nil {
		return WalletItem{}, err
	}
	if !claimed {
		return WalletItem{}, ErrSoldOut
	}

	now := s.now().UTC()
	item := domain.UserVoucher{UserVoucherID: uuid.NewString(), UserID: userID, VoucherID: voucherID, CreatedAt: now, UpdatedAt: now}
	if err := s.wallets.Save(ctx, item); err != nil {
		// The slot is already spent. Better a claim nobody holds than a
		// voucher two people hold, and the shop sees the gap in its count.
		return WalletItem{}, err
	}
	voucher.ClaimedCount++
	return WalletItem{UserVoucher: item, Voucher: voucher}, nil
}

func (s *Service) AddManual(ctx context.Context, userID string, input CreateInput) (WalletItem, error) {
	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	input.Title = strings.TrimSpace(input.Title)
	if userID == "" || input.ShopID == "" || input.Code == "" || input.Title == "" || input.ExpiresAt.IsZero() {
		return WalletItem{}, ErrInvalid
	}
	if shop, err := s.shops.GetByID(ctx, input.ShopID); err != nil || shop.ShopID == "" {
		return WalletItem{}, ErrNotFound
	}
	now := s.now().UTC()
	voucher := domain.Voucher{
		VoucherID: uuid.NewString(), ShopID: input.ShopID, Code: input.Code, Title: input.Title,
		ExpiresAt: input.ExpiresAt.UTC(), Active: true, Manual: true, Note: strings.TrimSpace(input.Note),
		CodeFormat: strings.ToUpper(strings.TrimSpace(input.CodeFormat)), CreatedAt: now, UpdatedAt: now,
	}
	if voucher.CodeFormat == "" {
		voucher.CodeFormat = "QR"
	}
	if err := s.vouchers.Save(ctx, voucher); err != nil {
		return WalletItem{}, err
	}
	return s.SaveToWallet(ctx, userID, voucher.VoucherID)
}

func (s *Service) Wallet(ctx context.Context, userID string) ([]WalletItem, error) {
	items, err := s.wallets.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]WalletItem, 0, len(items))
	for _, item := range items {
		voucher, getErr := s.vouchers.GetByID(ctx, item.VoucherID)
		if getErr == nil && voucher.VoucherID != "" {
			result = append(result, WalletItem{UserVoucher: item, Voucher: voucher})
		}
	}
	return result, nil
}

func (s *Service) Use(ctx context.Context, userID, userVoucherID string) (WalletItem, error) {
	item, err := s.wallets.GetByID(ctx, userVoucherID)
	if err != nil || item.UserVoucherID == "" {
		return WalletItem{}, ErrNotFound
	}
	if item.UserID != userID {
		return WalletItem{}, ErrForbidden
	}
	if item.Used {
		return WalletItem{}, ErrUsed
	}
	voucher, err := s.Get(ctx, item.VoucherID)
	if err != nil {
		return WalletItem{}, err
	}
	if !voucher.Active || s.now().After(voucher.ExpiresAt) {
		return WalletItem{}, ErrInvalid
	}
	now := s.now().UTC()
	item.Used, item.UsedAt, item.UpdatedAt = true, &now, now
	if err := s.wallets.Save(ctx, item); err != nil {
		return WalletItem{}, err
	}
	return WalletItem{UserVoucher: item, Voucher: voucher}, nil
}
