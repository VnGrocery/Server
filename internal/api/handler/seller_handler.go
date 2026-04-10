package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
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
	uploader  ImageUploader
}

type ImageUploader interface {
	AddBytes(ctx context.Context, filename string, data []byte) (ImageUploadResult, error)
}

type ImageUploadResult struct {
	CID        string
	GatewayURL string
}

func NewSellerHandler(scorer visionservice.ImageScorer, committer sellerservice.CommitService) *SellerHandler {
	return &SellerHandler{
		scorer:    scorer,
		committer: committer,
	}
}

func (h *SellerHandler) SetUploader(uploader ImageUploader) {
	h.uploader = uploader
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

	imageBytes, err := io.ReadAll(io.LimitReader(file, maxSellerImageBytes+1))
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
	if len(imageBytes) > maxSellerImageBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error": "image file exceeds 10 MB limit",
		})
		return
	}

	imageHash := sha256Hex(imageBytes)
	imageCID := ""
	if h.uploader != nil {
		uploaded, err := h.uploader.AddBytes(c.Request.Context(), fileHeader.Filename, imageBytes)
		if err == nil {
			imageCID = uploaded.CID
		}
	}
	result, err := h.scorer.Score(c.Request.Context(), visionservice.ImageInput{
		Filename: fileHeader.Filename,
		Size:     int64(len(imageBytes)),
		Content:  bytes.NewReader(imageBytes),
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
		ImageHash:  imageHash,
		ImageCID:   imageCID,
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
		ProductID:       request.ProductID,
		CreatedByUserID: principal.UserID,
		Score:           request.Score,
		Category:        request.Category,
		Confidence:      request.Confidence,
		ImageHash:       request.ImageHash,
		ImageCID:        request.ImageCID,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sellerservice.ErrInvalidCommit) {
			status = http.StatusBadRequest
		} else if errors.Is(err, sellerservice.ErrShopNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, sellerservice.ErrShopOwnership) {
			status = http.StatusForbidden
		}

		c.JSON(status, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, dto.SellerCommitResponse{
		PledgeID:          pledge.PledgeID,
		ShopID:            pledge.ShopID,
		ProductID:         pledge.ProductID,
		CreatedByUserID:   pledge.CreatedByUserID,
		Status:            pledge.Status,
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
		CreatedAt:         pledge.CreatedAt,
		UpdatedAt:         pledge.UpdatedAt,
	})
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
