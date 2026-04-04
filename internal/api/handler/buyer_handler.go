package handler

import (
	"bytes"
	"context"
	"errors"
	"io"
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
	checker buyerservice.CheckService
}

func NewBuyerHandler(checker buyerservice.CheckService) *BuyerHandler {
	return &BuyerHandler{checker: checker}
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
	if fileHeader.Size <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "image file must not be empty",
		})
		return
	}
	if fileHeader.Size > maxBuyerImageBytes {
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

	imageBytes, err := io.ReadAll(io.LimitReader(file, maxBuyerImageBytes+1))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to read uploaded image",
		})
		return
	}
	if len(imageBytes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "image file must not be empty",
		})
		return
	}
	if len(imageBytes) > maxBuyerImageBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error": "image file exceeds 10 MB limit",
		})
		return
	}

	result, err := h.checker.Check(c.Request.Context(), buyerservice.CheckInput{
		PledgeID:    pledgeID,
		BuyerUserID: principal.UserID,
		ImageHash:   sha256Hex(imageBytes),
		Image: visionservice.ImageInput{
			Filename: fileHeader.Filename,
			Size:     int64(len(imageBytes)),
			Content:  bytes.NewReader(imageBytes),
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
		CheckID:          check.CheckID,
		ShopID:           check.ShopID,
		ProductID:        check.ProductID,
		Status:           check.Status,
		Version:          check.Version,
		PolicyVersion:    check.PolicyVersion,
		HasPledge:        strings.TrimSpace(check.PledgeID) != "",
		PledgeID:         check.PledgeID,
		Trusted:          check.Trusted,
		Verdict:          check.Verdict,
		PledgedScore:     check.PledgedScore,
		ActualScore:      check.ActualScore,
		ScoreDelta:       check.ScoreDelta,
		ScoreDeltaAbs:    check.ScoreDeltaAbs,
		PledgedCategory:  check.PledgedCategory,
		ActualCategory:   check.ActualCategory,
		ActualConfidence: check.ActualConfidence,
		CategoryMatch:    check.CategoryMatch,
		Reasons:          check.Reasons,
	})
}
