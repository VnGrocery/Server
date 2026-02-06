package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/dto"
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
	pledgeID := c.PostForm("pledgeId")
	if pledgeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "pledgeId is required in multipart field 'pledgeId'",
		})
		return
	}

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

	result, err := h.checker.Check(c.Request.Context(), buyerservice.CheckInput{
		PledgeID: pledgeID,
		Image: visionservice.ImageInput{
			Filename: fileHeader.Filename,
			Size:     fileHeader.Size,
			Content:  file,
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
		PledgeID:         result.PledgeID,
		Trusted:          result.Trusted,
		Verdict:          result.Verdict,
		PledgedScore:     result.PledgedScore,
		ActualScore:      result.ActualScore,
		ScoreDelta:       result.ScoreDelta,
		PledgedCategory:  result.PledgedCategory,
		ActualCategory:   result.ActualCategory,
		ActualConfidence: result.ActualConfidence,
	})
}
