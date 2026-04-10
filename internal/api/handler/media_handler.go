package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/dto"
)

type MediaHandler struct {
	uploader  ImageUploader
	uploadCfg mediaUploadConfig
}

func NewMediaHandler(uploader ImageUploader, uploadCfg mediaUploadConfig) *MediaHandler {
	return &MediaHandler{
		uploader:  uploader,
		uploadCfg: uploadCfg,
	}
}

func (h *MediaHandler) UploadImage(c *gin.Context) {
	fileHeader, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image file is required in multipart field 'image'"})
		return
	}

	payload, err := readMultipartImage(fileHeader, h.uploadCfg)
	if err != nil {
		status := http.StatusBadRequest
		if payload.data == nil && fileHeader != nil && fileHeader.Size > h.uploadCfg.maxImageBytes {
			status = http.StatusRequestEntityTooLarge
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	response := dto.MediaImageUploadResponse{
		ImageHash:   sha256Hex(payload.data),
		ContentType: payload.contentType,
		SizeBytes:   int64(len(payload.data)),
	}

	if h.uploader != nil {
		uploaded, uploadErr := h.uploader.AddBytes(c.Request.Context(), payload.filename, payload.data)
		if uploadErr != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": uploadErr.Error()})
			return
		}
		response.ImageCID = uploaded.CID
		response.GatewayURL = uploaded.GatewayURL
	}

	c.JSON(http.StatusCreated, response)
}

type imageUploadService interface {
	AddBytes(ctx context.Context, filename string, data []byte) (ImageUploadResult, error)
}
