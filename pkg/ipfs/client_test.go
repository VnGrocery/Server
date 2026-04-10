package ipfs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAddBytesReturnsCIDAndGatewayURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v0/add" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Hash":"bafy-test-cid"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "http://127.0.0.1:8080")
	result, err := client.AddBytes(context.Background(), "test.jpg", []byte("hello"))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.CID != "bafy-test-cid" {
		t.Fatalf("unexpected cid: %s", result.CID)
	}
	if result.GatewayURL != "http://127.0.0.1:8080/ipfs/bafy-test-cid" {
		t.Fatalf("unexpected gateway url: %s", result.GatewayURL)
	}
}
