package vault

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

const algorithmEd25519 = "Ed25519"

type Config struct {
	Address        string
	Token          string
	KVMount        string
	KeysPathPrefix string
	HTTPClient     *http.Client
}

type AccountKey struct {
	PublicKey  string
	Algorithm  string
	VaultPath  string
	PrivateKey string
}

type Client struct {
	address        string
	token          string
	kvMount        string
	keysPathPrefix string
	httpClient     *http.Client
}

type storedAccountKey struct {
	UserID     string `json:"userId"`
	Algorithm  string `json:"algorithm"`
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
	CreatedAt  string `json:"createdAt"`
}

func NewClient(cfg Config) *Client {
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	return &Client{
		address:        strings.TrimRight(cfg.Address, "/"),
		token:          cfg.Token,
		kvMount:        strings.Trim(cfg.KVMount, "/"),
		keysPathPrefix: strings.Trim(cfg.KeysPathPrefix, "/"),
		httpClient:     httpClient,
	}
}

func (c *Client) CreateAccountKey(ctx context.Context, userID string) (AccountKey, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return AccountKey{}, fmt.Errorf("userID is required")
	}

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return AccountKey{}, fmt.Errorf("failed to generate key pair: %w", err)
	}

	key := AccountKey{
		PublicKey:  base64.StdEncoding.EncodeToString(publicKey),
		PrivateKey: base64.StdEncoding.EncodeToString(privateKey),
		Algorithm:  algorithmEd25519,
		VaultPath:  c.secretPath(userID),
	}

	payload := map[string]any{
		"data": map[string]any{
			"userId":     userID,
			"algorithm":  key.Algorithm,
			"publicKey":  key.PublicKey,
			"privateKey": key.PrivateKey,
			"createdAt":  time.Now().UTC().Format(time.RFC3339),
		},
	}

	if err := c.writeKVV2(ctx, key.VaultPath, payload); err != nil {
		return AccountKey{}, err
	}

	return key, nil
}

func (c *Client) DeleteAccountKey(ctx context.Context, vaultPath string) error {
	vaultPath = strings.Trim(vaultPath, "/")
	if vaultPath == "" {
		return nil
	}

	endpoint, err := url.JoinPath(c.address, "v1", c.kvMount, "metadata", vaultPath)
	if err != nil {
		return fmt.Errorf("failed to build Vault metadata endpoint: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to build Vault delete request: %w", err)
	}
	req.Header.Set("X-Vault-Token", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete Vault key: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("vault delete failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}

func (c *Client) SignAccountEvent(ctx context.Context, vaultPath string, message []byte) (string, error) {
	key, err := c.readAccountKey(ctx, vaultPath)
	if err != nil {
		return "", err
	}
	if key.Algorithm != algorithmEd25519 {
		return "", fmt.Errorf("unsupported key algorithm: %s", key.Algorithm)
	}

	privateKeyBytes, err := base64.StdEncoding.DecodeString(key.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode private key: %w", err)
	}
	signature := ed25519.Sign(ed25519.PrivateKey(privateKeyBytes), message)
	return base64.StdEncoding.EncodeToString(signature), nil
}

func (c *Client) secretPath(userID string) string {
	if c.keysPathPrefix == "" {
		return userID
	}
	return path.Join(c.keysPathPrefix, userID)
}

func (c *Client) writeKVV2(ctx context.Context, vaultPath string, payload map[string]any) error {
	endpoint, err := url.JoinPath(c.address, "v1", c.kvMount, "data", vaultPath)
	if err != nil {
		return fmt.Errorf("failed to build Vault endpoint: %w", err)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Vault payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build Vault write request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Vault-Token", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to store Vault key: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("vault write failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return nil
}

func (c *Client) readAccountKey(ctx context.Context, vaultPath string) (storedAccountKey, error) {
	vaultPath = strings.Trim(vaultPath, "/")
	if vaultPath == "" {
		return storedAccountKey{}, fmt.Errorf("vaultPath is required")
	}

	endpoint, err := url.JoinPath(c.address, "v1", c.kvMount, "data", vaultPath)
	if err != nil {
		return storedAccountKey{}, fmt.Errorf("failed to build Vault read endpoint: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return storedAccountKey{}, fmt.Errorf("failed to build Vault read request: %w", err)
	}
	req.Header.Set("X-Vault-Token", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return storedAccountKey{}, fmt.Errorf("failed to read Vault key: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return storedAccountKey{}, fmt.Errorf("vault read failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var response struct {
		Data struct {
			Data storedAccountKey `json:"data"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return storedAccountKey{}, fmt.Errorf("failed to decode Vault read response: %w", err)
	}
	return response.Data.Data, nil
}
