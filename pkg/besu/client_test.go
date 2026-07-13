package besu

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientFailsOverToHealthyRPCEndpoint(t *testing.T) {
	failed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer failed.Close()

	healthyHits := 0
	healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		healthyHits++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":null}`))
	}))
	defer healthy.Close()

	client := NewClient(Config{RPCURLs: []string{failed.URL, healthy.URL}})
	result, err := client.Receipt(context.Background(), "0xabc")
	if err != nil {
		t.Fatalf("expected failover to succeed, got %v", err)
	}
	if result.Mined {
		t.Fatalf("expected missing receipt to remain unmined")
	}
	if healthyHits != 1 {
		t.Fatalf("expected one request to healthy endpoint, got %d", healthyHits)
	}
}

func TestNormalizeRPCURLsSplitsAndDeduplicates(t *testing.T) {
	got := normalizeRPCURLs([]string{" http://a:8545,http://b:8545 ", "http://a:8545"})
	if len(got) != 2 || got[0] != "http://a:8545" || got[1] != "http://b:8545" {
		t.Fatalf("unexpected URLs: %#v", got)
	}
}
