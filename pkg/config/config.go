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
