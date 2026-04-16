package seller

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

var ErrInvalidCommit = errors.New("invalid seller commit request")
var ErrShopNotFound = errors.New("shop not found")
var ErrShopOwnership = errors.New("shop does not belong to the authenticated seller")
var ErrPledgeNotFound = errors.New("pledge not found")

const PledgeStatusCommitted = "committed"

type CommitInput struct {
	ShopID          string
	ProductID       string
	BundleID        string
	CreatedByUserID string
	Score           float64
	Category        string
	Confidence      float64
	ImageHash       string
	ImageCID        string
}

type CommitService interface {
	Commit(ctx context.Context, input CommitInput) (domain.Pledge, error)
}

type PledgeReader interface {
	GetPledgeForSeller(ctx context.Context, shopID, pledgeID, sellerUserID string) (domain.Pledge, error)
}

type Service struct {
	pledges   repository.PledgeRepository
	shops     repository.ShopRepository
	products  repository.ProductRepository
	audit     AuditLogger
	integrity IntegrityManager
	now       func() time.Time
}

type AuditLogger interface {
	Log(ctx context.Context, input audit.Input) error
}

type IntegrityManager interface {
	PreparePledge(pledge domain.Pledge) (domain.Pledge, error)
	SyncPledge(ctx context.Context, pledge domain.Pledge) (domain.Pledge, error)
}

func NewService(pledges repository.PledgeRepository, shops repository.ShopRepository, products repository.ProductRepository, auditLogger AuditLogger) *Service {
	return &Service{
		pledges:  pledges,
		shops:    shops,
		products: products,
		audit:    auditLogger,
		now:      time.Now,
	}
}

func (s *Service) SetIntegrityManager(manager IntegrityManager) {
	s.integrity = manager
}

func (s *Service) Commit(ctx context.Context, input CommitInput) (domain.Pledge, error) {
	if err := validateCommitInput(input); err != nil {
		return domain.Pledge{}, err
	}
	if s.pledges == nil {
		return domain.Pledge{}, fmt.Errorf("pledge repository is not configured")
	}
	if s.shops == nil {
		return domain.Pledge{}, fmt.Errorf("shop repository is not configured")
	}
	shop, err := s.shops.GetByID(ctx, strings.TrimSpace(input.ShopID))
	if err != nil {
		return domain.Pledge{}, fmt.Errorf("%w: %v", ErrShopNotFound, err)
	}
	if shop.OwnerUserID != strings.TrimSpace(input.CreatedByUserID) {
		return domain.Pledge{}, ErrShopOwnership
	}
	productID := strings.TrimSpace(input.ProductID)
	if productID != "" {
		if s.products == nil {
			return domain.Pledge{}, fmt.Errorf("product repository is not configured")
		}
		product, err := s.products.GetByID(ctx, productID)
		if err != nil {
			return domain.Pledge{}, fmt.Errorf("%w: %v", ErrInvalidCommit, err)
		}
		if product.ShopID != strings.TrimSpace(input.ShopID) {
			return domain.Pledge{}, fmt.Errorf("%w: productId does not belong to shop", ErrInvalidCommit)
		}
	}

	now := s.now().UTC()
	pledge := domain.Pledge{
		PledgeID:        uuid.NewString(),
		ShopID:          strings.TrimSpace(input.ShopID),
		ProductID:       productID,
		BundleID:        strings.TrimSpace(input.BundleID),
		CreatedByUserID: strings.TrimSpace(input.CreatedByUserID),
		Status:          PledgeStatusCommitted,
		Version:         1,
		Score:           input.Score,
		Category:        strings.TrimSpace(input.Category),
		Confidence:      input.Confidence,
		ImageHash:       strings.TrimSpace(input.ImageHash),
		ImageCID:        strings.TrimSpace(input.ImageCID),
		CommittedAt:     now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if s.integrity != nil {
		prepared, err := s.integrity.PreparePledge(pledge)
		if err != nil {
			return domain.Pledge{}, err
		}
		pledge = prepared
	}

	if err := s.pledges.Save(ctx, pledge); err != nil {
		return domain.Pledge{}, err
	}
	if s.integrity != nil {
		anchored, err := s.integrity.SyncPledge(ctx, pledge)
		if err == nil {
			pledge = anchored
			if saveErr := s.pledges.Save(ctx, pledge); saveErr != nil {
				return domain.Pledge{}, saveErr
			}
		}
	}
	if s.audit != nil {
		if err := s.audit.Log(ctx, audit.Input{
			ActorUserID:     pledge.CreatedByUserID,
			ResourceType:    "pledge",
			ResourceID:      pledge.PledgeID,
			ResourceVersion: pledge.Version,
			Action:          "pledge.committed",
			Status:          "committed",
			Payload:         audit.MutationPayload{After: pledge},
		}); err != nil {
			return domain.Pledge{}, err
		}
	}

	return pledge, nil
}

func (s *Service) GetPledgeForSeller(ctx context.Context, shopID, pledgeID, sellerUserID string) (domain.Pledge, error) {
	shopID = strings.TrimSpace(shopID)
	pledgeID = strings.TrimSpace(pledgeID)
	sellerUserID = strings.TrimSpace(sellerUserID)
	if shopID == "" || pledgeID == "" || sellerUserID == "" {
		return domain.Pledge{}, fmt.Errorf("%w: shopId, pledgeId and sellerUserId are required", ErrInvalidCommit)
	}
	if s.pledges == nil {
		return domain.Pledge{}, fmt.Errorf("pledge repository is not configured")
	}
	if s.shops == nil {
		return domain.Pledge{}, fmt.Errorf("shop repository is not configured")
	}

	shop, err := s.shops.GetByID(ctx, shopID)
	if err != nil {
		return domain.Pledge{}, fmt.Errorf("%w: %v", ErrShopNotFound, err)
	}
	if shop.OwnerUserID != sellerUserID {
		return domain.Pledge{}, ErrShopOwnership
	}

	pledge, err := s.pledges.GetByID(ctx, pledgeID)
	if err != nil {
		return domain.Pledge{}, fmt.Errorf("%w: %v", ErrPledgeNotFound, err)
	}
	if strings.TrimSpace(pledge.PledgeID) == "" || pledge.ShopID != shopID {
		return domain.Pledge{}, ErrPledgeNotFound
	}
	return pledge, nil
}

func validateCommitInput(input CommitInput) error {
	if strings.TrimSpace(input.ShopID) == "" {
		return fmt.Errorf("%w: shopId is required", ErrInvalidCommit)
	}
	if strings.TrimSpace(input.CreatedByUserID) == "" {
		return fmt.Errorf("%w: createdByUserId is required", ErrInvalidCommit)
	}
	if strings.TrimSpace(input.BundleID) == "" {
		return fmt.Errorf("%w: bundleId is required", ErrInvalidCommit)
	}
	if strings.TrimSpace(input.Category) == "" {
		return fmt.Errorf("%w: category is required", ErrInvalidCommit)
	}
	if strings.TrimSpace(input.ImageHash) == "" {
		return fmt.Errorf("%w: imageHash is required", ErrInvalidCommit)
	}
	if input.Score < 0 || input.Score > 10 {
		return fmt.Errorf("%w: score must be between 0 and 10", ErrInvalidCommit)
	}
	if input.Confidence < 0 || input.Confidence > 1 {
		return fmt.Errorf("%w: confidence must be between 0 and 1", ErrInvalidCommit)
	}

	return nil
}
