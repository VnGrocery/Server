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
)

var ErrInvalidCommit = errors.New("invalid seller commit request")
var ErrShopNotFound = errors.New("shop not found")
var ErrShopOwnership = errors.New("shop does not belong to the authenticated seller")

const PledgeStatusCommitted = "committed"

type CommitInput struct {
	ShopID          string
	CreatedByUserID string
	Score           float64
	Category        string
	Confidence      float64
}

type CommitService interface {
	Commit(ctx context.Context, input CommitInput) (domain.Pledge, error)
}

type Service struct {
	pledges repository.PledgeRepository
	shops   repository.ShopRepository
	now     func() time.Time
}

func NewService(pledges repository.PledgeRepository, shops repository.ShopRepository) *Service {
	return &Service{
		pledges: pledges,
		shops:   shops,
		now:     time.Now,
	}
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

	now := s.now().UTC()
	pledge := domain.Pledge{
		PledgeID:        uuid.NewString(),
		ShopID:          strings.TrimSpace(input.ShopID),
		CreatedByUserID: strings.TrimSpace(input.CreatedByUserID),
		Status:          PledgeStatusCommitted,
		Score:           input.Score,
		Category:        strings.TrimSpace(input.Category),
		Confidence:      input.Confidence,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.pledges.Save(ctx, pledge); err != nil {
		return domain.Pledge{}, err
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
	if strings.TrimSpace(input.Category) == "" {
		return fmt.Errorf("%w: category is required", ErrInvalidCommit)
	}
	if input.Score < 0 || input.Score > 10 {
		return fmt.Errorf("%w: score must be between 0 and 10", ErrInvalidCommit)
	}
	if input.Confidence < 0 || input.Confidence > 1 {
		return fmt.Errorf("%w: confidence must be between 0 and 1", ErrInvalidCommit)
	}

	return nil
}
