package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	visionservice "vngrocery/internal/service/vision"
)

type scorerStub struct {
	score func(ctx context.Context, input visionservice.ImageInput) (visionservice.ScoreResult, error)
}

func (s scorerStub) Score(ctx context.Context, input visionservice.ImageInput) (visionservice.ScoreResult, error) {
	return s.score(ctx, input)
}

func TestSellerScoreRequiresImage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewSellerHandler(scorerStub{
		score: func(ctx context.Context, input visionservice.ImageInput) (visionservice.ScoreResult, error) {
			t.Fatal("score should not be called when image is missing")
			return visionservice.ScoreResult{}, nil
		},
	})

	router := gin.New()
	router.POST("/v1/seller/score", handler.Score)

	req := httptest.NewRequest(http.MethodPost, "/v1/seller/score", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestSellerScoreReturnsStructuredResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewSellerHandler(scorerStub{
		score: func(ctx context.Context, input visionservice.ImageInput) (visionservice.ScoreResult, error) {
			if input.Filename != "shop.jpg" {
				t.Fatalf("unexpected filename: %s", input.Filename)
			}
			if input.Size <= 0 {
				t.Fatal("expected non-empty upload")
			}

			return visionservice.ScoreResult{
				Score:      8.7,
				Category:   "fresh_produce",
				Confidence: 0.94,
			}, nil
		},
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("image", "shop.jpg")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	if _, err := part.Write([]byte("fake-image-content")); err != nil {
		t.Fatalf("failed to write multipart body: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	router := gin.New()
	router.POST("/v1/seller/score", handler.Score)

	req := httptest.NewRequest(http.MethodPost, "/v1/seller/score", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response["category"] != "fresh_produce" {
		t.Fatalf("unexpected category: %#v", response["category"])
	}
}

func TestSellerScoreRejectsInvalidImage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewSellerHandler(scorerStub{
		score: func(ctx context.Context, input visionservice.ImageInput) (visionservice.ScoreResult, error) {
			return visionservice.ScoreResult{}, visionservice.ErrInvalidImage
		},
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("image", "shop.txt")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	if _, err := part.Write([]byte("not-an-image")); err != nil {
		t.Fatalf("failed to write multipart body: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	router := gin.New()
	router.POST("/v1/seller/score", handler.Score)

	req := httptest.NewRequest(http.MethodPost, "/v1/seller/score", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestSellerScoreHandlesProviderFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewSellerHandler(scorerStub{
		score: func(ctx context.Context, input visionservice.ImageInput) (visionservice.ScoreResult, error) {
			return visionservice.ScoreResult{}, errors.New("provider failed")
		},
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("image", "shop.jpg")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	if _, err := part.Write([]byte("fake-image-content")); err != nil {
		t.Fatalf("failed to write multipart body: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	router := gin.New()
	router.POST("/v1/seller/score", handler.Score)

	req := httptest.NewRequest(http.MethodPost, "/v1/seller/score", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}
