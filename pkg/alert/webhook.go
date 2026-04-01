package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type WebhookClient struct {
	url        string
	httpClient *http.Client
}

type IntegrityMismatchPayload struct {
	Event            string     `json:"event"`
	PledgeID         string     `json:"pledgeId"`
	ShopID           string     `json:"shopId"`
	CreatedByUserID  string     `json:"createdByUserId"`
	DataHash         string     `json:"dataHash"`
	ChainTxHash      string     `json:"chainTxHash,omitempty"`
	IntegrityStatus  string     `json:"integrityStatus"`
	DetectedAt       time.Time  `json:"detectedAt"`
	OnChainDataHash  string     `json:"onChainDataHash,omitempty"`
	OnChainVersion   int        `json:"onChainVersion,omitempty"`
	OnChainTimestamp *time.Time `json:"onChainTimestamp,omitempty"`
}

func NewWebhookClient(url string, timeout time.Duration) *WebhookClient {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &WebhookClient{
		url:        strings.TrimSpace(url),
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *WebhookClient) NotifyIntegrityMismatch(ctx context.Context, payload IntegrityMismatchPayload) error {
	if c == nil || c.url == "" {
		return nil
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to deliver webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}
