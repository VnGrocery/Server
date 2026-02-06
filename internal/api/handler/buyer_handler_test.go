package handler

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	buyerservice "vngrocery/internal/service/buyer"
)

type buyerCheckStub struct {
	check func(ctx context.Context, input buyerservice.CheckInput) (buyerservice.CheckResult, error)
}

func (s buyerCheckStub) Check(ctx context.Context, input buyerservice.CheckInput) (buyerservice.CheckResult, error) {
	return s.check(ctx, input)
}

func TestBuyerCheckRequiresPledgeID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewBuyerHandler(buyerCheckStub{
		check: func(ctx context.Context, input buyerservice.CheckInput) (buyerservice.CheckResult, error) {
			t.Fatal("check should not be called when pledgeId is missing")
			return buyerservice.CheckResult{}, nil
		},
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("image", "shop.jpg")
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	if _, err := part.Write([]byte("fake-image-content")); err != nil {
		t.Fatalf("failed to write image: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	router := gin.New()
	router.POST("/v1/buyer/check", handler.Check)

	req := httptest.NewRequest(http.MethodPost, "/v1/buyer/check", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestBuyerCheckReturnsComparison(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewBuyerHandler(buyerCheckStub{
		check: func(ctx context.Context, input buyerservice.CheckInput) (buyerservice.CheckResult, error) {
			if input.PledgeID != "pledge-1" {
				t.Fatalf("unexpected pledge id: %s", input.PledgeID)
			}
			return buyerservice.CheckResult{
				PledgeID:         "pledge-1",
				Trusted:          true,
				Verdict:          "trusted",
				PledgedScore:     8.5,
				ActualScore:      8.1,
				ScoreDelta:       -0.4,
				PledgedCategory:  "fresh_produce",
				ActualCategory:   "fresh_produce",
				ActualConfidence: 0.89,
			}, nil
		},
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("pledgeId", "pledge-1"); err != nil {
		t.Fatalf("failed to write pledgeId: %v", err)
	}
	part, err := writer.CreateFormFile("image", "shop.jpg")
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	if _, err := part.Write([]byte("fake-image-content")); err != nil {
		t.Fatalf("failed to write image: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	router := gin.New()
	router.POST("/v1/buyer/check", handler.Check)

	req := httptest.NewRequest(http.MethodPost, "/v1/buyer/check", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestBuyerCheckHandlesInvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewBuyerHandler(buyerCheckStub{
		check: func(ctx context.Context, input buyerservice.CheckInput) (buyerservice.CheckResult, error) {
			return buyerservice.CheckResult{}, errors.New("provider failed")
		},
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("pledgeId", "pledge-1"); err != nil {
		t.Fatalf("failed to write pledgeId: %v", err)
	}
	part, err := writer.CreateFormFile("image", "shop.jpg")
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	if _, err := part.Write([]byte("fake-image-content")); err != nil {
		t.Fatalf("failed to write image: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	router := gin.New()
	router.POST("/v1/buyer/check", handler.Check)

	req := httptest.NewRequest(http.MethodPost, "/v1/buyer/check", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}
