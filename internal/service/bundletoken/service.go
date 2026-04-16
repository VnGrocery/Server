package bundletoken

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
)

var (
	ErrInvalidToken = errors.New("invalid bundle token")
	ErrExpiredToken = errors.New("bundle token expired")
	ErrReplayToken  = errors.New("bundle token already used")
)

const QRVersionV1 = "bundle_qr_v1"

type IssueInput struct {
	ShopID      string
	ProductID   string
	BundleID    string
	PledgeID    string
	CreatedByID string
	CommittedAt time.Time
}

type VerifyInput struct {
	Token            string
	BuyerUserID      string
	ExpectedBundleID string
	ExpectedPledgeID string
}

type Claims struct {
	QRVersion string
	ShopID    string
	ProductID string
	BundleID  string
	PledgeID  string
	Nonce     string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

type Service struct {
	secret []byte
	issuer string
	ttl    time.Duration
	replay repository.BundleTokenUseRepository
	now    func() time.Time
}

func NewService(secret, issuer string, ttl time.Duration, replay repository.BundleTokenUseRepository) *Service {
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	return &Service{
		secret: []byte(strings.TrimSpace(secret)),
		issuer: strings.TrimSpace(issuer),
		ttl:    ttl,
		replay: replay,
		now:    time.Now,
	}
}

func (s *Service) Issue(input IssueInput) (string, time.Time, error) {
	bundleID := strings.TrimSpace(input.BundleID)
	if bundleID == "" {
		return "", time.Time{}, fmt.Errorf("%w: bundleId is required", ErrInvalidToken)
	}
	shopID := strings.TrimSpace(input.ShopID)
	if shopID == "" {
		return "", time.Time{}, fmt.Errorf("%w: shopId is required", ErrInvalidToken)
	}
	if len(s.secret) == 0 {
		return "", time.Time{}, fmt.Errorf("%w: secret is not configured", ErrInvalidToken)
	}

	issuedAt := s.now().UTC()
	expiresAt := issuedAt.Add(s.ttl)
	claims := jwt.MapClaims{
		"iss":         s.issuer,
		"iat":         issuedAt.Unix(),
		"exp":         expiresAt.Unix(),
		"tokenType":   "bundle_qr",
		"qrVersion":   QRVersionV1,
		"nonce":       uuid.NewString(),
		"shopId":      shopID,
		"productId":   strings.TrimSpace(input.ProductID),
		"bundleId":    bundleID,
		"pledgeId":    strings.TrimSpace(input.PledgeID),
		"sellerId":    strings.TrimSpace(input.CreatedByID),
		"committedAt": input.CommittedAt.UTC().Format(time.RFC3339),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign bundle token: %w", err)
	}
	return signed, expiresAt, nil
}

func (s *Service) VerifyAndConsume(ctx context.Context, input VerifyInput) (Claims, error) {
	var out Claims
	tokenRaw := strings.TrimSpace(input.Token)
	if tokenRaw == "" {
		return out, fmt.Errorf("%w: token is required", ErrInvalidToken)
	}
	if len(s.secret) == 0 {
		return out, fmt.Errorf("%w: secret is not configured", ErrInvalidToken)
	}

	parser := jwt.NewParser(jwt.WithTimeFunc(s.now), jwt.WithoutClaimsValidation())
	parsed, err := parser.Parse(tokenRaw, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("%w: unexpected signing method", ErrInvalidToken)
		}
		return s.secret, nil
	})
	if err != nil || !parsed.Valid {
		return out, fmt.Errorf("%w: malformed or signature mismatch", ErrInvalidToken)
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return out, fmt.Errorf("%w: invalid claims", ErrInvalidToken)
	}

	getString := func(key string) string {
		value, ok := claims[key].(string)
		if !ok {
			return ""
		}
		return strings.TrimSpace(value)
	}

	bundleID := getString("bundleId")
	shopID := getString("shopId")
	nonce := getString("nonce")
	pledgeID := getString("pledgeId")
	productID := getString("productId")
	qrVersion := getString("qrVersion")
	if bundleID == "" || shopID == "" || nonce == "" {
		return out, fmt.Errorf("%w: required claims missing", ErrInvalidToken)
	}
	if qrVersion != QRVersionV1 {
		return out, fmt.Errorf("%w: unsupported qrVersion", ErrInvalidToken)
	}

	expiresAt, err := claims.GetExpirationTime()
	if err != nil || expiresAt == nil {
		return out, fmt.Errorf("%w: exp is invalid", ErrInvalidToken)
	}
	exp := expiresAt.Time.UTC()
	if !exp.After(s.now().UTC()) {
		return out, ErrExpiredToken
	}
	issuedAt, err := claims.GetIssuedAt()
	if err != nil || issuedAt == nil {
		return out, fmt.Errorf("%w: iat is invalid", ErrInvalidToken)
	}

	expectedBundleID := strings.TrimSpace(input.ExpectedBundleID)
	if expectedBundleID != "" && bundleID != expectedBundleID {
		return out, fmt.Errorf("%w: bundleId mismatch", ErrInvalidToken)
	}
	expectedPledgeID := strings.TrimSpace(input.ExpectedPledgeID)
	if expectedPledgeID != "" && pledgeID != expectedPledgeID {
		return out, fmt.Errorf("%w: pledgeId mismatch", ErrInvalidToken)
	}

	if s.replay != nil {
		nonceHash := sha256Hex(nonce)
		reserved, err := s.replay.Reserve(ctx, domain.BundleTokenUse{
			UseID:       nonceHash,
			NonceHash:   nonceHash,
			ShopID:      shopID,
			ProductID:   productID,
			BundleID:    bundleID,
			PledgeID:    pledgeID,
			BuyerUserID: strings.TrimSpace(input.BuyerUserID),
			UsedAt:      s.now().UTC(),
			ExpiresAt:   exp,
		})
		if err != nil {
			return out, err
		}
		if !reserved {
			return out, ErrReplayToken
		}
	}

	out = Claims{
		QRVersion: qrVersion,
		ShopID:    shopID,
		ProductID: productID,
		BundleID:  bundleID,
		PledgeID:  pledgeID,
		Nonce:     nonce,
		IssuedAt:  issuedAt.Time.UTC(),
		ExpiresAt: exp,
	}
	return out, nil
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
