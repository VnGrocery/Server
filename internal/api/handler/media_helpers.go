package handler

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
)

type mediaUploadConfig struct {
	maxImageBytes int64
	allowedTypes  map[string]struct{}
}

type imageUploadPayload struct {
	filename    string
	contentType string
	data        []byte
}

func newMediaUploadConfig(maxImageBytes int64, allowedTypes []string) mediaUploadConfig {
	allowed := make(map[string]struct{}, len(allowedTypes))
	for _, value := range allowedTypes {
		trimmed := strings.TrimSpace(strings.ToLower(value))
		if trimmed == "" {
			continue
		}
		allowed[trimmed] = struct{}{}
	}
	if maxImageBytes <= 0 {
		maxImageBytes = 10 << 20
	}
	if len(allowed) == 0 {
		allowed["image/jpeg"] = struct{}{}
		allowed["image/png"] = struct{}{}
		allowed["image/webp"] = struct{}{}
	}
	return mediaUploadConfig{
		maxImageBytes: maxImageBytes,
		allowedTypes:  allowed,
	}
}

func NewMediaUploadConfigForRuntime(maxImageBytes int64, allowedTypes []string) mediaUploadConfig {
	return newMediaUploadConfig(maxImageBytes, allowedTypes)
}

func readMultipartImage(fileHeader *multipart.FileHeader, cfg mediaUploadConfig) (imageUploadPayload, error) {
	if fileHeader == nil {
		return imageUploadPayload{}, fmt.Errorf("image file is required in multipart field 'image'")
	}
	if fileHeader.Size <= 0 {
		return imageUploadPayload{}, fmt.Errorf("image file must not be empty")
	}
	if fileHeader.Size > cfg.maxImageBytes {
		return imageUploadPayload{}, fmt.Errorf("image file exceeds %d bytes limit", cfg.maxImageBytes)
	}

	file, err := fileHeader.Open()
	if err != nil {
		return imageUploadPayload{}, fmt.Errorf("failed to open uploaded image")
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, cfg.maxImageBytes+1))
	if err != nil {
		return imageUploadPayload{}, fmt.Errorf("failed to read uploaded image")
	}
	if len(data) == 0 {
		return imageUploadPayload{}, fmt.Errorf("image file must not be empty")
	}
	if int64(len(data)) > cfg.maxImageBytes {
		return imageUploadPayload{}, fmt.Errorf("image file exceeds %d bytes limit", cfg.maxImageBytes)
	}

	contentType := detectContentType(data)
	if _, ok := cfg.allowedTypes[strings.ToLower(contentType)]; !ok {
		if fallback := contentTypeFromFilename(fileHeader.Filename); fallback != "" {
			contentType = fallback
		}
	}
	if _, ok := cfg.allowedTypes[strings.ToLower(contentType)]; !ok {
		return imageUploadPayload{}, fmt.Errorf("unsupported image type: %s", contentType)
	}

	return imageUploadPayload{
		filename:    fileHeader.Filename,
		contentType: contentType,
		data:        data,
	}, nil
}

func detectContentType(data []byte) string {
	sniffLen := len(data)
	if sniffLen > 512 {
		sniffLen = 512
	}
	return strings.ToLower(http.DetectContentType(data[:sniffLen]))
}

func contentTypeFromFilename(filename string) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return ""
	}
}

func newImageReader(data []byte) *bytes.Reader {
	return bytes.NewReader(data)
}
