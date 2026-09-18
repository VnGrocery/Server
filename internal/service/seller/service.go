package seller

import (
	"context"
	"crypto/rand"
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

	// Why this score. Required: it is hashed with the rest of the pledge and
	// anchored, so a buyer reading the record sees the reasoning, not just a
	// number somebody typed.
	Note string
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
	bundleID, err := s.resolveBundleID(ctx, strings.TrimSpace(input.BundleID), now)
	if err != nil {
		return domain.Pledge{}, err
	}
	pledge := domain.Pledge{
		PledgeID:        uuid.NewString(),
		ShopID:          strings.TrimSpace(input.ShopID),
		ProductID:       productID,
		BundleID:        bundleID,
		CreatedByUserID: strings.TrimSpace(input.CreatedByUserID),
		Status:          PledgeStatusCommitted,
		Version:         1,
		Score:           input.Score,
		Category:        strings.TrimSpace(input.Category),
		Confidence:      input.Confidence,
		ImageHash:       strings.TrimSpace(input.ImageHash),
		ImageCID:        strings.TrimSpace(input.ImageCID),
		Note:            strings.TrimSpace(input.Note),
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

// resolveBundleID returns the lot code to print on the label.
//
// A seller who has their own lot numbering keeps it; the code is only checked
// for not colliding with one already in use. Otherwise one is minted here,
// because the value the app used to send was a per-launch counter.
func (s *Service) resolveBundleID(ctx context.Context, supplied string, now time.Time) (string, error) {
	if supplied != "" {
		if s.taken(ctx, supplied) {
			return "", fmt.Errorf("%w: bundleId %q is already in use", ErrInvalidCommit, supplied)
		}
		return supplied, nil
	}
	// ponytail: check-then-write, so two commits racing on the same code in the
	// same millisecond could both pass. With 40 random bits per attempt that is
	// far rarer than the collisions this replaces; a unique index on bundleId
	// is the real fix if lot codes ever get minted in bulk.
	for attempt := 0; attempt < 5; attempt++ {
		candidate := mintBundleID(now)
		if !s.taken(ctx, candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("could not mint an unused bundleId")
}

func (s *Service) taken(ctx context.Context, bundleID string) bool {
	if s.pledges == nil {
		return false
	}
	_, err := s.pledges.GetByBundleID(ctx, bundleID)
	return err == nil
}

// mintBundleID builds a code a person can read off a crate and say out loud:
// the commit date, then 40 bits of randomness in Crockford base32, which drops
// the letters that get misread as digits.
func mintBundleID(now time.Time) string {
	const alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	var buf [8]byte
	if _, err := rand.Read(buf[:5]); err != nil {
		// Randomness is not optional here; a predictable lot code would let
		// anyone guess a neighbour's label.
		panic(fmt.Sprintf("mint bundle id: %v", err))
	}
	suffix := make([]byte, 8)
	value := uint64(buf[0])<<32 | uint64(buf[1])<<24 | uint64(buf[2])<<16 | uint64(buf[3])<<8 | uint64(buf[4])
	for i := 7; i >= 0; i-- {
		suffix[i] = alphabet[value&31]
		value >>= 5
	}
	return fmt.Sprintf("LO-%s-%s", now.Format("060102"), string(suffix))
}

func validateCommitInput(input CommitInput) error {
	if strings.TrimSpace(input.ShopID) == "" {
		return fmt.Errorf("%w: shopId is required", ErrInvalidCommit)
	}
	if strings.TrimSpace(input.CreatedByUserID) == "" {
		return fmt.Errorf("%w: createdByUserId is required", ErrInvalidCommit)
	}
	// bundleId is deliberately not required. The app used to send a counter
	// from its mock database - g1, g2, restarting at g1 on every launch and
	// colliding across devices - so demanding a value only guaranteed a
	// meaningless one. An empty value is minted server-side instead.
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
	note := strings.TrimSpace(input.Note)
	if len([]rune(note)) < 5 {
		return fmt.Errorf("%w: note is required", ErrInvalidCommit)
	}
	if len([]rune(note)) > 200 {
		return fmt.Errorf("%w: note is too long", ErrInvalidCommit)
	}

	return nil
}
