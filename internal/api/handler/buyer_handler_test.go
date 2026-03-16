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

	buyerservice "vngrocery/internal/service/buyer"
)

type buyerCheckStub struct {
	check func(ctx context.Context, input buyerservice.CheckInput) (buyerservice.CheckResult, error)
}

func (s buyerCheckStub) Check(ctx context.Context, input buyerservice.CheckInput) (buyerservice.CheckResult, error) {
	return s.check(ctx, input)
}

func TestBuyerCheckAllowsMissingPledgeID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewBuyerHandler(buyerCheckStub{
		check: func(ctx context.Context, input buyerservice.CheckInput) (buyerservice.CheckResult, error) {
			if input.PledgeID != "" {
				t.Fatalf("expected empty pledge id, got %q", input.PledgeID)
			}
			return buyerservice.CheckResult{
				PolicyVersion:    "trust_policy_v1",
				HasPledge:        false,
				Trusted:          false,
				Verdict:          "no_pledge",
				ActualScore:      6.2,
				ActualCategory:   "bruised_fruit",
				ActualConfidence: 0.86,
				Reasons:          []string{"no_seller_pledge"},
			}, nil
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

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response["hasPledge"] != false {
		t.Fatalf("expected hasPledge=false, got %v", response["hasPledge"])
	}
	if response["verdict"] != "no_pledge" {
		t.Fatalf("unexpected verdict: %v", response["verdict"])
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
				PolicyVersion:    "trust_policy_v1",
				HasPledge:        true,
				PledgeID:         "pledge-1",
				Trusted:          true,
				Verdict:          "trusted",
				PledgedScore:     8.5,
				ActualScore:      8.1,
				ScoreDelta:       -0.4,
				ScoreDeltaAbs:    0.4,
				PledgedCategory:  "fresh_produce",
				ActualCategory:   "fresh_produce",
				ActualConfidence: 0.89,
				CategoryMatch:    true,
				Reasons:          []string{},
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

	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response["policyVersion"] != "trust_policy_v1" {
		t.Fatalf("unexpected policyVersion: %v", response["policyVersion"])
	}
	if response["categoryMatch"] != true {
		t.Fatalf("expected categoryMatch=true, got %v", response["categoryMatch"])
	}
	if response["hasPledge"] != true {
		t.Fatalf("expected hasPledge=true, got %v", response["hasPledge"])
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
