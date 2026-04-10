package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
	"time"
)

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

type IntegrityNotifier interface {
	NotifyIntegrityMismatch(ctx context.Context, payload IntegrityMismatchPayload) error
}

type MultiNotifier struct {
	notifiers []IntegrityNotifier
}

func NewMultiNotifier(notifiers ...IntegrityNotifier) *MultiNotifier {
	filtered := make([]IntegrityNotifier, 0, len(notifiers))
	for _, notifier := range notifiers {
		if notifier == nil {
			continue
		}
		filtered = append(filtered, notifier)
	}
	return &MultiNotifier{notifiers: filtered}
}

func (m *MultiNotifier) NotifyIntegrityMismatch(ctx context.Context, payload IntegrityMismatchPayload) error {
	if m == nil || len(m.notifiers) == 0 {
		return nil
	}
	errors := make([]string, 0, len(m.notifiers))
	for _, notifier := range m.notifiers {
		if err := notifier.NotifyIntegrityMismatch(ctx, payload); err != nil {
			errors = append(errors, err.Error())
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("alert delivery failed: %s", strings.Join(errors, "; "))
	}
	return nil
}

type WebhookClient struct {
	url        string
	httpClient *http.Client
}

func NewWebhookClient(url string, timeout time.Duration) *WebhookClient {
	if strings.TrimSpace(url) == "" {
		return nil
	}
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
	return postJSON(ctx, c.httpClient, c.url, payload)
}

type SlackClient struct {
	webhookURL string
	httpClient *http.Client
}

func NewSlackClient(webhookURL string, timeout time.Duration) *SlackClient {
	if strings.TrimSpace(webhookURL) == "" {
		return nil
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &SlackClient{
		webhookURL: strings.TrimSpace(webhookURL),
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *SlackClient) NotifyIntegrityMismatch(ctx context.Context, payload IntegrityMismatchPayload) error {
	if c == nil || c.webhookURL == "" {
		return nil
	}
	body := map[string]any{
		"text": fmt.Sprintf(
			"Integrity mismatch detected\nshop=%s pledge=%s status=%s tx=%s detectedAt=%s",
			payload.ShopID,
			payload.PledgeID,
			payload.IntegrityStatus,
			payload.ChainTxHash,
			payload.DetectedAt.Format(time.RFC3339),
		),
	}
	return postJSON(ctx, c.httpClient, c.webhookURL, body)
}

type TelegramClient struct {
	apiURL     string
	chatID     string
	httpClient *http.Client
}

func NewTelegramClient(botToken, chatID string, timeout time.Duration) *TelegramClient {
	if strings.TrimSpace(botToken) == "" || strings.TrimSpace(chatID) == "" {
		return nil
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &TelegramClient{
		apiURL:     fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", strings.TrimSpace(botToken)),
		chatID:     strings.TrimSpace(chatID),
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *TelegramClient) NotifyIntegrityMismatch(ctx context.Context, payload IntegrityMismatchPayload) error {
	if c == nil || c.apiURL == "" {
		return nil
	}
	form := url.Values{}
	form.Set("chat_id", c.chatID)
	form.Set(
		"text",
		fmt.Sprintf(
			"Integrity mismatch detected\nshop=%s\npledge=%s\nstatus=%s\ntx=%s\ndetectedAt=%s",
			payload.ShopID,
			payload.PledgeID,
			payload.IntegrityStatus,
			payload.ChainTxHash,
			payload.DetectedAt.Format(time.RFC3339),
		),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("failed to build telegram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to deliver telegram alert: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram returned status %d", resp.StatusCode)
	}
	return nil
}

type SMTPClient struct {
	host     string
	port     string
	username string
	password string
	from     string
	to       []string
}

func NewSMTPClient(host, port, username, password, from, to string) *SMTPClient {
	if strings.TrimSpace(host) == "" || strings.TrimSpace(from) == "" || strings.TrimSpace(to) == "" {
		return nil
	}
	if strings.TrimSpace(port) == "" {
		port = "587"
	}
	recipients := make([]string, 0)
	for _, value := range strings.Split(to, ",") {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			recipients = append(recipients, trimmed)
		}
	}
	if len(recipients) == 0 {
		return nil
	}
	return &SMTPClient{
		host:     strings.TrimSpace(host),
		port:     strings.TrimSpace(port),
		username: strings.TrimSpace(username),
		password: password,
		from:     strings.TrimSpace(from),
		to:       recipients,
	}
}

func (c *SMTPClient) NotifyIntegrityMismatch(ctx context.Context, payload IntegrityMismatchPayload) error {
	if c == nil {
		return nil
	}
	subject := fmt.Sprintf("VNGrocery integrity mismatch for pledge %s", payload.PledgeID)
	body := fmt.Sprintf(
		"Integrity mismatch detected\n\nShop: %s\nPledge: %s\nStatus: %s\nChain TX: %s\nDetected at: %s\nOn-chain version: %d\n",
		payload.ShopID,
		payload.PledgeID,
		payload.IntegrityStatus,
		payload.ChainTxHash,
		payload.DetectedAt.Format(time.RFC3339),
		payload.OnChainVersion,
	)
	message := "To: " + strings.Join(c.to, ",") + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		body

	address := c.host + ":" + c.port
	var auth smtp.Auth
	if c.username != "" {
		auth = smtp.PlainAuth("", c.username, c.password, c.host)
	}

	done := make(chan error, 1)
	go func() {
		done <- smtp.SendMail(address, auth, c.from, c.to, []byte(message))
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		if err != nil {
			return fmt.Errorf("failed to deliver smtp alert: %w", err)
		}
		return nil
	}
}

func postJSON(ctx context.Context, client *http.Client, targetURL string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal alert payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to build alert request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to deliver alert: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("alert endpoint returned status %d", resp.StatusCode)
	}
	return nil
}
