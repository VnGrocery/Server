package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/dto"
	visionservice "vngrocery/internal/service/vision"
)

const maxSellerImageBytes = 10 << 20

type SellerHandler struct {
	scorer visionservice.ImageScorer
}

func NewSellerHandler(scorer visionservice.ImageScorer) *SellerHandler {
	return &SellerHandler{scorer: scorer}
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
