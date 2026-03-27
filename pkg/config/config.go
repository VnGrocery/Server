package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

const defaultPort = "8080"

type Config struct {
	Port                    string
	FirebaseProjectID       string
	FirebaseCredentialsFile string
	JWTSecret               string
	GoogleClientID          string
	VaultEnabled            bool
	VaultAddr               string
	VaultToken              string
	VaultKVMount            string
	VaultKeysPathPrefix     string
	BesuEnabled             bool
	BesuRPCURL              string
	BesuChainID             string
	BesuContractAddress     string
	BesuFromAddress         string
	BesuGasLimit            string
	BesuReceiptTimeoutSec   string
	BesuPendingIntervalSec  string
	BesuVerifyIntervalSec   string
	BesuPendingBatchSize    string
	BesuVerifyBatchSize     string
	AIProvider              string
	OpenAIAPIKey            string
	OpenAIBaseURL           string
	OpenAIModel             string
	VisionPromptVersion     string
}

func Load() (Config, error) {
	loadDotEnvIfPresent()

	cfg := Config{
		Port:                    getEnvOrDefault("PORT", defaultPort),
		FirebaseProjectID:       os.Getenv("FIREBASE_PROJECT_ID"),
		FirebaseCredentialsFile: os.Getenv("FIREBASE_CREDENTIALS_FILE"),
		JWTSecret:               os.Getenv("JWT_SECRET"),
		GoogleClientID:          os.Getenv("GOOGLE_CLIENT_ID"),
		VaultEnabled:            os.Getenv("VAULT_ENABLED") == "true",
		VaultAddr:               os.Getenv("VAULT_ADDR"),
		VaultToken:              os.Getenv("VAULT_TOKEN"),
		VaultKVMount:            getEnvOrDefault("VAULT_KV_MOUNT", "secret"),
		VaultKeysPathPrefix:     getEnvOrDefault("VAULT_KEYS_PATH_PREFIX", "account-keys"),
		BesuEnabled:             os.Getenv("BESU_ENABLED") == "true",
		BesuRPCURL:              os.Getenv("BESU_RPC_URL"),
		BesuChainID:             os.Getenv("BESU_CHAIN_ID"),
		BesuContractAddress:     os.Getenv("BESU_CONTRACT_ADDRESS"),
		BesuFromAddress:         os.Getenv("BESU_FROM_ADDRESS"),
		BesuGasLimit:            getEnvOrDefault("BESU_GAS_LIMIT", "250000"),
		BesuReceiptTimeoutSec:   getEnvOrDefault("BESU_RECEIPT_TIMEOUT_SEC", "15"),
		BesuPendingIntervalSec:  getEnvOrDefault("BESU_PENDING_INTERVAL_SEC", "10"),
		BesuVerifyIntervalSec:   getEnvOrDefault("BESU_VERIFY_INTERVAL_SEC", "60"),
		BesuPendingBatchSize:    getEnvOrDefault("BESU_PENDING_BATCH_SIZE", "25"),
		BesuVerifyBatchSize:     getEnvOrDefault("BESU_VERIFY_BATCH_SIZE", "50"),
		AIProvider:              getEnvOrDefault("AI_PROVIDER", "openai"),
		OpenAIAPIKey:            os.Getenv("OPENAI_API_KEY"),
		OpenAIBaseURL:           getEnvOrDefault("OPENAI_BASE_URL", "https://api.openai.com/v1"),
		OpenAIModel:             getEnvOrDefault("OPENAI_MODEL", "gpt-4o-mini"),
		VisionPromptVersion:     getEnvOrDefault("VISION_PROMPT_VERSION", "v1"),
	}

	return cfg, cfg.Validate()
}

func loadDotEnvIfPresent() {
	_ = godotenv.Load()
}

func getEnvOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func (c Config) Validate() error {
	if c.FirebaseCredentialsFile == "" {
		return errors.New("FIREBASE_CREDENTIALS_FILE is required")
	}
	if c.Port == "" {
		return fmt.Errorf("PORT must not be empty")
	}
	if c.JWTSecret == "" {
		return errors.New("JWT_SECRET is required")
	}
	if c.VaultEnabled {
		if c.VaultAddr == "" {
			return errors.New("VAULT_ADDR is required when VAULT_ENABLED=true")
		}
		if c.VaultToken == "" {
			return errors.New("VAULT_TOKEN is required when VAULT_ENABLED=true")
		}
		if c.VaultKVMount == "" {
			return errors.New("VAULT_KV_MOUNT is required when VAULT_ENABLED=true")
		}
		if c.VaultKeysPathPrefix == "" {
			return errors.New("VAULT_KEYS_PATH_PREFIX is required when VAULT_ENABLED=true")
		}
	}
	if c.BesuEnabled {
		if c.BesuRPCURL == "" {
			return errors.New("BESU_RPC_URL is required when BESU_ENABLED=true")
		}
		if c.BesuContractAddress == "" {
			return errors.New("BESU_CONTRACT_ADDRESS is required when BESU_ENABLED=true")
		}
		if c.BesuFromAddress == "" {
			return errors.New("BESU_FROM_ADDRESS is required when BESU_ENABLED=true")
		}
	}

	return nil
}

func (c Config) HasVisionProvider() bool {
	return c.OpenAIAPIKey != ""
}

func (c Config) ValidateVision() error {
	if !c.HasVisionProvider() {
		return nil
	}
	if c.AIProvider == "" {
		return errors.New("AI_PROVIDER must not be empty")
	}
	if c.AIProvider != "openai" {
		return fmt.Errorf("unsupported AI_PROVIDER: %s", c.AIProvider)
	}
	if c.OpenAIBaseURL == "" {
		return errors.New("OPENAI_BASE_URL must not be empty")
	}
	if c.OpenAIModel == "" {
		return errors.New("OPENAI_MODEL must not be empty")
	}

	return nil
}
