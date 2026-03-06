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
	shopservice "vngrocery/internal/service/shop"
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
		DocsHandler:   handler.NewDocsHandler(),
		AuthHandler:   handler.NewAuthHandler(authAccountsStub{}),
		SellerHandler: handler.NewSellerHandler(sellerScorerStub{}, sellerCommitStub{}),
		BuyerHandler:  handler.NewBuyerHandler(buyerCheckRouteStub{}),
		ShopHandler:   handler.NewShopHandler(shopHandlerStub{}),
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
		DocsHandler:   handler.NewDocsHandler(),
		AuthHandler:   handler.NewAuthHandler(authAccountsStub{}),
		SellerHandler: handler.NewSellerHandler(sellerScorerStub{}, sellerCommitStub{}),
		BuyerHandler:  handler.NewBuyerHandler(buyerCheckRouteStub{}),
		ShopHandler:   handler.NewShopHandler(shopHandlerStub{}),
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
		DocsHandler:   handler.NewDocsHandler(),
		AuthHandler:   handler.NewAuthHandler(authAccountsStub{}),
		SellerHandler: handler.NewSellerHandler(sellerScorerStub{}, sellerCommitStub{}),
		BuyerHandler:  handler.NewBuyerHandler(buyerCheckRouteStub{}),
		ShopHandler:   handler.NewShopHandler(shopHandlerStub{}),
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
		DocsHandler:   handler.NewDocsHandler(),
		AuthHandler:   handler.NewAuthHandler(authAccountsStub{}),
		SellerHandler: handler.NewSellerHandler(sellerScorerStub{}, sellerCommitStub{}),
		BuyerHandler:  handler.NewBuyerHandler(buyerCheckRouteStub{}),
		ShopHandler:   handler.NewShopHandler(shopHandlerStub{}),
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
		DocsHandler:   handler.NewDocsHandler(),
		AuthHandler:   handler.NewAuthHandler(authAccountsStub{}),
		SellerHandler: handler.NewSellerHandler(sellerScorerStub{}, sellerCommitStub{}),
		BuyerHandler:  handler.NewBuyerHandler(buyerCheckRouteStub{}),
		ShopHandler:   handler.NewShopHandler(shopHandlerStub{}),
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
		DocsHandler:   handler.NewDocsHandler(),
		AuthHandler:   handler.NewAuthHandler(authAccountsStub{}),
		SellerHandler: handler.NewSellerHandler(sellerScorerStub{}, sellerCommitStub{}),
		BuyerHandler:  handler.NewBuyerHandler(buyerCheckRouteStub{}),
		ShopHandler:   handler.NewShopHandler(shopHandlerStub{}),
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

func TestRouterShopCreateProtected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(Dependencies{
		HealthHandler: handler.NewHealthHandler(),
		DocsHandler:   handler.NewDocsHandler(),
		AuthHandler:   handler.NewAuthHandler(authAccountsStub{}),
		SellerHandler: handler.NewSellerHandler(sellerScorerStub{}, sellerCommitStub{}),
		BuyerHandler:  handler.NewBuyerHandler(buyerCheckRouteStub{}),
		ShopHandler:   handler.NewShopHandler(shopHandlerStub{}),
		AuthMiddleware: middleware.NewAuthRequired(testVerifier{
			verify: func(ctx context.Context, token string) (authservice.Principal, error) {
				return authservice.Principal{}, authservice.ErrUnauthorized
			},
		}),
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/shops", bytes.NewBufferString(`{"name":"Green Shop","description":"Fresh daily","address":"123 Main St","latitude":10.7,"longitude":106.6}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRouterShopListPublic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(Dependencies{
		HealthHandler: handler.NewHealthHandler(),
		DocsHandler:   handler.NewDocsHandler(),
		AuthHandler:   handler.NewAuthHandler(authAccountsStub{}),
		SellerHandler: handler.NewSellerHandler(sellerScorerStub{}, sellerCommitStub{}),
		BuyerHandler:  handler.NewBuyerHandler(buyerCheckRouteStub{}),
		ShopHandler:   handler.NewShopHandler(shopHandlerStub{}),
		AuthMiddleware: middleware.NewAuthRequired(testVerifier{
			verify: func(ctx context.Context, token string) (authservice.Principal, error) {
				return authservice.Principal{}, authservice.ErrUnauthorized
			},
		}),
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/shops", nil)
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRouterShopReviewCreateProtected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(Dependencies{
		HealthHandler: handler.NewHealthHandler(),
		DocsHandler:   handler.NewDocsHandler(),
		AuthHandler:   handler.NewAuthHandler(authAccountsStub{}),
		SellerHandler: handler.NewSellerHandler(sellerScorerStub{}, sellerCommitStub{}),
		BuyerHandler:  handler.NewBuyerHandler(buyerCheckRouteStub{}),
		ShopHandler:   handler.NewShopHandler(shopHandlerStub{}),
		AuthMiddleware: middleware.NewAuthRequired(testVerifier{
			verify: func(ctx context.Context, token string) (authservice.Principal, error) {
				return authservice.Principal{}, authservice.ErrUnauthorized
			},
		}),
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/shops/shop-1/reviews", bytes.NewBufferString(`{"rating":5,"comment":"Great shop"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRouterShopReviewListPublic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(Dependencies{
		HealthHandler: handler.NewHealthHandler(),
		DocsHandler:   handler.NewDocsHandler(),
		AuthHandler:   handler.NewAuthHandler(authAccountsStub{}),
		SellerHandler: handler.NewSellerHandler(sellerScorerStub{}, sellerCommitStub{}),
		BuyerHandler:  handler.NewBuyerHandler(buyerCheckRouteStub{}),
		ShopHandler:   handler.NewShopHandler(shopHandlerStub{}),
		AuthMiddleware: middleware.NewAuthRequired(testVerifier{
			verify: func(ctx context.Context, token string) (authservice.Principal, error) {
				return authservice.Principal{}, authservice.ErrUnauthorized
			},
		}),
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/shops/shop-1/reviews", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRouterDocsPublic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(Dependencies{
		HealthHandler: handler.NewHealthHandler(),
		DocsHandler:   handler.NewDocsHandler(),
		AuthHandler:   handler.NewAuthHandler(authAccountsStub{}),
		SellerHandler: handler.NewSellerHandler(sellerScorerStub{}, sellerCommitStub{}),
		BuyerHandler:  handler.NewBuyerHandler(buyerCheckRouteStub{}),
		ShopHandler:   handler.NewShopHandler(shopHandlerStub{}),
		AuthMiddleware: middleware.NewAuthRequired(testVerifier{
			verify: func(ctx context.Context, token string) (authservice.Principal, error) {
				return authservice.Principal{}, authservice.ErrUnauthorized
			},
		}),
	})

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRouterOpenAPIPublic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(Dependencies{
		HealthHandler: handler.NewHealthHandler(),
		DocsHandler:   handler.NewDocsHandler(),
		AuthHandler:   handler.NewAuthHandler(authAccountsStub{}),
		SellerHandler: handler.NewSellerHandler(sellerScorerStub{}, sellerCommitStub{}),
		BuyerHandler:  handler.NewBuyerHandler(buyerCheckRouteStub{}),
		ShopHandler:   handler.NewShopHandler(shopHandlerStub{}),
		AuthMiddleware: middleware.NewAuthRequired(testVerifier{
			verify: func(ctx context.Context, token string) (authservice.Principal, error) {
				return authservice.Principal{}, authservice.ErrUnauthorized
			},
		}),
	})

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
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
		PolicyVersion:    "trust_policy_v1",
		PledgeID:         input.PledgeID,
		Trusted:          true,
		Verdict:          "trusted",
		PledgedScore:     8.4,
		ActualScore:      8.0,
		ScoreDelta:       -0.4,
		ScoreDeltaAbs:    0.4,
		PledgedCategory:  "fresh_produce",
		ActualCategory:   "fresh_produce",
		ActualConfidence: 0.9,
		CategoryMatch:    true,
		Reasons:          []string{},
	}, nil
}

type authAccountsStub struct{}

func (authAccountsStub) Register(ctx context.Context, email, password, displayName string) (string, authservice.Principal, string, error) {
	return "", authservice.Principal{}, "", errors.New("not implemented")
}

func (authAccountsStub) Login(ctx context.Context, email, password string) (string, authservice.Principal, string, error) {
	return "", authservice.Principal{}, "", errors.New("not implemented")
}

func (authAccountsStub) GoogleLogin(ctx context.Context, googleIDToken string) (string, authservice.Principal, string, error) {
	return "", authservice.Principal{}, "", errors.New("not implemented")
}

func (authAccountsStub) Delete(ctx context.Context, userID string) (authservice.DeleteResult, error) {
	return authservice.DeleteResult{}, errors.New("not implemented")
}

type shopHandlerStub struct{}

func (shopHandlerStub) Create(ctx context.Context, input shopservice.CreateInput) (domain.Shop, error) {
	return domain.Shop{
		ShopID:      "shop-1",
		OwnerUserID: input.OwnerUserID,
		Name:        input.Name,
		Description: input.Description,
		Address:     input.Address,
		Latitude:    input.Latitude,
		Longitude:   input.Longitude,
		Status:      shopservice.ShopStatusActive,
	}, nil
}

func (shopHandlerStub) Moderate(ctx context.Context, input shopservice.ModerateInput) (domain.Shop, error) {
	return domain.Shop{
		ShopID:            input.ShopID,
		Status:            input.Status,
		ModeratedByUserID: input.ModeratorUserID,
		ModerationNote:    input.ModerationNote,
	}, nil
}

func (shopHandlerStub) Update(ctx context.Context, input shopservice.UpdateInput) (domain.Shop, error) {
	return domain.Shop{
		ShopID:      input.ShopID,
		OwnerUserID: input.OwnerUserID,
		Name:        input.Name,
		Description: input.Description,
		Address:     input.Address,
		Latitude:    input.Latitude,
		Longitude:   input.Longitude,
		Status:      shopservice.ShopStatusActive,
	}, nil
}

func (shopHandlerStub) Delete(ctx context.Context, input shopservice.DeleteInput) (domain.Shop, error) {
	return domain.Shop{
		ShopID:      input.ShopID,
		OwnerUserID: input.OwnerUserID,
		Status:      shopservice.ShopStatusDeleted,
	}, nil
}

func (shopHandlerStub) GetByID(ctx context.Context, shopID string) (shopservice.ShopView, error) {
	return shopservice.ShopView{
		Shop: domain.Shop{
			ShopID:      shopID,
			OwnerUserID: "user-1",
			Name:        "Green Shop",
			Description: "Fresh daily",
			Address:     "123 Main St",
			Latitude:    10.7,
			Longitude:   106.6,
			Status:      shopservice.ShopStatusActive,
		},
	}, nil
}

func (shopHandlerStub) List(ctx context.Context, input shopservice.ListInput) (shopservice.ListResult, error) {
	return shopservice.ListResult{
		Items: []shopservice.ShopView{
			{
				Shop: domain.Shop{
					ShopID:      "shop-1",
					OwnerUserID: "user-1",
					Name:        "Green Shop",
					Description: "Fresh daily",
					Address:     "123 Main St",
					Latitude:    10.7,
					Longitude:   106.6,
					Status:      shopservice.ShopStatusActive,
				},
			},
		},
		Page:     1,
		PageSize: 20,
		Total:    1,
	}, nil
}

func (shopHandlerStub) Review(ctx context.Context, input shopservice.ReviewInput) (domain.ShopReview, error) {
	return domain.ShopReview{
		ReviewID:       "review-1",
		ShopID:         input.ShopID,
		ReviewerUserID: input.ReviewerUserID,
		Rating:         input.Rating,
		Comment:        input.Comment,
	}, nil
}

func (shopHandlerStub) ListReviews(ctx context.Context, shopID string) ([]domain.ShopReview, error) {
	return []domain.ShopReview{
		{
			ReviewID:       "review-1",
			ShopID:         shopID,
			ReviewerUserID: "user-1",
			Rating:         4,
			Comment:        "Good",
		},
	}, nil
}
