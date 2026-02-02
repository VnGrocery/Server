package config

import "testing"

func TestLoadUsesDefaultPort(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("FIREBASE_CREDENTIALS_FILE", "/tmp/firebase.json")
	t.Setenv("FIREBASE_PROJECT_ID", "demo-project")
	t.Setenv("OPENAI_API_KEY", "test-key")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if cfg.Port != defaultPort {
		t.Fatalf("expected default port %s, got %s", defaultPort, cfg.Port)
	}
}

func TestLoadRequiresCredentialsFile(t *testing.T) {
	t.Setenv("PORT", "8081")
	t.Setenv("FIREBASE_CREDENTIALS_FILE", "")
	t.Setenv("OPENAI_API_KEY", "test-key")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoadRequiresOpenAIAPIKey(t *testing.T) {
	t.Setenv("PORT", "8081")
	t.Setenv("FIREBASE_CREDENTIALS_FILE", "/tmp/firebase.json")
	t.Setenv("OPENAI_API_KEY", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
