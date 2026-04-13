package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/dto"
	"vngrocery/internal/api/middleware"
	"vngrocery/internal/domain"
	buyerservice "vngrocery/internal/service/buyer"
	visionservice "vngrocery/internal/service/vision"
)

const maxBuyerImageBytes = 10 << 20

type BuyerHandler struct {
	checker   buyerservice.CheckService
	uploader  ImageUploader
	uploadCfg mediaUploadConfig
}

type BuyerAdminService interface {
	Check(ctx context.Context, input buyerservice.CheckInput) (buyerservice.CheckResult, error)
	Moderate(ctx context.Context, input buyerservice.ModerateInput) (domain.BuyerCheck, error)
	List(ctx context.Context, input buyerservice.ListInput) (buyerservice.ListResult, error)
}

func NewBuyerHandler(checker buyerservice.CheckService) *BuyerHandler {
	return &BuyerHandler{
		checker:   checker,
		uploadCfg: newMediaUploadConfig(maxBuyerImageBytes, nil),
	}
}

func (h *BuyerHandler) SetUploader(uploader ImageUploader) {
	h.uploader = uploader
}

func (h *BuyerHandler) SetUploadConfig(cfg mediaUploadConfig) {
	h.uploadCfg = cfg
}

func (h *BuyerHandler) Check(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "authenticated principal was not found in request context",
		})
		return
	}

	pledgeID := c.PostForm("pledgeId")

	fileHeader, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "image file is required in multipart field 'image'",
		})
		return
	}
	upload, err := readMultipartImage(fileHeader, h.uploadCfg)
	if err != nil {
		status := http.StatusBadRequest
		if fileHeader.Size > h.uploadCfg.maxImageBytes {
			status = http.StatusRequestEntityTooLarge
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	imageCID := ""
	if h.uploader != nil {
		uploaded, err := h.uploader.AddBytes(c.Request.Context(), upload.filename, upload.data)
		if err == nil {
			imageCID = uploaded.CID
		}
	}

	result, err := h.checker.Check(c.Request.Context(), buyerservice.CheckInput{
		PledgeID:    pledgeID,
		BuyerUserID: principal.UserID,
		ImageHash:   sha256Hex(upload.data),
		ImageCID:    imageCID,
		Image: visionservice.ImageInput{
			Filename: upload.filename,
			Size:     int64(len(upload.data)),
			Content:  newImageReader(upload.data),
		},
	})
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, buyerservice.ErrInvalidCheck):
			status = http.StatusBadRequest
		case errors.Is(err, buyerservice.ErrRateLimited):
			status = http.StatusTooManyRequests
		case errors.Is(err, visionservice.ErrInvalidImage):
			status = http.StatusBadRequest
		case errors.Is(err, visionservice.ErrProviderUnavailable):
			status = http.StatusServiceUnavailable
		}

		c.JSON(status, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.BuyerCheckResponse{
		CheckID:          result.CheckID,
		ShopID:           result.ShopID,
		ProductID:        result.ProductID,
		BuyerUserID:      result.BuyerUserID,
		Status:           buyerservice.BuyerCheckStatusCompleted,
		Version:          1,
		PolicyVersion:    result.PolicyVersion,
		HasPledge:        result.HasPledge,
		PledgeID:         result.PledgeID,
		Trusted:          result.Trusted,
		Verdict:          result.Verdict,
		PledgedScore:     result.PledgedScore,
		ActualScore:      result.ActualScore,
		ScoreDelta:       result.ScoreDelta,
		ScoreDeltaAbs:    result.ScoreDeltaAbs,
		PledgedCategory:  result.PledgedCategory,
		ActualCategory:   result.ActualCategory,
		ActualConfidence: result.ActualConfidence,
		CategoryMatch:    result.CategoryMatch,
		ImageHash:        result.ImageHash,
		ImageCID:         result.ImageCID,
		Reasons:          result.Reasons,
	})
}

func (h *BuyerHandler) Moderate(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}

	var request dto.ModerateBuyerCheckRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	moderator, ok := h.checker.(interface {
		Moderate(ctx context.Context, input buyerservice.ModerateInput) (domain.BuyerCheck, error)
	})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "buyer moderation is not configured"})
		return
	}

	check, err := moderator.Moderate(c.Request.Context(), buyerservice.ModerateInput{
		CheckID:         c.Param("checkId"),
		ModeratorUserID: principal.UserID,
		ExpectedVersion: request.ExpectedVersion,
		Status:          request.Status,
		ModerationNote:  request.ModerationNote,
	})
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, buyerservice.ErrInvalidCheck):
			status = http.StatusBadRequest
		case errors.Is(err, buyerservice.ErrRateLimited):
			status = http.StatusTooManyRequests
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.BuyerCheckResponse{
		CheckID:           check.CheckID,
		ShopID:            check.ShopID,
		ProductID:         check.ProductID,
		BuyerUserID:       check.BuyerUserID,
		Status:            check.Status,
		Version:           check.Version,
		PolicyVersion:     check.PolicyVersion,
		HasPledge:         strings.TrimSpace(check.PledgeID) != "",
		PledgeID:          check.PledgeID,
		Trusted:           check.Trusted,
		Verdict:           check.Verdict,
		PledgedScore:      check.PledgedScore,
		ActualScore:       check.ActualScore,
		ScoreDelta:        check.ScoreDelta,
		ScoreDeltaAbs:     check.ScoreDeltaAbs,
		PledgedCategory:   check.PledgedCategory,
		ActualCategory:    check.ActualCategory,
		ActualConfidence:  check.ActualConfidence,
		CategoryMatch:     check.CategoryMatch,
		ImageHash:         check.ImageHash,
		ImageCID:          check.ImageCID,
		Reasons:           check.Reasons,
		ModeratedByUserID: check.ModeratedByUserID,
		ModerationNote:    check.ModerationNote,
		ModeratedAt:       check.ModeratedAt,
		CreatedAt:         check.CreatedAt,
		UpdatedAt:         check.UpdatedAt,
	})
}

func (h *BuyerHandler) ListAdmin(c *gin.Context) {
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
	createdAfter, err := parseOptionalTime(c.Query("createdAfter"), "createdAfter")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	createdBefore, err := parseOptionalTime(c.Query("createdBefore"), "createdBefore")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lister, ok := h.checker.(interface {
		List(ctx context.Context, input buyerservice.ListInput) (buyerservice.ListResult, error)
	})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "buyer check list is not configured"})
		return
	}
	result, err := lister.List(c.Request.Context(), buyerservice.ListInput{
		ActorUserID:   principal.UserID,
		CheckID:       c.Query("checkId"),
		ShopID:        c.Query("shopId"),
		ProductID:     c.Query("productId"),
		BuyerUserID:   c.Query("buyerUserId"),
		Status:        c.Query("status"),
		Verdict:       c.Query("verdict"),
		CreatedAfter:  createdAfter,
		CreatedBefore: createdBefore,
		Page:          page,
		PageSize:      pageSize,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	items := make([]dto.BuyerCheckResponse, 0, len(result.Items))
	for _, check := range result.Items {
		items = append(items, dto.BuyerCheckResponse{
			CheckID:           check.CheckID,
			ShopID:            check.ShopID,
			ProductID:         check.ProductID,
			BuyerUserID:       check.BuyerUserID,
			Status:            check.Status,
			Version:           check.Version,
			PolicyVersion:     check.PolicyVersion,
			HasPledge:         strings.TrimSpace(check.PledgeID) != "",
			PledgeID:          check.PledgeID,
			Trusted:           check.Trusted,
			Verdict:           check.Verdict,
			PledgedScore:      check.PledgedScore,
			ActualScore:       check.ActualScore,
			ScoreDelta:        check.ScoreDelta,
			ScoreDeltaAbs:     check.ScoreDeltaAbs,
			PledgedCategory:   check.PledgedCategory,
			ActualCategory:    check.ActualCategory,
			ActualConfidence:  check.ActualConfidence,
			CategoryMatch:     check.CategoryMatch,
			ImageHash:         check.ImageHash,
			ImageCID:          check.ImageCID,
			Reasons:           check.Reasons,
			ModeratedByUserID: check.ModeratedByUserID,
			ModerationNote:    check.ModerationNote,
			ModeratedAt:       check.ModeratedAt,
			CreatedAt:         check.CreatedAt,
			UpdatedAt:         check.UpdatedAt,
		})
	}
	totalPages := 0
	if result.Total > 0 {
		totalPages = (result.Total + result.PageSize - 1) / result.PageSize
	}
	c.JSON(http.StatusOK, dto.BuyerCheckListResponse{
		Items: items,
		Pagination: dto.PaginationResponse{
			Page:       result.Page,
			PageSize:   result.PageSize,
			TotalItems: result.Total,
			TotalPages: totalPages,
		},
	})
}
