package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadUsesDefaultPort(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("FIREBASE_CREDENTIALS_FILE", "/tmp/firebase.json")
	t.Setenv("FIREBASE_PROJECT_ID", "demo-project")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("VAULT_ENABLED", "false")
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
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("VAULT_ENABLED", "false")
	t.Setenv("OPENAI_API_KEY", "test-key")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoadRequiresOpenAIAPIKey(t *testing.T) {
	t.Setenv("PORT", "8081")
	t.Setenv("FIREBASE_CREDENTIALS_FILE", "/tmp/firebase.json")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("VAULT_ENABLED", "false")
	t.Setenv("OPENAI_API_KEY", "")

	cfg, err := Load()
	if err == nil {
		if cfg.HasVisionProvider() {
			t.Fatal("expected vision provider to be disabled without OPENAI_API_KEY")
		}
	}
}

func TestValidateVisionRejectsUnsupportedProvider(t *testing.T) {
	cfg := Config{
		Port:                    "8080",
		FirebaseCredentialsFile: "/tmp/firebase.json",
		AIProvider:              "gemini",
		OpenAIAPIKey:            "test-key",
		OpenAIBaseURL:           "https://api.openai.com/v1",
		OpenAIModel:             "gpt-4o-mini",
	}

	err := cfg.ValidateVision()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoadReadsDotEnvFile(t *testing.T) {
	tempDir := t.TempDir()
	envFile := filepath.Join(tempDir, ".env")

	content := []byte("FIREBASE_CREDENTIALS_FILE=./secrets/firebase-service-account.json\nFIREBASE_PROJECT_ID=demo-from-env-file\nJWT_SECRET=from-env-file\n")
	if err := os.WriteFile(envFile, content, 0o644); err != nil {
		t.Fatalf("failed to write .env file: %v", err)
	}

	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(workingDir)
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}

	for _, key := range []string{
		"FIREBASE_CREDENTIALS_FILE",
		"FIREBASE_PROJECT_ID",
		"JWT_SECRET",
		"VAULT_ENABLED",
		"OPENAI_API_KEY",
		"PORT",
	} {
		originalValue, hadValue := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("failed to unset %s: %v", key, err)
		}
		keyCopy := key
		valueCopy := originalValue
		hadValueCopy := hadValue
		t.Cleanup(func() {
			if !hadValueCopy {
				_ = os.Unsetenv(keyCopy)
				return
			}
			_ = os.Setenv(keyCopy, valueCopy)
		})
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if cfg.FirebaseCredentialsFile != "./secrets/firebase-service-account.json" {
		t.Fatalf("unexpected FIREBASE_CREDENTIALS_FILE: %s", cfg.FirebaseCredentialsFile)
	}
	if cfg.FirebaseProjectID != "demo-from-env-file" {
		t.Fatalf("unexpected FIREBASE_PROJECT_ID: %s", cfg.FirebaseProjectID)
	}
}

func TestValidateRequiresVaultSettingsWhenEnabled(t *testing.T) {
	cfg := Config{
		Port:                    "8080",
		FirebaseCredentialsFile: "/tmp/firebase.json",
		JWTSecret:               "test-secret",
		VaultEnabled:            true,
		VaultAddr:               "",
		VaultToken:              "",
		VaultKVMount:            "secret",
		VaultKeysPathPrefix:     "account-keys",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestValidateRequiresBesuSignerWhenEnabled(t *testing.T) {
	cfg := Config{
		Port:                    "8080",
		FirebaseCredentialsFile: "/tmp/firebase.json",
		JWTSecret:               "test-secret",
		BesuEnabled:             true,
		BesuRPCURL:              "http://127.0.0.1:8545",
		BesuContractAddress:     "0x123",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestValidateAllowsBesuPrivateKeyWithoutFromAddress(t *testing.T) {
	cfg := Config{
		Port:                    "8080",
		FirebaseCredentialsFile: "/tmp/firebase.json",
		JWTSecret:               "test-secret",
		BesuEnabled:             true,
		BesuRPCURL:              "http://127.0.0.1:8545",
		BesuContractAddress:     "0x123",
		BesuPrivateKey:          "abcd",
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}
