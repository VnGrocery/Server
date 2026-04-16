package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/dto"
	"vngrocery/internal/api/middleware"
	"vngrocery/internal/domain"
	bundletokenservice "vngrocery/internal/service/bundletoken"
	sellerservice "vngrocery/internal/service/seller"
	visionservice "vngrocery/internal/service/vision"
)

const maxSellerImageBytes = 10 << 20

type SellerHandler struct {
	scorer    visionservice.ImageScorer
	committer sellerservice.CommitService
	tokens    BundleTokenIssuer
	uploader  ImageUploader
	uploadCfg mediaUploadConfig
}

type BundleTokenIssuer interface {
	Issue(input bundletokenservice.IssueInput) (string, time.Time, error)
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
		uploadCfg: newMediaUploadConfig(maxSellerImageBytes, nil),
	}
}

func (h *SellerHandler) SetUploader(uploader ImageUploader) {
	h.uploader = uploader
}

func (h *SellerHandler) SetUploadConfig(cfg mediaUploadConfig) {
	h.uploadCfg = cfg
}

func (h *SellerHandler) SetBundleTokenIssuer(issuer BundleTokenIssuer) {
	h.tokens = issuer
}

func (h *SellerHandler) Score(c *gin.Context) {
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

	imageHash := sha256Hex(upload.data)
	imageCID := ""
	if h.uploader != nil {
		uploaded, err := h.uploader.AddBytes(c.Request.Context(), upload.filename, upload.data)
		if err == nil {
			imageCID = uploaded.CID
		}
	}
	result, err := h.scorer.Score(c.Request.Context(), visionservice.ImageInput{
		Filename: upload.filename,
		Size:     int64(len(upload.data)),
		Content:  newImageReader(upload.data),
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
		BundleID:        request.BundleID,
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

	var bundleToken string
	var bundleTokenExp *time.Time
	if h.tokens != nil {
		token, expiresAt, tokenErr := h.tokens.Issue(bundletokenservice.IssueInput{
			ShopID:      pledge.ShopID,
			ProductID:   pledge.ProductID,
			BundleID:    pledge.BundleID,
			PledgeID:    pledge.PledgeID,
			CreatedByID: pledge.CreatedByUserID,
		})
		if tokenErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": tokenErr.Error(),
			})
			return
		}
		bundleToken = token
		exp := expiresAt.UTC()
		bundleTokenExp = &exp
	}

	c.JSON(http.StatusCreated, dto.SellerCommitResponse{
		PledgeID:          pledge.PledgeID,
		ShopID:            pledge.ShopID,
		ProductID:         pledge.ProductID,
		BundleID:          pledge.BundleID,
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
		BundleToken:       bundleToken,
		BundleTokenExp:    bundleTokenExp,
		CommittedAt:       pledge.CommittedAt,
		CreatedAt:         pledge.CreatedAt,
		UpdatedAt:         pledge.UpdatedAt,
	})
}

func (h *SellerHandler) ReissueBundleToken(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "authenticated principal was not found in request context",
		})
		return
	}
	if h.tokens == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "bundle token issuer is not configured",
		})
		return
	}

	pledges, ok := h.committer.(interface {
		GetPledgeForSeller(ctx context.Context, shopID, pledgeID, sellerUserID string) (domain.Pledge, error)
	})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "pledge reader is not configured",
		})
		return
	}

	pledge, err := pledges.GetPledgeForSeller(
		c.Request.Context(),
		c.Param("shopId"),
		c.Param("pledgeId"),
		principal.UserID,
	)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, sellerservice.ErrInvalidCommit):
			status = http.StatusBadRequest
		case errors.Is(err, sellerservice.ErrShopNotFound), errors.Is(err, sellerservice.ErrPledgeNotFound):
			status = http.StatusNotFound
		case errors.Is(err, sellerservice.ErrShopOwnership):
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{
			"error": err.Error(),
		})
		return
	}

	token, expiresAt, err := h.tokens.Issue(bundletokenservice.IssueInput{
		ShopID:      pledge.ShopID,
		ProductID:   pledge.ProductID,
		BundleID:    pledge.BundleID,
		PledgeID:    pledge.PledgeID,
		CreatedByID: pledge.CreatedByUserID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	exp := expiresAt.UTC()
	c.JSON(http.StatusOK, dto.BundleTokenResponse{
		BundleToken:          token,
		BundleTokenExpiresAt: &exp,
	})
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
