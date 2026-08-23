package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/dto"
	"vngrocery/internal/api/middleware"
	"vngrocery/internal/domain"
	shopsvc "vngrocery/internal/service/shop"
)

type ShopService interface {
	Create(ctx context.Context, input shopsvc.CreateInput) (domain.Shop, error)
	Update(ctx context.Context, input shopsvc.UpdateInput) (domain.Shop, error)
	Delete(ctx context.Context, input shopsvc.DeleteInput) (domain.Shop, error)
	Moderate(ctx context.Context, input shopsvc.ModerateInput) (domain.Shop, error)
	GetByID(ctx context.Context, shopID string) (shopsvc.ShopView, error)
	List(ctx context.Context, input shopsvc.ListInput) (shopsvc.ListResult, error)
	ListPledges(ctx context.Context, input shopsvc.PledgeHistoryInput) ([]domain.Pledge, error)
	GetPledgeIntegrity(ctx context.Context, input shopsvc.PledgeIntegrityInput) (shopsvc.PledgeIntegrityView, error)
	GetPledgeProof(ctx context.Context, input shopsvc.PledgeIntegrityInput) (shopsvc.PledgeProofBundle, error)
	ReanchorPledgeIntegrity(ctx context.Context, input shopsvc.ModeratePledgeIntegrityInput) (domain.Pledge, error)
	RevokePledgeIntegrity(ctx context.Context, input shopsvc.ModeratePledgeIntegrityInput) (domain.Pledge, error)
	Review(ctx context.Context, input shopsvc.ReviewInput) (domain.ShopReview, error)
	DeleteReview(ctx context.Context, input shopsvc.DeleteReviewInput) (domain.ShopReview, error)
	ListReviews(ctx context.Context, shopID string) ([]shopsvc.ReviewView, error)
}

type ShopHandler struct {
	shops ShopService
}

func NewShopHandler(shops ShopService) *ShopHandler {
	return &ShopHandler{shops: shops}
}

func (h *ShopHandler) Create(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}

	var request dto.UpsertShopRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	shop, err := h.shops.Create(c.Request.Context(), shopsvc.CreateInput{
		OwnerUserID: principal.UserID,
		OwnerRole:   principal.Role,
		Name:        request.Name,
		Description: request.Description,
		Address:     request.Address,
		Latitude:    request.Latitude,
		Longitude:   request.Longitude,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toShopResponse(shopsvc.ShopView{Shop: shop}))
}

func (h *ShopHandler) GetMine(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}
	reader, ok := h.shops.(interface {
		GetMine(context.Context, string) (shopsvc.ShopView, error)
	})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "owned shop reader is not configured"})
		return
	}
	shop, err := reader.GetMine(c.Request.Context(), principal.UserID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toShopResponse(shop))
}

func (h *ShopHandler) Update(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}

	var request dto.UpsertShopRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	shop, err := h.shops.Update(c.Request.Context(), shopsvc.UpdateInput{
		ShopID:          c.Param("shopId"),
		OwnerUserID:     principal.UserID,
		ChangeReason:    request.ChangeReason,
		ExpectedVersion: request.ExpectedVersion,
		Name:            request.Name,
		Description:     request.Description,
		Address:         request.Address,
		Latitude:        request.Latitude,
		Longitude:       request.Longitude,

		CommentModeration: request.CommentModeration,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toShopResponse(shopsvc.ShopView{Shop: shop}))
}

func (h *ShopHandler) Delete(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}

	expectedVersion, parseErr := parsePositiveIntQuery(c.Query("expectedVersion"), "expectedVersion")
	if parseErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": parseErr.Error()})
		return
	}

	shop, err := h.shops.Delete(c.Request.Context(), shopsvc.DeleteInput{
		ShopID:          c.Param("shopId"),
		OwnerUserID:     principal.UserID,
		ChangeReason:    c.Query("changeReason"),
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toShopResponse(shopsvc.ShopView{Shop: shop}))
}

func (h *ShopHandler) Moderate(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}

	var request dto.ModerateShopRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	shop, err := h.shops.Moderate(c.Request.Context(), shopsvc.ModerateInput{
		ShopID:          c.Param("shopId"),
		ModeratorUserID: principal.UserID,
		ExpectedVersion: request.ExpectedVersion,
		Status:          request.Status,
		ModerationNote:  request.ModerationNote,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toShopResponse(shopsvc.ShopView{Shop: shop}))
}

func (h *ShopHandler) List(c *gin.Context) {
	page, pageSize, err := parsePagination(c.Query("page"), c.Query("pageSize"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	near, err := parseNearQuery(c.Query("lat"), c.Query("lng"), c.Query("radiusKm"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.shops.List(c.Request.Context(), shopsvc.ListInput{
		Page:               page,
		PageSize:           pageSize,
		Query:              c.Query("q"),
		Status:             c.Query("status"),
		OwnerUserID:        c.Query("ownerUserId"),
		IncludeAllStatuses: false,
		Near:               near,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}

	items := make([]dto.ShopResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toShopResponse(item))
	}
	c.JSON(http.StatusOK, dto.ShopListResponse{
		Items:    items,
		Page:     result.Page,
		PageSize: result.PageSize,
		Total:    result.Total,
		HasNext:  result.HasNext,
	})
}

func (h *ShopHandler) AdminList(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}

	page, pageSize, err := parsePagination(c.Query("page"), c.Query("pageSize"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.shops.List(c.Request.Context(), shopsvc.ListInput{
		Page:               page,
		PageSize:           pageSize,
		Query:              c.Query("q"),
		Status:             c.Query("status"),
		OwnerUserID:        c.Query("ownerUserId"),
		ActorUserID:        principal.UserID,
		IncludeAllStatuses: true,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}

	items := make([]dto.ShopResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toShopResponse(item))
	}
	c.JSON(http.StatusOK, dto.ShopListResponse{
		Items:    items,
		Page:     result.Page,
		PageSize: result.PageSize,
		Total:    result.Total,
		HasNext:  result.HasNext,
	})
}

func (h *ShopHandler) GetByID(c *gin.Context) {
	shop, err := h.shops.GetByID(c.Request.Context(), c.Param("shopId"))
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toShopResponse(shop))
}

func (h *ShopHandler) ListPledges(c *gin.Context) {
	pledges, err := h.shops.ListPledges(c.Request.Context(), shopsvc.PledgeHistoryInput{
		ShopID:    c.Param("shopId"),
		ProductID: c.Query("productId"),
		Category:  c.Query("category"),
	})
	if err != nil {
		h.writeError(c, err)
		return
	}

	items := make([]dto.PledgeResponse, 0, len(pledges))
	for _, pledge := range pledges {
		items = append(items, toPledgeResponse(pledge))
	}

	c.JSON(http.StatusOK, dto.PledgeHistoryResponse{Items: items})
}

func (h *ShopHandler) GetPledgeIntegrity(c *gin.Context) {
	integrityView, err := h.shops.GetPledgeIntegrity(c.Request.Context(), shopsvc.PledgeIntegrityInput{
		ShopID:   c.Param("shopId"),
		PledgeID: c.Param("pledgeId"),
		DataHash: c.Query("dataHash"),
	})
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.PledgeIntegrityResponse{
		PledgeID:          integrityView.PledgeID,
		ShopID:            integrityView.ShopID,
		DataHash:          integrityView.DataHash,
		ProvidedDataHash:  integrityView.ProvidedDataHash,
		ChainTxHash:       integrityView.ChainTxHash,
		ChainBlockNumber:  integrityView.ChainBlockNumber,
		ChainAnchorStatus: integrityView.ChainAnchorStatus,
		ChainAnchorTime:   integrityView.ChainAnchorTime,
		IntegrityStatus:   integrityView.IntegrityStatus,
		OnChainMatch:      integrityView.OnChainMatch,
		ProvidedHashMatch: integrityView.ProvidedHashMatch,
		OnChainDataHash:   integrityView.OnChainDataHash,
		OnChainVersion:    integrityView.OnChainVersion,
		OnChainTimestamp:  integrityView.OnChainTimestamp,
		OnChainPresent:    integrityView.OnChainPresent,
		MismatchReason:    integrityView.MismatchReason,
		LastCheckedAt:     integrityView.LastCheckedAt,
		CanReanchor:       integrityView.CanReanchor,
		CanRevoke:         integrityView.CanRevoke,
	})
}

func (h *ShopHandler) GetPledgeProof(c *gin.Context) {
	proof, err := h.shops.GetPledgeProof(c.Request.Context(), shopsvc.PledgeIntegrityInput{
		ShopID:   c.Param("shopId"),
		PledgeID: c.Param("pledgeId"),
		DataHash: c.Query("dataHash"),
	})
	if err != nil {
		h.writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.PledgeProofBundleResponse{
		PledgeID:           proof.PledgeID,
		ShopID:             proof.ShopID,
		ProductID:          proof.ProductID,
		BundleID:           proof.BundleID,
		Score:              proof.Score,
		Category:           proof.Category,
		Confidence:         proof.Confidence,
		CommittedAt:        proof.CommittedAt,
		ImageHash:          proof.ImageHash,
		ImageCID:           proof.ImageCID,
		ProofStatus:        proof.ProofStatus,
		ProofHeadline:      proof.ProofHeadline,
		ProofSummary:       proof.ProofSummary,
		RecommendedActions: proof.RecommendedActions,
		Integrity: dto.PledgeIntegrityResponse{
			PledgeID:          proof.Integrity.PledgeID,
			ShopID:            proof.Integrity.ShopID,
			DataHash:          proof.Integrity.DataHash,
			ProvidedDataHash:  proof.Integrity.ProvidedDataHash,
			ChainTxHash:       proof.Integrity.ChainTxHash,
			ChainBlockNumber:  proof.Integrity.ChainBlockNumber,
			ChainAnchorStatus: proof.Integrity.ChainAnchorStatus,
			ChainAnchorTime:   proof.Integrity.ChainAnchorTime,
			IntegrityStatus:   proof.Integrity.IntegrityStatus,
			OnChainMatch:      proof.Integrity.OnChainMatch,
			ProvidedHashMatch: proof.Integrity.ProvidedHashMatch,
			OnChainDataHash:   proof.Integrity.OnChainDataHash,
			OnChainVersion:    proof.Integrity.OnChainVersion,
			OnChainTimestamp:  proof.Integrity.OnChainTimestamp,
			OnChainPresent:    proof.Integrity.OnChainPresent,
			MismatchReason:    proof.Integrity.MismatchReason,
			LastCheckedAt:     proof.Integrity.LastCheckedAt,
			CanReanchor:       proof.Integrity.CanReanchor,
			CanRevoke:         proof.Integrity.CanRevoke,
		},
	})
}

func (h *ShopHandler) ReanchorPledgeIntegrity(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}
	var request dto.ModeratePledgeIntegrityRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}
	pledge, err := h.shops.ReanchorPledgeIntegrity(c.Request.Context(), shopsvc.ModeratePledgeIntegrityInput{
		ShopID:          c.Param("shopId"),
		PledgeID:        c.Param("pledgeId"),
		ActorUserID:     principal.UserID,
		ExpectedVersion: request.ExpectedVersion,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toPledgeResponse(pledge))
}

func (h *ShopHandler) RevokePledgeIntegrity(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}
	var request dto.ModeratePledgeIntegrityRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}
	pledge, err := h.shops.RevokePledgeIntegrity(c.Request.Context(), shopsvc.ModeratePledgeIntegrityInput{
		ShopID:          c.Param("shopId"),
		PledgeID:        c.Param("pledgeId"),
		ActorUserID:     principal.UserID,
		ExpectedVersion: request.ExpectedVersion,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toPledgeResponse(pledge))
}

func (h *ShopHandler) CreateReview(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}

	var request dto.CreateShopReviewRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	review, err := h.shops.Review(c.Request.Context(), shopsvc.ReviewInput{
		ShopID:          c.Param("shopId"),
		ReviewerUserID:  principal.UserID,
		ExpectedVersion: request.ExpectedVersion,
		Rating:          request.Rating,
		Comment:         request.Comment,
		ImageURLs:       request.ImageURLs,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}

	// The caller is the reviewer and already knows their own name, so this
	// response leaves it empty; the list endpoint fills it in for everyone else.
	c.JSON(http.StatusCreated, toShopReviewResponse(review, ""))
}

func (h *ShopHandler) DeleteReview(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}
	expectedVersion, parseErr := parsePositiveIntQuery(c.Query("expectedVersion"), "expectedVersion")
	if parseErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": parseErr.Error()})
		return
	}
	review, err := h.shops.DeleteReview(c.Request.Context(), shopsvc.DeleteReviewInput{
		ShopID:          c.Param("shopId"),
		ReviewerUserID:  principal.UserID,
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toShopReviewResponse(review, ""))
}

func (h *ShopHandler) ListReviews(c *gin.Context) {
	reviews, err := h.shops.ListReviews(c.Request.Context(), c.Param("shopId"))
	if err != nil {
		h.writeError(c, err)
		return
	}

	response := make([]dto.ShopReviewResponse, 0, len(reviews))
	for _, view := range reviews {
		response = append(response, toShopReviewResponse(view.Review, view.ReviewerName))
	}

	c.JSON(http.StatusOK, response)
}

// One place that shapes a review response; the three endpoints used to repeat
// the same ten-field literal.
func toShopReviewResponse(review domain.ShopReview, reviewerName string) dto.ShopReviewResponse {
	return dto.ShopReviewResponse{
		ReviewID:       review.ReviewID,
		ShopID:         review.ShopID,
		ReviewerUserID: review.ReviewerUserID,
		ReviewerName:   reviewerName,
		Rating:         review.Rating,
		Comment:        review.Comment,
		ImageURLs:      review.ImageURLs,
		Status:         review.Status,
		Version:        review.Version,
		CreatedAt:      review.CreatedAt,
		UpdatedAt:      review.UpdatedAt,
	}
}

func (h *ShopHandler) writeError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, shopsvc.ErrInvalidShop):
		status = http.StatusBadRequest
	case errors.Is(err, shopsvc.ErrVersionConflict):
		status = http.StatusConflict
	case errors.Is(err, shopsvc.ErrForbidden),
		errors.Is(err, shopsvc.ErrAdminRequired),
		errors.Is(err, shopsvc.ErrSellerRoleRequired):
		status = http.StatusForbidden
	case errors.Is(err, shopsvc.ErrShopAlreadyExists):
		status = http.StatusConflict
	case errors.Is(err, shopsvc.ErrNotFound):
		status = http.StatusNotFound
	}

	c.JSON(status, gin.H{"error": err.Error()})
}

func toShopResponse(view shopsvc.ShopView) dto.ShopResponse {
	shop := view.Shop
	return dto.ShopResponse{
		DistanceKm:        view.DistanceKm,
		ShopID:            shop.ShopID,
		OwnerUserID:       shop.OwnerUserID,
		Version:           shop.Version,
		Name:              shop.Name,
		Description:       shop.Description,
		Address:           shop.Address,
		CommentModeration: shop.CommentModeration,
		Latitude:          shop.Latitude,
		Longitude:         shop.Longitude,
		Status:            shop.Status,
		DataHash:          shop.DataHash,
		ChainTxHash:       shop.ChainTxHash,
		ChainBlockNumber:  shop.ChainBlockNumber,
		ChainAnchorStatus: shop.ChainAnchorStatus,
		ChainAnchorTime:   shop.ChainAnchorTime,
		IntegrityStatus:   shop.IntegrityStatus,
		ModeratedByUserID: shop.ModeratedByUserID,
		ModerationNote:    shop.ModerationNote,
		ModeratedAt:       shop.ModeratedAt,
		TrustSummary: dto.ShopTrustSummaryResponse{
			HasPledges:         view.TrustSummary.HasPledges,
			PledgeCount:        view.TrustSummary.PledgeCount,
			LatestPledgeID:     view.TrustSummary.LatestPledgeID,
			LatestPledgeStatus: view.TrustSummary.LatestPledgeStatus,
			LatestScore:        view.TrustSummary.LatestScore,
			LatestCategory:     view.TrustSummary.LatestCategory,
			LatestConfidence:   view.TrustSummary.LatestConfidence,
			LastCommittedAt:    view.TrustSummary.LastCommittedAt,
			Score:              view.TrustSummary.Score,
			Grade:              view.TrustSummary.Grade,
			FormulaVersion:     view.TrustSummary.FormulaVersion,
			PledgeScore:        view.TrustSummary.PledgeScore,
			ReviewScore:        view.TrustSummary.ReviewScore,
			BuyerCheckScore:    view.TrustSummary.BuyerCheckScore,
			ConsistencyScore:   view.TrustSummary.ConsistencyScore,
			RecencyScore:       view.TrustSummary.RecencyScore,
			CoverageScore:      view.TrustSummary.CoverageScore,
			BuyerCheckCount:    view.TrustSummary.BuyerCheckCount,
			TrustedCheckCount:  view.TrustSummary.TrustedCheckCount,
			HighRiskCheckCount: view.TrustSummary.HighRiskCheckCount,

			CommentScore:         view.TrustSummary.CommentScore,
			CommentModeration:    view.TrustSummary.CommentModeration,
			CommentCount:         view.TrustSummary.CommentCount,
			CommentPendingCount:  view.TrustSummary.CommentPendingCount,
			CommentRejectedCount: view.TrustSummary.CommentRejectedCount,
			Reasons:              view.TrustSummary.Reasons,
		},
		RatingSummary: dto.ShopRatingSummaryResponse{
			RatingCount:   view.RatingSummary.RatingCount,
			AverageRating: view.RatingSummary.AverageRating,
		},
		CreatedAt: shop.CreatedAt,
		UpdatedAt: shop.UpdatedAt,
	}
}

func toPledgeResponse(pledge domain.Pledge) dto.PledgeResponse {
	committedAt := pledge.CommittedAt
	if committedAt.IsZero() {
		committedAt = pledge.CreatedAt
	}
	return dto.PledgeResponse{
		PledgeID:          pledge.PledgeID,
		ShopID:            pledge.ShopID,
		ProductID:         pledge.ProductID,
		BundleID:          pledge.BundleID,
		CreatedByUserID:   pledge.CreatedByUserID,
		Status:            pledge.Status,
		Version:           pledge.Version,
		Score:             pledge.Score,
		Category:          pledge.Category,
		Confidence:        pledge.Confidence,
		ImageHash:         pledge.ImageHash,
		ImageCID:          pledge.ImageCID,
		DataHash:          pledge.DataHash,
		ChainTxHash:       pledge.ChainTxHash,
		ChainBlockNumber:  pledge.ChainBlockNumber,
		ChainAnchorStatus: pledge.ChainAnchorStatus,
		ChainAnchorTime:   pledge.ChainAnchorTime,
		IntegrityStatus:   pledge.IntegrityStatus,
		CommittedAt:       committedAt,
		CreatedAt:         pledge.CreatedAt,
		UpdatedAt:         pledge.UpdatedAt,
	}
}

// parseNearQuery reads the optional lat/lng/radiusKm trio.
//
// All three are omitted by a client with no location, which is not an error —
// the listing simply is not narrowed. Supplying some but not all is an error:
// silently ignoring half a request would look like the filter had run.
func parseNearQuery(latValue, lngValue, radiusValue string) (shopsvc.NearFilter, error) {
	lat := strings.TrimSpace(latValue)
	lng := strings.TrimSpace(lngValue)
	radius := strings.TrimSpace(radiusValue)

	if lat == "" && lng == "" && radius == "" {
		return shopsvc.NearFilter{}, nil
	}
	if lat == "" || lng == "" || radius == "" {
		return shopsvc.NearFilter{}, errors.New("lat, lng and radiusKm must be supplied together")
	}

	near := shopsvc.NearFilter{}
	var err error
	if near.Latitude, err = strconv.ParseFloat(lat, 64); err != nil {
		return shopsvc.NearFilter{}, errors.New("lat must be a number")
	}
	if near.Longitude, err = strconv.ParseFloat(lng, 64); err != nil {
		return shopsvc.NearFilter{}, errors.New("lng must be a number")
	}
	if near.RadiusKm, err = strconv.ParseFloat(radius, 64); err != nil {
		return shopsvc.NearFilter{}, errors.New("radiusKm must be a number")
	}
	if !near.Valid() {
		return shopsvc.NearFilter{}, fmt.Errorf(
			"lat must be -90..90, lng -180..180 and radiusKm 0..%g",
			shopsvc.MaxNearRadiusKm,
		)
	}
	return near, nil
}

func parsePagination(pageValue, pageSizeValue string) (int, int, error) {
	page := 1
	pageSize := 20
	var err error
	if pageValue != "" {
		page, err = strconv.Atoi(pageValue)
		if err != nil || page < 1 {
			return 0, 0, errors.New("page must be a positive integer")
		}
	}
	if pageSizeValue != "" {
		pageSize, err = strconv.Atoi(pageSizeValue)
		if err != nil || pageSize < 1 {
			return 0, 0, errors.New("pageSize must be a positive integer")
		}
	}
	return page, pageSize, nil
}
