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
	auditservice "vngrocery/internal/service/audit"
	authservice "vngrocery/internal/service/auth"
	buyerservice "vngrocery/internal/service/buyer"
	productservice "vngrocery/internal/service/product"
	sellerservice "vngrocery/internal/service/seller"
	shopservice "vngrocery/internal/service/shop"
	useradminservice "vngrocery/internal/service/useradmin"
	visionservice "vngrocery/internal/service/vision"
)

func newTestDependencies(verifier testVerifier) Dependencies {
	return Dependencies{
		HealthHandler:    handler.NewHealthHandler(),
		DocsHandler:      handler.NewDocsHandler(),
		AuthHandler:      handler.NewAuthHandler(authAccountsStub{}),
		AdminUserHandler: handler.NewAdminUserHandler(adminUserHandlerStub{}),
		EventLogHandler:  handler.NewEventLogHandler(eventLogUsecaseStub{}),
		ProductHandler:   handler.NewProductHandler(productHandlerStub{}),
		SellerHandler:    handler.NewSellerHandler(sellerScorerStub{}, sellerCommitStub{}),
		BuyerHandler:     handler.NewBuyerHandler(buyerCheckRouteStub{}),
		ShopHandler:      handler.NewShopHandler(shopHandlerStub{}),
		AuthMiddleware:   middleware.NewAuthRequired(verifier),
	}
}

type testVerifier struct {
	verify func(ctx context.Context, token string) (authservice.Principal, error)
}

func (t testVerifier) Verify(ctx context.Context, token string) (authservice.Principal, error) {
	return t.verify(ctx, token)
}

func TestRouterHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(newTestDependencies(testVerifier{
		verify: func(ctx context.Context, token string) (authservice.Principal, error) {
			return authservice.Principal{}, nil
		},
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRouterMeUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(newTestDependencies(testVerifier{
		verify: func(ctx context.Context, token string) (authservice.Principal, error) {
			return authservice.Principal{}, authservice.ErrUnauthorized
		},
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRouterMeAuthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(newTestDependencies(testVerifier{
		verify: func(ctx context.Context, token string) (authservice.Principal, error) {
			return authservice.Principal{UserID: "user-123", Email: "u@example.com"}, nil
		},
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	req.Header.Set("Authorization", "Bearer good-token")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRouterEventsProtected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(newTestDependencies(testVerifier{
		verify: func(ctx context.Context, token string) (authservice.Principal, error) {
			return authservice.Principal{}, authservice.ErrUnauthorized
		},
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/events", nil)
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRouterSellerScoreProtected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(newTestDependencies(testVerifier{
		verify: func(ctx context.Context, token string) (authservice.Principal, error) {
			return authservice.Principal{}, authservice.ErrUnauthorized
		},
	}))

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
	engine := New(newTestDependencies(testVerifier{
		verify: func(ctx context.Context, token string) (authservice.Principal, error) {
			return authservice.Principal{}, authservice.ErrUnauthorized
		},
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/seller/commit", bytes.NewBufferString(`{"shopId":"shop-1","score":8.5,"category":"fresh_produce","confidence":0.91}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRouterBuyerCheckProtected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(newTestDependencies(testVerifier{
		verify: func(ctx context.Context, token string) (authservice.Principal, error) {
			return authservice.Principal{}, authservice.ErrUnauthorized
		},
	}))

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

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRouterShopCreateProtected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(newTestDependencies(testVerifier{
		verify: func(ctx context.Context, token string) (authservice.Principal, error) {
			return authservice.Principal{}, authservice.ErrUnauthorized
		},
	}))

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
	engine := New(newTestDependencies(testVerifier{
		verify: func(ctx context.Context, token string) (authservice.Principal, error) {
			return authservice.Principal{}, authservice.ErrUnauthorized
		},
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/shops", nil)
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRouterShopReviewCreateProtected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(newTestDependencies(testVerifier{
		verify: func(ctx context.Context, token string) (authservice.Principal, error) {
			return authservice.Principal{}, authservice.ErrUnauthorized
		},
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/shops/shop-1/reviews", bytes.NewBufferString(`{"rating":5,"comment":"Great shop"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRouterProductCreateProtected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(newTestDependencies(testVerifier{
		verify: func(ctx context.Context, token string) (authservice.Principal, error) {
			return authservice.Principal{}, authservice.ErrUnauthorized
		},
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/shops/shop-1/products", bytes.NewBufferString(`{"name":"Apple","price":10,"currency":"VND"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRouterProductListPublic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(newTestDependencies(testVerifier{
		verify: func(ctx context.Context, token string) (authservice.Principal, error) {
			return authservice.Principal{}, authservice.ErrUnauthorized
		},
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/shops/shop-1/products", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRouterShopReviewListPublic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(newTestDependencies(testVerifier{
		verify: func(ctx context.Context, token string) (authservice.Principal, error) {
			return authservice.Principal{}, authservice.ErrUnauthorized
		},
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/shops/shop-1/reviews", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRouterAdminUserRoleProtected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(newTestDependencies(testVerifier{
		verify: func(ctx context.Context, token string) (authservice.Principal, error) {
			return authservice.Principal{}, authservice.ErrUnauthorized
		},
	}))

	req := httptest.NewRequest(http.MethodPatch, "/v1/admin/users/user-1/role", bytes.NewBufferString(`{"expectedVersion":1,"role":"admin"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRouterDocsPublic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(newTestDependencies(testVerifier{
		verify: func(ctx context.Context, token string) (authservice.Principal, error) {
			return authservice.Principal{}, authservice.ErrUnauthorized
		},
	}))

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRouterOpenAPIPublic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := New(newTestDependencies(testVerifier{
		verify: func(ctx context.Context, token string) (authservice.Principal, error) {
			return authservice.Principal{}, authservice.ErrUnauthorized
		},
	}))

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
		ProductID:       input.ProductID,
		CreatedByUserID: input.CreatedByUserID,
		Status:          sellerservice.PledgeStatusCommitted,
		Score:           input.Score,
		Category:        input.Category,
		Confidence:      input.Confidence,
		ImageHash:       input.ImageHash,
	}, nil
}

type buyerCheckRouteStub struct{}

func (buyerCheckRouteStub) Check(ctx context.Context, input buyerservice.CheckInput) (buyerservice.CheckResult, error) {
	return buyerservice.CheckResult{
		CheckID:          "check-1",
		ShopID:           "shop-1",
		PolicyVersion:    "trust_policy_v1",
		HasPledge:        input.PledgeID != "",
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

func (authAccountsStub) Register(ctx context.Context, email, password, displayName string) (authservice.AuthResult, error) {
	return authservice.AuthResult{}, errors.New("not implemented")
}

func (authAccountsStub) Login(ctx context.Context, email, password string) (authservice.AuthResult, error) {
	return authservice.AuthResult{}, errors.New("not implemented")
}

func (authAccountsStub) GoogleLogin(ctx context.Context, googleIDToken string) (authservice.AuthResult, error) {
	return authservice.AuthResult{}, errors.New("not implemented")
}

func (authAccountsStub) Refresh(ctx context.Context, refreshToken string) (authservice.AuthResult, error) {
	return authservice.AuthResult{}, errors.New("not implemented")
}

func (authAccountsStub) Logout(ctx context.Context, refreshToken string) error {
	return errors.New("not implemented")
}

func (authAccountsStub) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	return errors.New("not implemented")
}

func (authAccountsStub) RequestPasswordReset(ctx context.Context, email string) (authservice.PasswordResetResult, error) {
	return authservice.PasswordResetResult{}, errors.New("not implemented")
}

func (authAccountsStub) ResetPassword(ctx context.Context, resetToken, newPassword string) error {
	return errors.New("not implemented")
}

func (authAccountsStub) Delete(ctx context.Context, userID string, expectedVersion int) (authservice.DeleteResult, error) {
	return authservice.DeleteResult{}, errors.New("not implemented")
}

type eventLogUsecaseStub struct{}

func (eventLogUsecaseStub) List(ctx context.Context, input auditservice.ListInput) (auditservice.ListResult, error) {
	return auditservice.ListResult{Items: []domain.EventLog{}, Page: 1, PageSize: 50}, nil
}

func (eventLogUsecaseStub) VerifyEvent(ctx context.Context, input auditservice.VerifyEventInput) (auditservice.EventVerificationResult, error) {
	return auditservice.EventVerificationResult{EventID: input.EventID, Verified: true, ContentHashValid: true, SignatureValid: true, ChainLinkValid: true}, nil
}

func (eventLogUsecaseStub) VerifyResource(ctx context.Context, input auditservice.VerifyResourceInput) (auditservice.VerifyResourceResult, error) {
	return auditservice.VerifyResourceResult{ResourceType: input.ResourceType, ResourceID: input.ResourceID, Verified: true}, nil
}

type adminUserHandlerStub struct{}

func (adminUserHandlerStub) List(ctx context.Context, input useradminservice.ListInput) ([]domain.User, error) {
	return []domain.User{{UserID: "user-1", Role: "user", Status: "active", Version: 1}}, nil
}

func (adminUserHandlerStub) UpdateRole(ctx context.Context, input useradminservice.UpdateRoleInput) (domain.User, error) {
	return domain.User{
		UserID:  input.TargetUserID,
		Role:    input.Role,
		Status:  "active",
		Version: input.ExpectedVersion + 1,
	}, nil
}

func (adminUserHandlerStub) UpdateStatus(ctx context.Context, input useradminservice.UpdateStatusInput) (domain.User, error) {
	return domain.User{
		UserID:  input.TargetUserID,
		Role:    "user",
		Status:  input.Status,
		Version: input.ExpectedVersion + 1,
	}, nil
}

func (adminUserHandlerStub) RotateAccountKey(ctx context.Context, input useradminservice.AccountKeyInput) (useradminservice.AccountKeyResult, error) {
	return useradminservice.AccountKeyResult{UserID: input.TargetUserID, PublicKey: "pub-key", KeyAlgorithm: "Ed25519", VaultKeyPath: "account-keys/user-1", Version: input.ExpectedVersion + 1}, nil
}

func (adminUserHandlerStub) RecoverAccountKey(ctx context.Context, input useradminservice.AccountKeyInput) (useradminservice.AccountKeyResult, error) {
	return useradminservice.AccountKeyResult{UserID: input.TargetUserID, PublicKey: "pub-key", KeyAlgorithm: "Ed25519", VaultKeyPath: "account-keys/user-1", Version: input.ExpectedVersion + 1}, nil
}

func (adminUserHandlerStub) BackfillAccountKey(ctx context.Context, input useradminservice.AccountKeyInput) (useradminservice.AccountKeyResult, error) {
	return useradminservice.AccountKeyResult{UserID: input.TargetUserID, PublicKey: "pub-key", KeyAlgorithm: "Ed25519", VaultKeyPath: "account-keys/user-1", Version: input.ExpectedVersion + 1}, nil
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

func (shopHandlerStub) ListPledges(ctx context.Context, input shopservice.PledgeHistoryInput) ([]domain.Pledge, error) {
	return []domain.Pledge{{
		PledgeID:  "pledge-1",
		ShopID:    input.ShopID,
		ProductID: input.ProductID,
		Category:  input.Category,
		Score:     8.5,
	}}, nil
}

func (shopHandlerStub) GetPledgeIntegrity(ctx context.Context, input shopservice.PledgeIntegrityInput) (shopservice.PledgeIntegrityView, error) {
	return shopservice.PledgeIntegrityView{
		PledgeID:          input.PledgeID,
		ShopID:            input.ShopID,
		DataHash:          "data-hash",
		ChainAnchorStatus: "pending_anchor",
		IntegrityStatus:   "pending_anchor",
	}, nil
}

func (shopHandlerStub) GetPledgeProof(ctx context.Context, input shopservice.PledgeIntegrityInput) (shopservice.PledgeProofBundle, error) {
	return shopservice.PledgeProofBundle{
		PledgeID:      input.PledgeID,
		ShopID:        input.ShopID,
		ProofStatus:   "verified",
		ProofHeadline: "Cam ket da duoc xac thuc",
		ProofSummary:  "Hash trung khop voi ban ghi da neo",
		Integrity: shopservice.PledgeIntegrityView{
			PledgeID:          input.PledgeID,
			ShopID:            input.ShopID,
			DataHash:          "data-hash",
			ChainAnchorStatus: "anchored",
			IntegrityStatus:   "anchored",
			OnChainMatch:      true,
		},
	}, nil
}

func (shopHandlerStub) ReanchorPledgeIntegrity(ctx context.Context, input shopservice.ModeratePledgeIntegrityInput) (domain.Pledge, error) {
	return domain.Pledge{PledgeID: input.PledgeID, ShopID: input.ShopID, IntegrityStatus: "reanchored", Version: input.ExpectedVersion + 1}, nil
}

func (shopHandlerStub) RevokePledgeIntegrity(ctx context.Context, input shopservice.ModeratePledgeIntegrityInput) (domain.Pledge, error) {
	return domain.Pledge{PledgeID: input.PledgeID, ShopID: input.ShopID, IntegrityStatus: "revoked", Version: input.ExpectedVersion + 1}, nil
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

func (shopHandlerStub) DeleteReview(ctx context.Context, input shopservice.DeleteReviewInput) (domain.ShopReview, error) {
	return domain.ShopReview{
		ReviewID:       "review-1",
		ShopID:         input.ShopID,
		ReviewerUserID: input.ReviewerUserID,
		Status:         shopservice.ShopStatusDeleted,
		Version:        input.ExpectedVersion + 1,
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

type productHandlerStub struct{}

func (productHandlerStub) Create(ctx context.Context, input productservice.CreateInput) (domain.Product, error) {
	return domain.Product{
		ProductID:   "product-1",
		ShopID:      input.ShopID,
		OwnerUserID: input.OwnerUserID,
		Name:        input.Name,
		Description: input.Description,
		Price:       input.Price,
		Currency:    input.Currency,
		Status:      productservice.ProductStatusActive,
		Version:     1,
	}, nil
}

func (productHandlerStub) Update(ctx context.Context, input productservice.UpdateInput) (domain.Product, error) {
	return domain.Product{
		ProductID:   input.ProductID,
		ShopID:      input.ShopID,
		OwnerUserID: input.OwnerUserID,
		Name:        input.Name,
		Description: input.Description,
		Price:       input.Price,
		Currency:    input.Currency,
		Status:      productservice.ProductStatusActive,
		Version:     input.ExpectedVersion + 1,
	}, nil
}

func (productHandlerStub) Delete(ctx context.Context, input productservice.DeleteInput) (domain.Product, error) {
	return domain.Product{
		ProductID:   input.ProductID,
		ShopID:      input.ShopID,
		OwnerUserID: input.OwnerUserID,
		Status:      productservice.ProductStatusDeleted,
		Version:     input.ExpectedVersion + 1,
	}, nil
}

func (productHandlerStub) Moderate(ctx context.Context, input productservice.ModerateInput) (domain.Product, error) {
	return domain.Product{
		ProductID:         input.ProductID,
		Status:            input.Status,
		Version:           input.ExpectedVersion + 1,
		ModeratedByUserID: input.ModeratorUserID,
		ModerationNote:    input.ModerationNote,
	}, nil
}

func (productHandlerStub) BulkUpsert(ctx context.Context, input productservice.BulkUpsertInput) ([]domain.Product, error) {
	return []domain.Product{{
		ProductID:   "product-1",
		ShopID:      input.ShopID,
		OwnerUserID: input.OwnerUserID,
		Name:        input.Items[0].Name,
		Status:      productservice.ProductStatusActive,
		Version:     1,
	}}, nil
}

func (productHandlerStub) GetByID(ctx context.Context, shopID, productID string) (domain.Product, error) {
	return domain.Product{
		ProductID:   productID,
		ShopID:      shopID,
		OwnerUserID: "user-1",
		Name:        "Apple",
		Price:       10,
		Currency:    "VND",
		Status:      productservice.ProductStatusActive,
		Version:     1,
	}, nil
}

func (productHandlerStub) List(ctx context.Context, input productservice.ListInput) ([]domain.Product, error) {
	return []domain.Product{{
		ProductID:   "product-1",
		ShopID:      input.ShopID,
		OwnerUserID: "user-1",
		Name:        "Apple",
		Price:       10,
		Currency:    "VND",
		Status:      productservice.ProductStatusActive,
		Version:     1,
	}}, nil
}

func (productHandlerStub) CreateFreshnessReport(ctx context.Context, input productservice.FreshnessReportInput) (domain.ProductFreshnessReport, error) {
	return domain.ProductFreshnessReport{
		ReportID:       "report-1",
		ProductID:      input.ProductID,
		ShopID:         input.ShopID,
		ReporterUserID: input.ReporterUserID,
		Status:         productservice.FreshnessReportStatusActive,
		Version:        1,
		Score:          input.Score,
		Category:       input.Category,
		Confidence:     input.Confidence,
		Comment:        input.Comment,
		ImageHash:      input.ImageHash,
	}, nil
}

func (productHandlerStub) ModerateFreshnessReport(ctx context.Context, input productservice.ModerateFreshnessReportInput) (domain.ProductFreshnessReport, error) {
	return domain.ProductFreshnessReport{
		ReportID:          input.ReportID,
		Status:            input.Status,
		ModeratedByUserID: input.ModeratorUserID,
		ModerationNote:    input.ModerationNote,
		Version:           input.ExpectedVersion + 1,
	}, nil
}

func (productHandlerStub) ListFreshnessReports(ctx context.Context, shopID, productID string) ([]domain.ProductFreshnessReport, error) {
	return []domain.ProductFreshnessReport{{
		ReportID:  "report-1",
		ProductID: productID,
		ShopID:    shopID,
		Status:    productservice.FreshnessReportStatusActive,
		Version:   1,
	}}, nil
}

func (productHandlerStub) ListFreshnessReportsAdmin(ctx context.Context, input productservice.ListFreshnessReportAdminInput) (productservice.ListFreshnessReportAdminResult, error) {
	return productservice.ListFreshnessReportAdminResult{
		Items: []domain.ProductFreshnessReport{{
			ReportID:       "report-1",
			ProductID:      "product-1",
			ShopID:         "shop-1",
			ReporterUserID: "buyer-1",
			Status:         productservice.FreshnessReportStatusActive,
			Version:        1,
		}},
		Page:     input.Page,
		PageSize: input.PageSize,
		Total:    1,
	}, nil
}
