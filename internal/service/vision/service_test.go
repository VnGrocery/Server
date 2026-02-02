package vision

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

type providerStub struct {
	scoreImage func(ctx context.Context, image ImagePayload) (ScoreResult, error)
}

func (s providerStub) ScoreImage(ctx context.Context, image ImagePayload) (ScoreResult, error) {
	return s.scoreImage(ctx, image)
}

func TestScoreRejectsUnsupportedContentType(t *testing.T) {
	service := NewService(providerStub{
		scoreImage: func(ctx context.Context, image ImagePayload) (ScoreResult, error) {
			t.Fatal("provider should not be called for unsupported content")
			return ScoreResult{}, nil
		},
	})

	_, err := service.Score(context.Background(), ImageInput{
		Filename: "payload.txt",
		Content:  bytes.NewBufferString("plain text"),
	})
	if !errors.Is(err, ErrInvalidImage) {
		t.Fatalf("expected ErrInvalidImage, got %v", err)
	}
}

func TestScorePassesValidatedImageToProvider(t *testing.T) {
	service := NewService(providerStub{
		scoreImage: func(ctx context.Context, image ImagePayload) (ScoreResult, error) {
			if image.ContentType != "image/png" {
				t.Fatalf("expected image/png, got %s", image.ContentType)
			}
			if image.Filename != "store.png" {
				t.Fatalf("unexpected filename: %s", image.Filename)
			}

			return ScoreResult{
				Score:      9.1,
				Category:   "fresh_produce",
				Confidence: 0.88,
			}, nil
		},
	})

	pngHeader := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	payload := append(pngHeader, bytes.Repeat([]byte{0x00}, 32)...)

	result, err := service.Score(context.Background(), ImageInput{
		Filename: "store.png",
		Content:  bytes.NewReader(payload),
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.Category != "fresh_produce" {
		t.Fatalf("unexpected category: %s", result.Category)
	}
}

func TestScoreRejectsInvalidProviderResponse(t *testing.T) {
	service := NewService(providerStub{
		scoreImage: func(ctx context.Context, image ImagePayload) (ScoreResult, error) {
			return ScoreResult{
				Score:      11,
				Category:   "fresh_produce",
				Confidence: 0.9,
			}, nil
		},
	})

	jpegHeader := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00}
	payload := append(jpegHeader, bytes.Repeat([]byte{0x00}, 32)...)

	_, err := service.Score(context.Background(), ImageInput{
		Filename: "store.jpg",
		Content:  bytes.NewReader(payload),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
