package repository

import (
	"context"
	"time"

	"vngrocery/internal/domain"
)

type ShopListFilter struct {
	Status      string
	OwnerUserID string
}

type ProductListFilter struct {
	ShopID      string
	Status      string
	OwnerUserID string
}

type UserListFilter struct {
	Status string
	Role   string
}

type UserRepository interface {
	Save(ctx context.Context, user domain.User) error
	GetByID(ctx context.Context, userID string) (domain.User, error)
	List(ctx context.Context, filter UserListFilter) ([]domain.User, error)
}

type RefreshTokenRepository interface {
	Save(ctx context.Context, token domain.RefreshToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (domain.RefreshToken, error)
}

type PasswordResetTokenRepository interface {
	Save(ctx context.Context, token domain.PasswordResetToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (domain.PasswordResetToken, error)
}

type BundleTokenUseRepository interface {
	Reserve(ctx context.Context, usage domain.BundleTokenUse) (bool, error)
	GetByID(ctx context.Context, useID string) (domain.BundleTokenUse, error)
	DeleteExpired(ctx context.Context, before time.Time, limit int) (int, error)
}

type ShopRepository interface {
	Save(ctx context.Context, shop domain.Shop) error
	GetByID(ctx context.Context, shopID string) (domain.Shop, error)
	List(ctx context.Context, filter ShopListFilter) ([]domain.Shop, error)
}

type ProductRepository interface {
	Save(ctx context.Context, product domain.Product) error
	GetByID(ctx context.Context, productID string) (domain.Product, error)
	List(ctx context.Context, filter ProductListFilter) ([]domain.Product, error)
}

type ProductFreshnessReportRepository interface {
	Save(ctx context.Context, report domain.ProductFreshnessReport) error
	GetByID(ctx context.Context, reportID string) (domain.ProductFreshnessReport, error)
	ListByProductID(ctx context.Context, productID string) ([]domain.ProductFreshnessReport, error)
	ListByReporterUserID(ctx context.Context, reporterUserID string) ([]domain.ProductFreshnessReport, error)
	List(ctx context.Context, filter ProductFreshnessReportListFilter) ([]domain.ProductFreshnessReport, error)
}

type PledgeRepository interface {
	Save(ctx context.Context, pledge domain.Pledge) error
	GetByID(ctx context.Context, pledgeID string) (domain.Pledge, error)
	ListByShopID(ctx context.Context, shopID string) ([]domain.Pledge, error)
	ListByChainAnchorStatus(ctx context.Context, status string, limit int) ([]domain.Pledge, error)
}

type BuyerCheckRepository interface {
	Save(ctx context.Context, check domain.BuyerCheck) error
	GetByID(ctx context.Context, checkID string) (domain.BuyerCheck, error)
	ListByShopID(ctx context.Context, shopID string) ([]domain.BuyerCheck, error)
	ListByBuyerUserID(ctx context.Context, buyerUserID string) ([]domain.BuyerCheck, error)
	List(ctx context.Context, filter BuyerCheckListFilter) ([]domain.BuyerCheck, error)
}

type BuyerCheckListFilter struct {
	CheckID        string
	ShopID         string
	BundleID       string
	ProductID      string
	BuyerUserID    string
	Status         string
	Verdict        string
	LocationStatus string
	CreatedAfter   time.Time
	CreatedBefore  time.Time
}

type ProductFreshnessReportListFilter struct {
	ReportID       string
	ShopID         string
	ProductID      string
	ReporterUserID string
	Status         string
	CreatedAfter   time.Time
	CreatedBefore  time.Time
}

type ShopReviewRepository interface {
	Save(ctx context.Context, review domain.ShopReview) error
	GetByShopAndUser(ctx context.Context, shopID, reviewerUserID string) (domain.ShopReview, error)
	ListByShopID(ctx context.Context, shopID string) ([]domain.ShopReview, error)
}

type VoucherRepository interface {
	Save(ctx context.Context, voucher domain.Voucher) error
	GetByID(ctx context.Context, voucherID string) (domain.Voucher, error)
	GetByCode(ctx context.Context, code string) (domain.Voucher, error)
	ListByShopID(ctx context.Context, shopID string) ([]domain.Voucher, error)
}

type UserVoucherRepository interface {
	Save(ctx context.Context, voucher domain.UserVoucher) error
	GetByID(ctx context.Context, userVoucherID string) (domain.UserVoucher, error)
	GetByUserAndVoucher(ctx context.Context, userID, voucherID string) (domain.UserVoucher, error)
	ListByUserID(ctx context.Context, userID string) ([]domain.UserVoucher, error)
}

type AuthUserRepository interface {
	NewUserID() string
	Save(ctx context.Context, user domain.AuthUser) error
	GetByID(ctx context.Context, userID string) (domain.AuthUser, error)
	GetByEmail(ctx context.Context, emailLower string) (domain.AuthUser, error)
	GetByGoogleSub(ctx context.Context, googleSub string) (domain.AuthUser, error)
}

type EventLogRepository interface {
	Save(ctx context.Context, event domain.EventLog) error
	GetByID(ctx context.Context, eventID string) (domain.EventLog, error)
	GetLatestByResource(ctx context.Context, resourceType, resourceID string) (domain.EventLog, error)
	List(ctx context.Context, filter EventLogListFilter) ([]domain.EventLog, error)
}

type EventLogListFilter struct {
	ResourceType  string
	ResourceID    string
	ActorUserID   string
	Action        string
	Status        string
	MinSequence   int
	MaxSequence   int
	CreatedAfter  time.Time
	CreatedBefore time.Time
}
