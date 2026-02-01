package config

import (
	"errors"
	"fmt"
	"os"
)

const defaultPort = "8080"

type Config struct {
	Port                    string
	FirebaseProjectID       string
	FirebaseCredentialsFile string
}

func Load() (Config, error) {
	cfg := Config{
		Port:                    getEnvOrDefault("PORT", defaultPort),
		FirebaseProjectID:       os.Getenv("FIREBASE_PROJECT_ID"),
		FirebaseCredentialsFile: os.Getenv("FIREBASE_CREDENTIALS_FILE"),
	}

	if cfg.FirebaseCredentialsFile == "" {
		return Config{}, errors.New("FIREBASE_CREDENTIALS_FILE is required")
	}

	return cfg, nil
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
