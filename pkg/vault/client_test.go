package vault

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateAccountKeyWritesKVV2Secret(t *testing.T) {
	t.Helper()

	var capturedPath string
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		if got := r.Header.Get("X-Vault-Token"); got != "test-token" {
			t.Fatalf("unexpected vault token: %s", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(Config{
		Address:        server.URL,
		Token:          "test-token",
		KVMount:        "secret",
		KeysPathPrefix: "account-keys",
	})

	key, err := client.CreateAccountKey(context.Background(), "user-123")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if capturedPath != "/v1/secret/data/account-keys/user-123" {
		t.Fatalf("unexpected path: %s", capturedPath)
	}
	if key.Algorithm != algorithmEd25519 {
		t.Fatalf("unexpected algorithm: %s", key.Algorithm)
	}

	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested data payload, got %#v", payload["data"])
	}
	for _, field := range []string{"publicKey", "privateKey", "algorithm", "userId", "createdAt"} {
		value, ok := data[field].(string)
		if !ok || strings.TrimSpace(value) == "" {
			t.Fatalf("expected non-empty string for %s, got %#v", field, data[field])
		}
	}
}

func TestDeleteAccountKeyUsesMetadataEndpoint(t *testing.T) {
	t.Helper()

	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		if r.Method != http.MethodDelete {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(Config{
		Address:        server.URL,
		Token:          "test-token",
		KVMount:        "secret",
		KeysPathPrefix: "account-keys",
	})

	if err := client.DeleteAccountKey(context.Background(), "account-keys/user-123"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if capturedPath != "/v1/secret/metadata/account-keys/user-123" {
		t.Fatalf("unexpected path: %s", capturedPath)
	}
}

func TestSignAccountEventReadsAndSignsWithStoredKey(t *testing.T) {
	t.Helper()

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/v1/secret/data/account-keys/user-123" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if _, err := w.Write([]byte(`{"data":{"data":{"userId":"user-123","algorithm":"Ed25519","publicKey":"` + base64.StdEncoding.EncodeToString(publicKey) + `","privateKey":"` + base64.StdEncoding.EncodeToString(privateKey) + `"}}}`)); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	client := NewClient(Config{
		Address:        server.URL,
		Token:          "test-token",
		KVMount:        "secret",
		KeysPathPrefix: "account-keys",
	})

	signature, err := client.SignAccountEvent(context.Background(), "account-keys/user-123", []byte("hello"))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if strings.TrimSpace(signature) == "" {
		t.Fatal("expected signature, got empty string")
	}
}
