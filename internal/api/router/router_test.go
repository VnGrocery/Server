package router

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/handler"
	"vngrocery/internal/api/middleware"
	"vngrocery/internal/domain"
	authservice "vngrocery/internal/service/auth"
	buyerservice "vngrocery/internal/service/buyer"
	sellerservice "vngrocery/internal/service/seller"
	visionservice "vngrocery/internal/service/vision"
)

type testVerifier struct {
	verify func(ctx context.Context, token string) (authservice.Principal, error)
}

func (t testVerifier) Verify(ctx context.Context, token string) (authservice.Principal, error) {
	return t.verify(ctx, token)
}

func TestRouterHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(Dependencies{
		HealthHandler: handler.NewHealthHandler(),
		AuthHandler:   handler.NewAuthHandler(authAccountsStub{}),
		SellerHandler: handler.NewSellerHandler(sellerScorerStub{}, sellerCommitStub{}),
		BuyerHandler:  handler.NewBuyerHandler(buyerCheckRouteStub{}),
		AuthMiddleware: middleware.NewAuthRequired(testVerifier{
			verify: func(ctx context.Context, token string) (authservice.Principal, error) {
				return authservice.Principal{}, nil
			},
		}),
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRouterMeUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(Dependencies{
		HealthHandler: handler.NewHealthHandler(),
		AuthHandler:   handler.NewAuthHandler(authAccountsStub{}),
		SellerHandler: handler.NewSellerHandler(sellerScorerStub{}, sellerCommitStub{}),
		BuyerHandler:  handler.NewBuyerHandler(buyerCheckRouteStub{}),
		AuthMiddleware: middleware.NewAuthRequired(testVerifier{
			verify: func(ctx context.Context, token string) (authservice.Principal, error) {
				return authservice.Principal{}, authservice.ErrUnauthorized
			},
		}),
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRouterMeAuthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(Dependencies{
		HealthHandler: handler.NewHealthHandler(),
		AuthHandler:   handler.NewAuthHandler(authAccountsStub{}),
		SellerHandler: handler.NewSellerHandler(sellerScorerStub{}, sellerCommitStub{}),
		BuyerHandler:  handler.NewBuyerHandler(buyerCheckRouteStub{}),
		AuthMiddleware: middleware.NewAuthRequired(testVerifier{
			verify: func(ctx context.Context, token string) (authservice.Principal, error) {
				return authservice.Principal{UserID: "user-123", Email: "u@example.com"}, nil
			},
		}),
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	req.Header.Set("Authorization", "Bearer good-token")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRouterSellerScoreProtected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(Dependencies{
		HealthHandler: handler.NewHealthHandler(),
		AuthHandler:   handler.NewAuthHandler(authAccountsStub{}),
		SellerHandler: handler.NewSellerHandler(sellerScorerStub{}, sellerCommitStub{}),
		BuyerHandler:  handler.NewBuyerHandler(buyerCheckRouteStub{}),
		AuthMiddleware: middleware.NewAuthRequired(testVerifier{
			verify: func(ctx context.Context, token string) (authservice.Principal, error) {
				return authservice.Principal{}, authservice.ErrUnauthorized
			},
		}),
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("image", "shop.jpg")
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	if _, err := part.Write([]byte("fake-image-content")); err != nil {
		t.Fatalf("failed to write body: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/seller/score", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRouterSellerCommitProtected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(Dependencies{
		HealthHandler: handler.NewHealthHandler(),
		AuthHandler:   handler.NewAuthHandler(authAccountsStub{}),
		SellerHandler: handler.NewSellerHandler(sellerScorerStub{}, sellerCommitStub{}),
		BuyerHandler:  handler.NewBuyerHandler(buyerCheckRouteStub{}),
		AuthMiddleware: middleware.NewAuthRequired(testVerifier{
			verify: func(ctx context.Context, token string) (authservice.Principal, error) {
				return authservice.Principal{}, authservice.ErrUnauthorized
			},
		}),
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/seller/commit", bytes.NewBufferString(`{"shopId":"shop-1","score":8.5,"category":"fresh_produce","confidence":0.91}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRouterBuyerCheckPublic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(Dependencies{
		HealthHandler: handler.NewHealthHandler(),
		AuthHandler:   handler.NewAuthHandler(authAccountsStub{}),
		SellerHandler: handler.NewSellerHandler(sellerScorerStub{}, sellerCommitStub{}),
		BuyerHandler:  handler.NewBuyerHandler(buyerCheckRouteStub{}),
		AuthMiddleware: middleware.NewAuthRequired(testVerifier{
			verify: func(ctx context.Context, token string) (authservice.Principal, error) {
				return authservice.Principal{}, authservice.ErrUnauthorized
			},
		}),
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("pledgeId", "pledge-1"); err != nil {
		t.Fatalf("failed to write pledgeId: %v", err)
	}
	part, err := writer.CreateFormFile("image", "shop.jpg")
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	if _, err := part.Write([]byte("fake-image-content")); err != nil {
		t.Fatalf("failed to write body: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/buyer/check", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

type sellerScorerStub struct{}

func (sellerScorerStub) Score(ctx context.Context, input visionservice.ImageInput) (visionservice.ScoreResult, error) {
	return visionservice.ScoreResult{
		Score:      8.3,
		Category:   "fresh_produce",
		Confidence: 0.91,
	}, nil
}

type sellerCommitStub struct{}

func (sellerCommitStub) Commit(ctx context.Context, input sellerservice.CommitInput) (domain.Pledge, error) {
	return domain.Pledge{
		PledgeID:        "pledge-1",
		ShopID:          input.ShopID,
		CreatedByUserID: input.CreatedByUserID,
		Status:          sellerservice.PledgeStatusCommitted,
		Score:           input.Score,
		Category:        input.Category,
		Confidence:      input.Confidence,
	}, nil
}

type buyerCheckRouteStub struct{}

func (buyerCheckRouteStub) Check(ctx context.Context, input buyerservice.CheckInput) (buyerservice.CheckResult, error) {
	return buyerservice.CheckResult{
		PledgeID:         input.PledgeID,
		Trusted:          true,
		Verdict:          "trusted",
		PledgedScore:     8.4,
		ActualScore:      8.0,
		ScoreDelta:       -0.4,
		PledgedCategory:  "fresh_produce",
		ActualCategory:   "fresh_produce",
		ActualConfidence: 0.9,
	}, nil
}

type authAccountsStub struct{}

func (authAccountsStub) Register(ctx context.Context, email, password, displayName string) (string, authservice.Principal, error) {
	return "", authservice.Principal{}, errors.New("not implemented")
}

func (authAccountsStub) Login(ctx context.Context, email, password string) (string, authservice.Principal, error) {
	return "", authservice.Principal{}, errors.New("not implemented")
}

func (authAccountsStub) GoogleLogin(ctx context.Context, googleIDToken string) (string, authservice.Principal, error) {
	return "", authservice.Principal{}, errors.New("not implemented")
}
