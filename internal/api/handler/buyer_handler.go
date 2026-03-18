package handler

import (
	"bytes"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/dto"
	"vngrocery/internal/api/middleware"
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
