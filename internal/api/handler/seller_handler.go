package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/dto"
	"vngrocery/internal/api/middleware"
	sellerservice "vngrocery/internal/service/seller"
	visionservice "vngrocery/internal/service/vision"
)

const maxSellerImageBytes = 10 << 20

type SellerHandler struct {
	scorer    visionservice.ImageScorer
	committer sellerservice.CommitService
}

func NewSellerHandler(scorer visionservice.ImageScorer, committer sellerservice.CommitService) *SellerHandler {
	return &SellerHandler{
		scorer:    scorer,
		committer: committer,
	}
}

func (h *SellerHandler) Score(c *gin.Context) {
	fileHeader, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "image file is required in multipart field 'image'",
		})
		return
	}

	if fileHeader.Size <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "image file must not be empty",
		})
		return
	}

	if fileHeader.Size > maxSellerImageBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error": "image file exceeds 10 MB limit",
		})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to open uploaded image",
		})
		return
	}
	defer file.Close()

	result, err := h.scorer.Score(c.Request.Context(), visionservice.ImageInput{
		Filename: fileHeader.Filename,
		Size:     fileHeader.Size,
		Content:  file,
	})
	if err != nil {
		status := http.StatusInternalServerError
		switch {
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

	c.JSON(http.StatusOK, dto.SellerScoreResponse{
		Score:      result.Score,
		Category:   result.Category,
		Confidence: result.Confidence,
	})
}

func (h *SellerHandler) Commit(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "authenticated principal was not found in request context",
		})
		return
	}

	var request dto.SellerCommitRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid JSON payload",
		})
		return
	}

	pledge, err := h.committer.Commit(c.Request.Context(), sellerservice.CommitInput{
		ShopID:          request.ShopID,
		CreatedByUserID: principal.UserID,
		Score:           request.Score,
		Category:        request.Category,
		Confidence:      request.Confidence,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sellerservice.ErrInvalidCommit) {
			status = http.StatusBadRequest
		}

		c.JSON(status, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, dto.SellerCommitResponse{
		PledgeID:        pledge.PledgeID,
		ShopID:          pledge.ShopID,
		CreatedByUserID: pledge.CreatedByUserID,
		Status:          pledge.Status,
		Score:           pledge.Score,
		Category:        pledge.Category,
		Confidence:      pledge.Confidence,
		CreatedAt:       pledge.CreatedAt,
		UpdatedAt:       pledge.UpdatedAt,
	})
}
