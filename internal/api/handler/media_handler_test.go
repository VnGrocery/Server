package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type imageUploaderStub struct {
	addBytes func(ctx context.Context, filename string, data []byte) (ImageUploadResult, error)
}

func (s imageUploaderStub) AddBytes(ctx context.Context, filename string, data []byte) (ImageUploadResult, error) {
	return s.addBytes(ctx, filename, data)
}

func TestUploadImage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewMediaHandler(imageUploaderStub{
		addBytes: func(ctx context.Context, filename string, data []byte) (ImageUploadResult, error) {
			if filename != "proof.png" {
				t.Fatalf("unexpected filename: %s", filename)
			}
			return ImageUploadResult{CID: "cid-1", GatewayURL: "https://ipfs.example/ipfs/cid-1"}, nil
		},
	}, newMediaUploadConfig(1024*1024, []string{"image/png"}))

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("image", "proof.png")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}); err != nil {
		t.Fatalf("write image: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	router := gin.New()
	router.POST("/v1/media/images", handler.UploadImage)

	req := httptest.NewRequest(http.MethodPost, "/v1/media/images", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["imageCid"] != "cid-1" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}
