package shop

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	"vngrocery/internal/service/audit"
)

var (
	ErrInvalidShop       = errors.New("invalid shop request")
	ErrForbidden         = errors.New("forbidden")
	ErrNotFound          = errors.New("shop not found")
	ErrAdminRequired     = errors.New("admin role is required")
	ErrVersionConflict   = errors.New("version conflict")
	ErrShopAlreadyExists = errors.New("account already owns a shop")
)

const (
	ShopStatusActive        = "active"
	ShopStatusPendingReview = "pending_review"
	ShopStatusSuspended     = "suspended"
	ShopStatusRejected      = "rejected"
	ShopStatusDeleted       = "deleted"
	ReviewStatusActive      = "active"
	defaultPage             = 1
	defaultPageSize         = 20
	maxPageSize             = 100
	trustedDeltaThreshold   = 1.0
	warningDeltaThreshold   = 2.5
)

type CreateInput struct {
	OwnerUserID string
	Name        string
	Description string
	Address     string
	Latitude    float64
	Longitude   float64
}

type UpdateInput struct {
	ShopID          string
	OwnerUserID     string
	ExpectedVersion int
	Name            string
	Description     string
	Address         string
	Latitude        float64
	Longitude       float64
}

type ModerateInput struct {
	ShopID          string
	ModeratorUserID string
	ExpectedVersion int
	Status          string
	ModerationNote  string
}

type DeleteInput struct {
	ShopID          string
	OwnerUserID     string
	ExpectedVersion int
}

type PledgeHistoryInput struct {
	ShopID    string
	ProductID string
	Category  string
}

type PledgeIntegrityInput struct {
	ShopID   string
	PledgeID string
	DataHash string
}

type PledgeProofBundle struct {
	PledgeID           string
	ShopID             string
	ProductID          string
	BatchID            string
	BundleID           string
	Score              float64
	Category           string
	Confidence         float64
	CommittedAt        time.Time
	ImageHash          string
	ImageCID           string
	ProofStatus        string
	ProofHeadline      string
	ProofSummary       string
	RecommendedActions []string
	Integrity          PledgeIntegrityView
}

type ModeratePledgeIntegrityInput struct {
	ShopID          string
	PledgeID        string
	ActorUserID     string
	ExpectedVersion int
}

type ListInput struct {
	Page               int
	PageSize           int
	Query              string
	Status             string
	OwnerUserID        string
	ActorUserID        string
	IncludeAllStatuses bool
}

type TrustSummary struct {
	HasPledges         bool
	PledgeCount        int
	LatestPledgeID     string
	LatestPledgeStatus string
	LatestScore        float64
	LatestCategory     string
	LatestConfidence   float64
	LastCommittedAt    *time.Time
	Score              float64
	Grade              string
	FormulaVersion     string
	PledgeScore        float64
	ReviewScore        float64
	BuyerCheckScore    float64
	ConsistencyScore   float64
	RecencyScore       float64
	CoverageScore      float64
	BuyerCheckCount    int
	TrustedCheckCount  int
	HighRiskCheckCount int
	Reasons            []string
}

type RatingSummary struct {
	RatingCount   int
	AverageRating float64
}

type ShopView struct {
	Shop          domain.Shop
	TrustSummary  TrustSummary
	RatingSummary RatingSummary
}

type ListResult struct {
	Items    []ShopView
	Page     int
	PageSize int
	Total    int
	HasNext  bool
}

type Service struct {
	shops         repository.ShopRepository
	pledges       repository.PledgeRepository
	checks        repository.BuyerCheckRepository
	reviews       repository.ShopReviewRepository
	users         repository.UserRepository
	audit         AuditLogger
	integrity     PledgeIntegrityReader
	shopIntegrity ShopIntegrityManager
	now           func() time.Time
}

type AuditLogger interface {
	Log(ctx context.Context, input audit.Input) error
}

type PledgeIntegrityReader interface {
	GetPledgeIntegrity(ctx context.Context, pledge domain.Pledge) (PledgeIntegrityView, error)
	VerifyPledgeHash(ctx context.Context, pledge domain.Pledge, dataHash string) (PledgeIntegrityView, error)
	ReanchorPledge(ctx context.Context, pledge domain.Pledge) (domain.Pledge, error)
	RevokePledge(ctx context.Context, pledge domain.Pledge) (domain.Pledge, error)
}

type ShopIntegrityManager interface {
	PrepareShop(shop domain.Shop) (domain.Shop, error)
	SyncShop(ctx context.Context, shop domain.Shop) (domain.Shop, error)
}

type PledgeIntegrityView struct {
	PledgeID          string
	ShopID            string
	DataHash          string
	ProvidedDataHash  string
	ChainTxHash       string
	ChainBlockNumber  int64
	ChainAnchorStatus string
	ChainAnchorTime   *time.Time
	IntegrityStatus   string
	OnChainMatch      bool
	ProvidedHashMatch bool
	OnChainDataHash   string
	OnChainVersion    int
	OnChainTimestamp  *time.Time
	OnChainPresent    bool
	MismatchReason    string
	LastCheckedAt     *time.Time
	CanReanchor       bool
	CanRevoke         bool
}

func buildProofBundle(pledge domain.Pledge, integrity PledgeIntegrityView) PledgeProofBundle {
	committedAt := pledge.CommittedAt
	if committedAt.IsZero() {
		committedAt = pledge.CreatedAt
	}
	bundle := PledgeProofBundle{
		PledgeID:           pledge.PledgeID,
		ShopID:             pledge.ShopID,
		ProductID:          pledge.ProductID,
		BatchID:            pledge.BatchID,
		BundleID:           pledge.BundleID,
		Score:              pledge.Score,
		Category:           pledge.Category,
		Confidence:         pledge.Confidence,
		CommittedAt:        committedAt,
		ImageHash:          pledge.ImageHash,
		ImageCID:           pledge.ImageCID,
		RecommendedActions: []string{},
		Integrity:          integrity,
	}

	switch {
	case integrity.IntegrityStatus == "mismatch_detected":
		bundle.ProofStatus = "warning"
		bundle.ProofHeadline = "Phat hien sai lech du lieu"
		bundle.ProofSummary = "Du lieu hien tai khong con khop voi ban ghi da duoc neo len blockchain."
		bundle.RecommendedActions = []string{"show_warning", "contact_admin", "consider_reanchor"}
	case integrity.IntegrityStatus == "revoked":
		bundle.ProofStatus = "revoked"
		bundle.ProofHeadline = "Cam ket da bi thu hoi"
		bundle.ProofSummary = "Ban ghi nay da bi thu hoi tren lop integrity va khong con duoc xem la cam ket hop le."
		bundle.RecommendedActions = []string{"hide_trust_badge", "show_revoked_state"}
	case integrity.ChainAnchorStatus != "anchored":
		bundle.ProofStatus = "pending"
		bundle.ProofHeadline = "Dang cho neo len blockchain"
		bundle.ProofSummary = "Cam ket da duoc tao nhung chua hoan tat viec neo hash len blockchain."
		bundle.RecommendedActions = []string{"show_pending_badge", "retry_later"}
	case integrity.OnChainMatch:
		bundle.ProofStatus = "verified"
		bundle.ProofHeadline = "Cam ket da duoc xac thuc"
		bundle.ProofSummary = "Hash du lieu trong co so du lieu trung khop voi ban ghi da duoc neo len blockchain."
		bundle.RecommendedActions = []string{"show_verified_badge"}
	default:
		bundle.ProofStatus = "unknown"
		bundle.ProofHeadline = "Chua xac thuc duoc"
		bundle.ProofSummary = "He thong chua co du thong tin de ket luan trang thai integrity cua cam ket nay."
		bundle.RecommendedActions = []string{"show_neutral_state"}
	}

	if integrity.ProvidedDataHash != "" && !integrity.ProvidedHashMatch {
		bundle.ProofStatus = "warning"
		bundle.ProofHeadline = "Hash doi chieu khong khop"
		bundle.ProofSummary = "Hash duoc cung cap khong trung voi ban ghi pledge hien tai."
		bundle.RecommendedActions = []string{"show_warning", "refresh_record"}
	}

	return bundle
}

func NewService(shops repository.ShopRepository, pledges repository.PledgeRepository, checks repository.BuyerCheckRepository, reviews repository.ShopReviewRepository, users repository.UserRepository, auditLogger AuditLogger) *Service {
	return &Service{
		shops:   shops,
		pledges: pledges,
		checks:  checks,
		reviews: reviews,
		users:   users,
		audit:   auditLogger,
		now:     time.Now,
	}
}

func (s *Service) SetPledgeIntegrityReader(reader PledgeIntegrityReader) {
	s.integrity = reader
}

func (s *Service) SetShopIntegrityManager(manager ShopIntegrityManager) {
	s.shopIntegrity = manager
}

type ReviewInput struct {
	ShopID          string
	ReviewerUserID  string
	ExpectedVersion int
	Rating          int
	Comment         string
}

type DeleteReviewInput struct {
	ShopID          string
	ReviewerUserID  string
	ExpectedVersion int
}

func (s *Service) Create(ctx context.Context, input CreateInput) (domain.Shop, error) {
	if err := validate(input.OwnerUserID, input.Name, input.Address, input.Latitude, input.Longitude); err != nil {
		return domain.Shop{}, err
	}
	if s.shops == nil {
		return domain.Shop{}, fmt.Errorf("shop repository is not configured")
	}

	existing, err := s.shops.List(ctx, repository.ShopListFilter{OwnerUserID: strings.TrimSpace(input.OwnerUserID)})
	if err != nil {
		return domain.Shop{}, err
	}
	for _, shop := range existing {
		if shop.Status != ShopStatusDeleted {
			return domain.Shop{}, ErrShopAlreadyExists
		}
	}
	now := s.now().UTC()
	shop := domain.Shop{
		ShopID:      uuid.NewString(),
		OwnerUserID: strings.TrimSpace(input.OwnerUserID),
		Name:        strings.TrimSpace(input.Name),
		Description: strings.TrimSpace(input.Description),
		Address:     strings.TrimSpace(input.Address),
		Latitude:    input.Latitude,
		Longitude:   input.Longitude,
		Status:      ShopStatusActive,
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if s.shopIntegrity != nil {
		prepared, err := s.shopIntegrity.PrepareShop(shop)
		if err != nil {
			return domain.Shop{}, err
		}
		shop = prepared
	}

	if err := s.shops.Save(ctx, shop); err != nil {
		return domain.Shop{}, err
	}
	if s.shopIntegrity != nil {
		anchored, err := s.shopIntegrity.SyncShop(ctx, shop)
		if err == nil {
			shop = anchored
			if saveErr := s.shops.Save(ctx, shop); saveErr != nil {
				return domain.Shop{}, saveErr
			}
		}
	}
	if err := s.logMutation(ctx, input.OwnerUserID, "shop", shop.ShopID, shop.Version, "shop.created", audit.MutationPayload{
		After: shop,
	}, "created"); err != nil {
		return domain.Shop{}, err
	}

	return shop, nil
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (domain.Shop, error) {
	if strings.TrimSpace(input.ShopID) == "" {
		return domain.Shop{}, fmt.Errorf("%w: shopId is required", ErrInvalidShop)
	}
	if err := validate(input.OwnerUserID, input.Name, input.Address, input.Latitude, input.Longitude); err != nil {
		return domain.Shop{}, err
	}
	if s.shops == nil {
		return domain.Shop{}, fmt.Errorf("shop repository is not configured")
	}

	existing, err := s.shops.GetByID(ctx, input.ShopID)
	if err != nil {
		return domain.Shop{}, fmt.Errorf("%w: %v", ErrNotFound, err)
	}
	before := existing
	if existing.ShopID == "" {
		return domain.Shop{}, ErrNotFound
	}
	if existing.OwnerUserID != strings.TrimSpace(input.OwnerUserID) {
		return domain.Shop{}, ErrForbidden
	}
	if existing.Status == ShopStatusDeleted {
		return domain.Shop{}, ErrNotFound
	}
	if input.ExpectedVersion <= 0 || existing.Version != input.ExpectedVersion {
		return domain.Shop{}, ErrVersionConflict
	}

	existing.Name = strings.TrimSpace(input.Name)
	existing.Description = strings.TrimSpace(input.Description)
	existing.Address = strings.TrimSpace(input.Address)
	existing.Latitude = input.Latitude
	existing.Longitude = input.Longitude
	existing.Version++
	existing.UpdatedAt = s.now().UTC()

	if err := s.shops.Save(ctx, existing); err != nil {
		return domain.Shop{}, err
	}
	if err := s.logMutation(ctx, input.OwnerUserID, "shop", existing.ShopID, existing.Version, "shop.updated", audit.MutationPayload{
		Before: before,
		After:  existing,
	}, "updated"); err != nil {
		return domain.Shop{}, err
	}

	return existing, nil
}

func (s *Service) Delete(ctx context.Context, input DeleteInput) (domain.Shop, error) {
	if strings.TrimSpace(input.ShopID) == "" {
		return domain.Shop{}, fmt.Errorf("%w: shopId is required", ErrInvalidShop)
	}
	if strings.TrimSpace(input.OwnerUserID) == "" {
		return domain.Shop{}, fmt.Errorf("%w: ownerUserId is required", ErrInvalidShop)
	}
	if s.shops == nil {
		return domain.Shop{}, fmt.Errorf("shop repository is not configured")
	}

	existing, err := s.shops.GetByID(ctx, input.ShopID)
	if err != nil {
		return domain.Shop{}, fmt.Errorf("%w: %v", ErrNotFound, err)
	}
	before := existing
	if existing.ShopID == "" || existing.Status == ShopStatusDeleted {
		return domain.Shop{}, ErrNotFound
	}
	if existing.OwnerUserID != strings.TrimSpace(input.OwnerUserID) {
		return domain.Shop{}, ErrForbidden
	}
	if input.ExpectedVersion <= 0 || existing.Version != input.ExpectedVersion {
		return domain.Shop{}, ErrVersionConflict
	}

	existing.Status = ShopStatusDeleted
	existing.Version++
	existing.UpdatedAt = s.now().UTC()
	if err := s.shops.Save(ctx, existing); err != nil {
		return domain.Shop{}, err
	}
	if err := s.logMutation(ctx, input.OwnerUserID, "shop", existing.ShopID, existing.Version, "shop.deleted", audit.MutationPayload{
		Before: before,
		After:  existing,
	}, "deleted"); err != nil {
		return domain.Shop{}, err
	}
	return existing, nil
}

func (s *Service) Moderate(ctx context.Context, input ModerateInput) (domain.Shop, error) {
	if strings.TrimSpace(input.ShopID) == "" {
		return domain.Shop{}, fmt.Errorf("%w: shopId is required", ErrInvalidShop)
	}
	if strings.TrimSpace(input.ModeratorUserID) == "" {
		return domain.Shop{}, fmt.Errorf("%w: moderatorUserId is required", ErrInvalidShop)
	}
	status := strings.TrimSpace(input.Status)
	if !isAllowedModerationStatus(status) {
		return domain.Shop{}, fmt.Errorf("%w: unsupported moderation status", ErrInvalidShop)
	}
	if s.shops == nil || s.users == nil {
		return domain.Shop{}, fmt.Errorf("shop moderation dependencies are not configured")
	}
	if err := s.ensureAdmin(ctx, input.ModeratorUserID); err != nil {
		return domain.Shop{}, err
	}

	existing, err := s.shops.GetByID(ctx, input.ShopID)
	if err != nil {
		return domain.Shop{}, fmt.Errorf("%w: %v", ErrNotFound, err)
	}
	before := existing
	if input.ExpectedVersion <= 0 || existing.Version != input.ExpectedVersion {
		return domain.Shop{}, ErrVersionConflict
	}

	now := s.now().UTC()
	existing.Status = status
	existing.Version++
	existing.ModeratedByUserID = strings.TrimSpace(input.ModeratorUserID)
	existing.ModerationNote = strings.TrimSpace(input.ModerationNote)
	existing.ModeratedAt = &now
	existing.UpdatedAt = now

	if err := s.shops.Save(ctx, existing); err != nil {
		return domain.Shop{}, err
	}
	if err := s.logMutation(ctx, input.ModeratorUserID, "shop", existing.ShopID, existing.Version, "shop.moderated", audit.MutationPayload{
		Before: before,
		After:  existing,
	}, "moderated"); err != nil {
		return domain.Shop{}, err
	}

	return existing, nil
}

func (s *Service) GetByID(ctx context.Context, shopID string) (ShopView, error) {
	if strings.TrimSpace(shopID) == "" {
		return ShopView{}, fmt.Errorf("%w: shopId is required", ErrInvalidShop)
	}
	if s.shops == nil {
		return ShopView{}, fmt.Errorf("shop repository is not configured")
	}
	shop, err := s.shops.GetByID(ctx, strings.TrimSpace(shopID))
	if err != nil {
		return ShopView{}, fmt.Errorf("%w: %v", ErrNotFound, err)
	}
	if shop.Status != ShopStatusActive {
		return ShopView{}, ErrNotFound
	}

	return s.buildShopView(ctx, shop)
}

func (s *Service) Review(ctx context.Context, input ReviewInput) (domain.ShopReview, error) {
	if strings.TrimSpace(input.ShopID) == "" {
		return domain.ShopReview{}, fmt.Errorf("%w: shopId is required", ErrInvalidShop)
	}
	if strings.TrimSpace(input.ReviewerUserID) == "" {
		return domain.ShopReview{}, fmt.Errorf("%w: reviewerUserId is required", ErrInvalidShop)
	}
	if input.Rating < 1 || input.Rating > 5 {
		return domain.ShopReview{}, fmt.Errorf("%w: rating must be between 1 and 5", ErrInvalidShop)
	}
	if s.shops == nil || s.reviews == nil {
		return domain.ShopReview{}, fmt.Errorf("shop review dependencies are not configured")
	}

	shop, err := s.shops.GetByID(ctx, strings.TrimSpace(input.ShopID))
	if err != nil {
		return domain.ShopReview{}, fmt.Errorf("%w: %v", ErrNotFound, err)
	}
	if shop.Status != ShopStatusActive {
		return domain.ShopReview{}, fmt.Errorf("%w: shop is not open for reviews", ErrInvalidShop)
	}

	now := s.now().UTC()
	existing, err := s.reviews.GetByShopAndUser(ctx, strings.TrimSpace(input.ShopID), strings.TrimSpace(input.ReviewerUserID))
	if err != nil {
		return domain.ShopReview{}, err
	}
	if existing.ReviewID != "" {
		if existing.Version != input.ExpectedVersion {
			return domain.ShopReview{}, ErrVersionConflict
		}
		before := existing
		existing.Rating = input.Rating
		existing.Comment = strings.TrimSpace(input.Comment)
		existing.Status = ReviewStatusActive
		existing.Version++
		existing.UpdatedAt = now
		if err := s.reviews.Save(ctx, existing); err != nil {
			return domain.ShopReview{}, err
		}
		if err := s.logMutation(ctx, input.ReviewerUserID, "shop_review", existing.ReviewID, existing.Version, "shop_review.updated", audit.MutationPayload{
			Before: before,
			After:  existing,
		}, "updated"); err != nil {
			return domain.ShopReview{}, err
		}
		return existing, nil
	}

	review := domain.ShopReview{
		ReviewID:       uuid.NewString(),
		ShopID:         strings.TrimSpace(input.ShopID),
		ReviewerUserID: strings.TrimSpace(input.ReviewerUserID),
		Rating:         input.Rating,
		Comment:        strings.TrimSpace(input.Comment),
		Status:         ReviewStatusActive,
		Version:        1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.reviews.Save(ctx, review); err != nil {
		return domain.ShopReview{}, err
	}
	if err := s.logMutation(ctx, input.ReviewerUserID, "shop_review", review.ReviewID, review.Version, "shop_review.created", audit.MutationPayload{
		After: review,
	}, "created"); err != nil {
		return domain.ShopReview{}, err
	}
	return review, nil
}

func (s *Service) DeleteReview(ctx context.Context, input DeleteReviewInput) (domain.ShopReview, error) {
	if strings.TrimSpace(input.ShopID) == "" {
		return domain.ShopReview{}, fmt.Errorf("%w: shopId is required", ErrInvalidShop)
	}
	if strings.TrimSpace(input.ReviewerUserID) == "" {
		return domain.ShopReview{}, fmt.Errorf("%w: reviewerUserId is required", ErrInvalidShop)
	}
	if input.ExpectedVersion <= 0 {
		return domain.ShopReview{}, ErrVersionConflict
	}
	if s.reviews == nil {
		return domain.ShopReview{}, fmt.Errorf("shop review repository is not configured")
	}
	existing, err := s.reviews.GetByShopAndUser(ctx, strings.TrimSpace(input.ShopID), strings.TrimSpace(input.ReviewerUserID))
	if err != nil || existing.ReviewID == "" || existing.Status != ReviewStatusActive {
		return domain.ShopReview{}, ErrNotFound
	}
	if existing.Version != input.ExpectedVersion {
		return domain.ShopReview{}, ErrVersionConflict
	}

	before := existing
	existing.Status = ShopStatusDeleted
	existing.Version++
	existing.UpdatedAt = s.now().UTC()
	if err := s.reviews.Save(ctx, existing); err != nil {
		return domain.ShopReview{}, err
	}
	if err := s.logMutation(ctx, input.ReviewerUserID, "shop_review", existing.ReviewID, existing.Version, "shop_review.deleted", audit.MutationPayload{
		Before: before,
		After:  existing,
	}, "deleted"); err != nil {
		return domain.ShopReview{}, err
	}
	return existing, nil
}

func (s *Service) logMutation(ctx context.Context, actorUserID, resourceType, resourceID string, resourceVersion int, action string, payload any, status string) error {
	if s.audit == nil {
		return nil
	}
	return s.audit.Log(ctx, audit.Input{
		ActorUserID:     strings.TrimSpace(actorUserID),
		ResourceType:    resourceType,
		ResourceID:      resourceID,
		ResourceVersion: resourceVersion,
		Action:          action,
		Status:          status,
		Payload:         payload,
	})
}

func (s *Service) ListReviews(ctx context.Context, shopID string) ([]domain.ShopReview, error) {
	if strings.TrimSpace(shopID) == "" {
		return nil, fmt.Errorf("%w: shopId is required", ErrInvalidShop)
	}
	if s.reviews == nil {
		return nil, fmt.Errorf("shop review repository is not configured")
	}
	reviews, err := s.reviews.ListByShopID(ctx, strings.TrimSpace(shopID))
	if err != nil {
		return nil, err
	}
	active := make([]domain.ShopReview, 0, len(reviews))
	for _, review := range reviews {
		if review.Status == ReviewStatusActive {
			active = append(active, review)
		}
	}
	return active, nil
}

func (s *Service) List(ctx context.Context, input ListInput) (ListResult, error) {
	if s.shops == nil {
		return ListResult{}, fmt.Errorf("shop repository is not configured")
	}

	page, pageSize := normalizePagination(input.Page, input.PageSize)
	filter := repository.ShopListFilter{
		OwnerUserID: strings.TrimSpace(input.OwnerUserID),
	}
	if input.IncludeAllStatuses {
		if strings.TrimSpace(input.ActorUserID) == "" {
			return ListResult{}, ErrAdminRequired
		}
		if s.users == nil {
			return ListResult{}, fmt.Errorf("user repository is not configured")
		}
		if err := s.ensureAdmin(ctx, input.ActorUserID); err != nil {
			return ListResult{}, err
		}
		filter.Status = strings.TrimSpace(input.Status)
	} else {
		status := strings.TrimSpace(input.Status)
		if status == "" {
			filter.Status = ShopStatusActive
		} else {
			if status != ShopStatusActive {
				return ListResult{}, fmt.Errorf("%w: public shop listing only supports active status", ErrInvalidShop)
			}
			filter.Status = status
		}
	}

	shops, err := s.shops.List(ctx, filter)
	if err != nil {
		return ListResult{}, err
	}

	query := strings.ToLower(strings.TrimSpace(input.Query))
	filtered := make([]domain.Shop, 0, len(shops))
	for _, shop := range shops {
		if query == "" || matchesQuery(shop, query) {
			filtered = append(filtered, shop)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].UpdatedAt.After(filtered[j].UpdatedAt)
	})

	total := len(filtered)
	start := (page - 1) * pageSize
	if start >= total {
		return ListResult{
			Items:    []ShopView{},
			Page:     page,
			PageSize: pageSize,
			Total:    total,
			HasNext:  false,
		}, nil
	}

	end := start + pageSize
	if end > total {
		end = total
	}

	pageItems := filtered[start:end]
	items := make([]ShopView, 0, len(pageItems))
	for _, shop := range pageItems {
		view, err := s.buildShopView(ctx, shop)
		if err != nil {
			return ListResult{}, err
		}
		items = append(items, view)
	}

	return ListResult{
		Items:    items,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
		HasNext:  end < total,
	}, nil
}

func (s *Service) ListPledges(ctx context.Context, input PledgeHistoryInput) ([]domain.Pledge, error) {
	shopID := strings.TrimSpace(input.ShopID)
	if shopID == "" {
		return nil, fmt.Errorf("%w: shopId is required", ErrInvalidShop)
	}
	if s.shops == nil {
		return nil, fmt.Errorf("shop repository is not configured")
	}
	if s.pledges == nil {
		return nil, fmt.Errorf("pledge repository is not configured")
	}

	shop, err := s.shops.GetByID(ctx, shopID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotFound, err)
	}
	if shop.Status == ShopStatusDeleted {
		return nil, ErrNotFound
	}

	pledges, err := s.pledges.ListByShopID(ctx, shopID)
	if err != nil {
		return nil, err
	}

	productID := strings.TrimSpace(input.ProductID)
	category := strings.TrimSpace(input.Category)
	filtered := make([]domain.Pledge, 0, len(pledges))
	for _, pledge := range pledges {
		if productID != "" && pledge.ProductID != productID {
			continue
		}
		if category != "" && !strings.EqualFold(strings.TrimSpace(pledge.Category), category) {
			continue
		}
		filtered = append(filtered, pledge)
	}

	return filtered, nil
}

func (s *Service) GetPledgeIntegrity(ctx context.Context, input PledgeIntegrityInput) (PledgeIntegrityView, error) {
	pledge, err := s.loadPledgeForIntegrity(ctx, input)
	if err != nil {
		return PledgeIntegrityView{}, err
	}
	return s.getPledgeIntegrityForPledge(ctx, pledge, input.DataHash)
}

func (s *Service) getPledgeIntegrityForPledge(ctx context.Context, pledge domain.Pledge, providedDataHash string) (PledgeIntegrityView, error) {
	view := PledgeIntegrityView{
		PledgeID:          pledge.PledgeID,
		ShopID:            pledge.ShopID,
		DataHash:          pledge.DataHash,
		ChainTxHash:       pledge.ChainTxHash,
		ChainBlockNumber:  pledge.ChainBlockNumber,
		ChainAnchorStatus: pledge.ChainAnchorStatus,
		ChainAnchorTime:   pledge.ChainAnchorTime,
		IntegrityStatus:   pledge.IntegrityStatus,
	}
	if s.integrity == nil {
		return view, nil
	}
	if strings.TrimSpace(providedDataHash) != "" {
		return s.integrity.VerifyPledgeHash(ctx, pledge, providedDataHash)
	}
	return s.integrity.GetPledgeIntegrity(ctx, pledge)
}

func (s *Service) loadPledgeForIntegrity(ctx context.Context, input PledgeIntegrityInput) (domain.Pledge, error) {
	if strings.TrimSpace(input.ShopID) == "" {
		return domain.Pledge{}, fmt.Errorf("%w: shopId is required", ErrInvalidShop)
	}
	if strings.TrimSpace(input.PledgeID) == "" {
		return domain.Pledge{}, fmt.Errorf("%w: pledgeId is required", ErrInvalidShop)
	}
	if s.pledges == nil {
		return domain.Pledge{}, fmt.Errorf("pledge repository is not configured")
	}

	pledge, err := s.pledges.GetByID(ctx, strings.TrimSpace(input.PledgeID))
	if err != nil {
		return domain.Pledge{}, fmt.Errorf("%w: %v", ErrNotFound, err)
	}
	if pledge.PledgeID == "" || pledge.ShopID != strings.TrimSpace(input.ShopID) {
		return domain.Pledge{}, ErrNotFound
	}
	return pledge, nil
}

func (s *Service) GetPledgeProof(ctx context.Context, input PledgeIntegrityInput) (PledgeProofBundle, error) {
	pledge, err := s.loadPledgeForIntegrity(ctx, input)
	if err != nil {
		return PledgeProofBundle{}, err
	}

	integrityView, err := s.getPledgeIntegrityForPledge(ctx, pledge, input.DataHash)
	if err != nil {
		return PledgeProofBundle{}, err
	}

	return buildProofBundle(pledge, integrityView), nil
}

func (s *Service) ReanchorPledgeIntegrity(ctx context.Context, input ModeratePledgeIntegrityInput) (domain.Pledge, error) {
	if err := s.ensureAdmin(ctx, input.ActorUserID); err != nil {
		return domain.Pledge{}, err
	}
	if input.ExpectedVersion <= 0 {
		return domain.Pledge{}, ErrVersionConflict
	}
	if s.pledges == nil || s.integrity == nil {
		return domain.Pledge{}, fmt.Errorf("pledge integrity dependencies are not configured")
	}
	pledge, err := s.pledges.GetByID(ctx, strings.TrimSpace(input.PledgeID))
	if err != nil || pledge.PledgeID == "" || pledge.ShopID != strings.TrimSpace(input.ShopID) {
		return domain.Pledge{}, ErrNotFound
	}
	if pledge.Version != input.ExpectedVersion {
		return domain.Pledge{}, ErrVersionConflict
	}
	before := pledge
	updated, err := s.integrity.ReanchorPledge(ctx, pledge)
	if err != nil {
		return domain.Pledge{}, err
	}
	if err := s.pledges.Save(ctx, updated); err != nil {
		return domain.Pledge{}, err
	}
	if s.audit != nil {
		_ = s.audit.Log(ctx, audit.Input{
			ActorUserID:     input.ActorUserID,
			ResourceType:    "pledge",
			ResourceID:      updated.PledgeID,
			ResourceVersion: updated.Version,
			Action:          "pledge.reanchored",
			Status:          updated.IntegrityStatus,
			Payload:         audit.MutationPayload{Before: before, After: updated},
		})
	}
	return updated, nil
}

func (s *Service) RevokePledgeIntegrity(ctx context.Context, input ModeratePledgeIntegrityInput) (domain.Pledge, error) {
	if err := s.ensureAdmin(ctx, input.ActorUserID); err != nil {
		return domain.Pledge{}, err
	}
	if input.ExpectedVersion <= 0 {
		return domain.Pledge{}, ErrVersionConflict
	}
	if s.pledges == nil || s.integrity == nil {
		return domain.Pledge{}, fmt.Errorf("pledge integrity dependencies are not configured")
	}
	pledge, err := s.pledges.GetByID(ctx, strings.TrimSpace(input.PledgeID))
	if err != nil || pledge.PledgeID == "" || pledge.ShopID != strings.TrimSpace(input.ShopID) {
		return domain.Pledge{}, ErrNotFound
	}
	if pledge.Version != input.ExpectedVersion {
		return domain.Pledge{}, ErrVersionConflict
	}
	before := pledge
	updated, err := s.integrity.RevokePledge(ctx, pledge)
	if err != nil {
		return domain.Pledge{}, err
	}
	if err := s.pledges.Save(ctx, updated); err != nil {
		return domain.Pledge{}, err
	}
	if s.audit != nil {
		_ = s.audit.Log(ctx, audit.Input{
			ActorUserID:     input.ActorUserID,
			ResourceType:    "pledge",
			ResourceID:      updated.PledgeID,
			ResourceVersion: updated.Version,
			Action:          "pledge.revoked",
			Status:          updated.IntegrityStatus,
			Payload:         audit.MutationPayload{Before: before, After: updated},
		})
	}
	return updated, nil
}

func (s *Service) buildShopView(ctx context.Context, shop domain.Shop) (ShopView, error) {
	shopView := ShopView{Shop: shop}
	var pledges []domain.Pledge
	var reviews []domain.ShopReview
	var checks []domain.BuyerCheck
	if s.pledges != nil {
		var err error
		pledges, err = s.pledges.ListByShopID(ctx, shop.ShopID)
		if err != nil {
			return ShopView{}, err
		}
		pledges = filterCommittedPledges(pledges)
		if len(pledges) > 0 {
			latest := pledges[0]
			lastCommittedAt := latest.CommittedAt
			if lastCommittedAt.IsZero() {
				lastCommittedAt = latest.CreatedAt
			}
			shopView.TrustSummary = TrustSummary{
				HasPledges:         true,
				PledgeCount:        len(pledges),
				LatestPledgeID:     latest.PledgeID,
				LatestPledgeStatus: latest.Status,
				LatestScore:        latest.Score,
				LatestCategory:     latest.Category,
				LatestConfidence:   latest.Confidence,
				LastCommittedAt:    &lastCommittedAt,
			}
		}
	}
	if s.reviews != nil {
		var err error
		reviews, err = s.reviews.ListByShopID(ctx, shop.ShopID)
		if err != nil {
			return ShopView{}, err
		}
		reviews = filterActiveReviews(reviews)
		if len(reviews) > 0 {
			totalRating := 0
			for _, review := range reviews {
				totalRating += review.Rating
			}
			shopView.RatingSummary = RatingSummary{
				RatingCount:   len(reviews),
				AverageRating: float64(totalRating) / float64(len(reviews)),
			}
		}
	}
	if s.checks != nil {
		var err error
		checks, err = s.checks.ListByShopID(ctx, shop.ShopID)
		if err != nil {
			return ShopView{}, err
		}
		checks = filterEligibleBuyerChecks(checks)
	}
	applyTrustScore(&shopView.TrustSummary, shopView.RatingSummary, pledges, reviews, checks)
	return shopView, nil
}

func filterCommittedPledges(pledges []domain.Pledge) []domain.Pledge {
	filtered := pledges[:0]
	for _, pledge := range pledges {
		if strings.TrimSpace(pledge.Status) == "committed" {
			filtered = append(filtered, pledge)
		}
	}
	return filtered
}

func filterActiveReviews(reviews []domain.ShopReview) []domain.ShopReview {
	filtered := reviews[:0]
	for _, review := range reviews {
		if strings.TrimSpace(review.Status) == ReviewStatusActive {
			filtered = append(filtered, review)
		}
	}
	return filtered
}

func filterEligibleBuyerChecks(checks []domain.BuyerCheck) []domain.BuyerCheck {
	filtered := checks[:0]
	for _, check := range checks {
		switch strings.TrimSpace(check.Status) {
		case "completed", "flagged":
			filtered = append(filtered, check)
		}
	}
	return filtered
}

const trustScoreFormulaVersion = "trust_score_v2"

func applyTrustScore(summary *TrustSummary, rating RatingSummary, pledges []domain.Pledge, reviews []domain.ShopReview, checks []domain.BuyerCheck) {
	pledgeScore, pledgeReasons := calculatePledgeTrustScore(pledges)
	reviewScore, reviewReasons := calculateReviewTrustScore(rating, len(reviews))
	buyerCheckScore, trustedChecks, highRiskChecks, checkReasons := calculateBuyerCheckTrustScore(checks)
	consistencyScore, consistencyReasons := calculateConsistencyScore(pledges, checks)
	recencyScore, recencyReasons := calculateRecencyScore(pledges, reviews, checks)
	coverageScore, coverageReasons := calculateCoverageScore(pledges, reviews, checks)

	weights := []weightedTrustComponent{
		{score: pledgeScore, weight: 0.30, available: len(pledges) > 0},
		{score: reviewScore, weight: 0.20, available: len(reviews) > 0},
		{score: buyerCheckScore, weight: 0.20, available: len(checks) > 0},
		{score: consistencyScore, weight: 0.15, available: len(pledges) > 0 || len(checks) > 0},
		{score: recencyScore, weight: 0.10, available: len(pledges) > 0 || len(reviews) > 0 || len(checks) > 0},
		{score: coverageScore, weight: 0.05, available: true},
	}

	score, reasons := weightedTrustScore(weights)
	reasons = append(reasons, pledgeReasons...)
	reasons = append(reasons, reviewReasons...)
	reasons = append(reasons, checkReasons...)
	reasons = append(reasons, consistencyReasons...)
	reasons = append(reasons, recencyReasons...)
	reasons = append(reasons, coverageReasons...)

	summary.Score = round(score, 1)
	summary.Grade = trustGrade(summary.Score)
	summary.FormulaVersion = trustScoreFormulaVersion
	summary.PledgeScore = round(pledgeScore, 1)
	summary.ReviewScore = round(reviewScore, 1)
	summary.BuyerCheckScore = round(buyerCheckScore, 1)
	summary.ConsistencyScore = round(consistencyScore, 1)
	summary.RecencyScore = round(recencyScore, 1)
	summary.CoverageScore = round(coverageScore, 1)
	summary.BuyerCheckCount = len(checks)
	summary.TrustedCheckCount = trustedChecks
	summary.HighRiskCheckCount = highRiskChecks
	summary.Reasons = uniqueStrings(reasons)
}

type weightedTrustComponent struct {
	score     float64
	weight    float64
	available bool
}

func weightedTrustScore(components []weightedTrustComponent) (float64, []string) {
	totalWeight := 0.0
	total := 0.0
	reasons := make([]string, 0, 3)
	for _, component := range components {
		if !component.available {
			continue
		}
		total += component.score * component.weight
		totalWeight += component.weight
	}
	if totalWeight == 0 {
		return 50, []string{"insufficient_trust_data"}
	}
	if totalWeight < 1 {
		reasons = append(reasons, "partial_trust_data")
	}
	return clamp(total/totalWeight, 0, 100), reasons
}

func calculatePledgeTrustScore(pledges []domain.Pledge) (float64, []string) {
	if len(pledges) == 0 {
		return 50, []string{"no_seller_pledges"}
	}

	total := 0.0
	totalWeight := 0.0
	lowConfidence := 0
	for _, pledge := range pledges {
		scoreComponent := clamp(pledge.Score*10, 0, 100)
		confidenceComponent := clamp(pledge.Confidence*100, 0, 100)
		weight := recencyWeight(pledge.UpdatedAt)
		total += (scoreComponent*0.7 + confidenceComponent*0.3) * weight
		totalWeight += weight
		if pledge.Confidence < 0.60 {
			lowConfidence++
		}
	}

	reasons := make([]string, 0, 2)
	if len(pledges) >= 3 {
		reasons = append(reasons, "seller_has_pledge_history")
	}
	if lowConfidence > 0 {
		reasons = append(reasons, "some_pledges_low_confidence")
	}
	return total / totalWeight, reasons
}

func calculateReviewTrustScore(rating RatingSummary, reviewCount int) (float64, []string) {
	if reviewCount == 0 {
		return 50, []string{"no_customer_reviews"}
	}

	score := clamp(rating.AverageRating/5*100, 0, 100)
	reasons := []string{}
	if reviewCount >= 5 {
		reasons = append(reasons, "review_history_available")
	}
	if rating.AverageRating < 3 {
		reasons = append(reasons, "low_customer_rating")
	}
	return score, reasons
}

func calculateConsistencyScore(pledges []domain.Pledge, checks []domain.BuyerCheck) (float64, []string) {
	if len(pledges) == 0 && len(checks) == 0 {
		return 50, []string{"no_consistency_signals"}
	}

	total := 0.0
	count := 0.0
	for _, check := range checks {
		if check.Status == "rejected" {
			continue
		}
		score := 85.0
		if !check.CategoryMatch {
			score -= 20
		}
		if check.ScoreDeltaAbs > warningDeltaThreshold {
			score -= 25
		} else if check.ScoreDeltaAbs > trustedDeltaThreshold {
			score -= 10
		}
		if check.Verdict == "high_risk" {
			score -= 25
		}
		if check.Status == "flagged" {
			score -= 10
		}
		total += clamp(score, 0, 100)
		count++
	}
	if count == 0 {
		return 70, []string{"limited_consistency_data"}
	}
	reasons := []string{}
	avg := total / count
	if avg >= 80 {
		reasons = append(reasons, "pledges_consistent_with_buyer_checks")
	}
	if avg < 55 {
		reasons = append(reasons, "buyer_checks_show_consistency_issues")
	}
	return avg, reasons
}

func calculateRecencyScore(pledges []domain.Pledge, reviews []domain.ShopReview, checks []domain.BuyerCheck) (float64, []string) {
	latest := latestActivityTime(pledges, reviews, checks)
	if latest.IsZero() {
		return 50, []string{"no_recent_activity"}
	}
	age := time.Since(latest)
	score := 45.0
	switch {
	case age <= 72*time.Hour:
		score = 100
	case age <= 7*24*time.Hour:
		score = 90
	case age <= 14*24*time.Hour:
		score = 78
	case age <= 30*24*time.Hour:
		score = 64
	}
	reasons := []string{}
	if score >= 90 {
		reasons = append(reasons, "recent_activity_available")
	}
	if score <= 50 {
		reasons = append(reasons, "trust_signals_are_stale")
	}
	return score, reasons
}

func calculateCoverageScore(pledges []domain.Pledge, reviews []domain.ShopReview, checks []domain.BuyerCheck) (float64, []string) {
	coverage := 0.0
	reasons := []string{}
	if len(pledges) > 0 {
		coverage += 40
	}
	if len(reviews) > 0 {
		coverage += 30
	}
	if len(checks) > 0 {
		coverage += 30
	}
	if coverage == 100 {
		reasons = append(reasons, "full_trust_signal_coverage")
	} else if coverage < 70 {
		reasons = append(reasons, "limited_signal_coverage")
	}
	return coverage, reasons
}

func latestActivityTime(pledges []domain.Pledge, reviews []domain.ShopReview, checks []domain.BuyerCheck) time.Time {
	var latest time.Time
	for _, pledge := range pledges {
		if pledge.UpdatedAt.After(latest) {
			latest = pledge.UpdatedAt
		}
	}
	for _, review := range reviews {
		if review.UpdatedAt.After(latest) {
			latest = review.UpdatedAt
		}
	}
	for _, check := range checks {
		if check.UpdatedAt.After(latest) {
			latest = check.UpdatedAt
		}
	}
	return latest
}

func recencyWeight(updatedAt time.Time) float64 {
	age := time.Since(updatedAt)
	switch {
	case age <= 72*time.Hour:
		return 1.15
	case age <= 7*24*time.Hour:
		return 1.0
	case age <= 30*24*time.Hour:
		return 0.85
	default:
		return 0.7
	}
}

func calculateBuyerCheckTrustScore(checks []domain.BuyerCheck) (float64, int, int, []string) {
	if len(checks) == 0 {
		return 50, 0, 0, []string{"no_buyer_checks"}
	}

	total := 0.0
	totalWeight := 0.0
	trusted := 0
	highRisk := 0
	duplicateDiscounts := 0
	seenActors := map[string]int{}
	for _, check := range checks {
		if check.Status == "rejected" {
			continue
		}
		checkScore := 70.0
		switch check.Verdict {
		case "trusted":
			checkScore = 100
			trusted++
		case "warning":
			checkScore = 60
		case "high_risk":
			checkScore = 20
			highRisk++
		case "no_pledge":
			checkScore = 50
		}
		if !check.CategoryMatch {
			checkScore -= 15
		}
		if check.ScoreDeltaAbs > warningDeltaThreshold {
			checkScore -= 15
		}
		weight := 1.0
		actorKey := strings.TrimSpace(check.BuyerUserID)
		if actorKey != "" {
			seenActors[actorKey]++
			if seenActors[actorKey] > 1 {
				weight = 0.35
				duplicateDiscounts++
			}
		}
		if check.Status == "flagged" {
			weight *= 0.25
		}
		total += clamp(checkScore, 0, 100) * weight
		totalWeight += weight
	}

	reasons := make([]string, 0, 2)
	if trusted > 0 {
		reasons = append(reasons, "buyer_checks_confirmed")
	}
	if highRisk > 0 {
		reasons = append(reasons, "buyer_checks_high_risk")
	}
	if duplicateDiscounts > 0 {
		reasons = append(reasons, "duplicate_buyer_checks_discounted")
	}
	if totalWeight == 0 {
		return 50, trusted, highRisk, append(reasons, "no_eligible_buyer_checks")
	}
	return total / totalWeight, trusted, highRisk, reasons
}

const warningMaxScoreDelta = 2.5

func trustGrade(score float64) string {
	switch {
	case score >= 85:
		return "excellent"
	case score >= 70:
		return "good"
	case score >= 55:
		return "watch"
	default:
		return "risk"
	}
}

func clamp(value, minValue, maxValue float64) float64 {
	return math.Max(minValue, math.Min(maxValue, value))
}

func round(value float64, precision int) float64 {
	pow := math.Pow(10, float64(precision))
	return math.Round(value*pow) / pow
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func (s *Service) ensureAdmin(ctx context.Context, userID string) error {
	user, err := s.users.GetByID(ctx, strings.TrimSpace(userID))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAdminRequired, err)
	}
	if !strings.EqualFold(strings.TrimSpace(user.Role), "admin") {
		return ErrAdminRequired
	}
	return nil
}

func normalizePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = defaultPage
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

func matchesQuery(shop domain.Shop, query string) bool {
	name := strings.ToLower(shop.Name)
	address := strings.ToLower(shop.Address)
	description := strings.ToLower(shop.Description)
	return strings.Contains(name, query) || strings.Contains(address, query) || strings.Contains(description, query)
}

func isAllowedModerationStatus(status string) bool {
	switch status {
	case ShopStatusActive, ShopStatusPendingReview, ShopStatusSuspended, ShopStatusRejected:
		return true
	default:
		return false
	}
}

func validate(ownerUserID, name, address string, latitude, longitude float64) error {
	if strings.TrimSpace(ownerUserID) == "" {
		return fmt.Errorf("%w: ownerUserId is required", ErrInvalidShop)
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidShop)
	}
	if strings.TrimSpace(address) == "" {
		return fmt.Errorf("%w: address is required", ErrInvalidShop)
	}
	if math.IsNaN(latitude) || latitude < -90 || latitude > 90 {
		return fmt.Errorf("%w: latitude must be between -90 and 90", ErrInvalidShop)
	}
	if math.IsNaN(longitude) || longitude < -180 || longitude > 180 {
		return fmt.Errorf("%w: longitude must be between -180 and 180", ErrInvalidShop)
	}
	return nil
}
