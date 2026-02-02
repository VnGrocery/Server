package vision

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
)

var (
	ErrInvalidImage        = errors.New("invalid image file")
	ErrProviderUnavailable = errors.New("vision provider is unavailable")
)

type ImageInput struct {
	Filename string
	Size     int64
	Content  io.Reader
}

type ScoreResult struct {
	Score      float64 `json:"score"`
	Category   string  `json:"category"`
	Confidence float64 `json:"confidence"`
}

type Provider interface {
	ScoreImage(ctx context.Context, image ImagePayload) (ScoreResult, error)
}

type ImagePayload struct {
	Filename    string
	ContentType string
	Data        []byte
}

type ImageScorer interface {
	Score(ctx context.Context, input ImageInput) (ScoreResult, error)
}

type Service struct {
	provider Provider
}

func NewService(provider Provider) *Service {
	return &Service{provider: provider}
}

func (s *Service) Score(ctx context.Context, input ImageInput) (ScoreResult, error) {
	if input.Content == nil {
		return ScoreResult{}, ErrInvalidImage
	}
	if s.provider == nil {
		return ScoreResult{}, ErrProviderUnavailable
	}

	buffer, err := io.ReadAll(io.LimitReader(input.Content, maxImageBytes+1))
	if err != nil {
		return ScoreResult{}, fmt.Errorf("failed to read uploaded image: %w", err)
	}
	if len(buffer) == 0 {
		return ScoreResult{}, ErrInvalidImage
	}
	if len(buffer) > maxImageBytes {
		return ScoreResult{}, fmt.Errorf("%w: image exceeds 10 MB limit", ErrInvalidImage)
	}

	contentType := http.DetectContentType(buffer)
	if !isSupportedImageType(contentType) {
		return ScoreResult{}, fmt.Errorf("%w: unsupported image content type %s", ErrInvalidImage, contentType)
	}

	filename := sanitizeFilename(input.Filename, contentType)
	result, err := s.provider.ScoreImage(ctx, ImagePayload{
		Filename:    filename,
		ContentType: contentType,
		Data:        bytes.Clone(buffer),
	})
	if err != nil {
		return ScoreResult{}, err
	}

	if err := validateScoreResult(result); err != nil {
		return ScoreResult{}, err
	}

	return result, nil
}

const maxImageBytes = 10 << 20

func validateScoreResult(result ScoreResult) error {
	if result.Score < 0 || result.Score > 10 {
		return fmt.Errorf("provider returned score outside range 0..10")
	}
	if result.Confidence < 0 || result.Confidence > 1 {
		return fmt.Errorf("provider returned confidence outside range 0..1")
	}
	if strings.TrimSpace(result.Category) == "" {
		return fmt.Errorf("provider returned an empty category")
	}

	return nil
}

func isSupportedImageType(contentType string) bool {
	switch contentType {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
		return true
	default:
		return false
	}
}

func sanitizeFilename(filename, contentType string) string {
	name := strings.TrimSpace(filepath.Base(filename))
	if name == "" || name == "." || name == string(filepath.Separator) {
		return "upload" + extensionForContentType(contentType)
	}

	return name
}

func extensionForContentType(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ""
	}
}
