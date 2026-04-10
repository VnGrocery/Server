package ipfs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type Client struct {
	apiURL     string
	gatewayURL string
	httpClient *http.Client
}

type AddResult struct {
	CID        string
	GatewayURL string
}

func NewClient(apiURL, gatewayURL string) *Client {
	return &Client{
		apiURL:     strings.TrimRight(strings.TrimSpace(apiURL), "/"),
		gatewayURL: strings.TrimRight(strings.TrimSpace(gatewayURL), "/"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) AddBytes(ctx context.Context, filename string, data []byte) (AddResult, error) {
	if c == nil || c.apiURL == "" {
		return AddResult{}, fmt.Errorf("IPFS_API_URL is required")
	}
	if strings.TrimSpace(filename) == "" {
		filename = "upload.bin"
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return AddResult{}, fmt.Errorf("failed to create multipart body: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return AddResult{}, fmt.Errorf("failed to write file body: %w", err)
	}
	if err := writer.Close(); err != nil {
		return AddResult{}, fmt.Errorf("failed to finalize multipart body: %w", err)
	}

	endpoint, err := url.JoinPath(c.apiURL, "api", "v0", "add")
	if err != nil {
		return AddResult{}, fmt.Errorf("failed to build ipfs add endpoint: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"?pin=true&cid-version=1", &body)
	if err != nil {
		return AddResult{}, fmt.Errorf("failed to build ipfs add request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return AddResult{}, fmt.Errorf("failed to upload to ipfs: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if err != nil {
		return AddResult{}, fmt.Errorf("failed to read ipfs response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return AddResult{}, fmt.Errorf("ipfs add failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var payload struct {
		Hash string `json:"Hash"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return AddResult{}, fmt.Errorf("failed to decode ipfs add response: %w", err)
	}
	result := AddResult{CID: payload.Hash}
	if c.gatewayURL != "" && payload.Hash != "" {
		result.GatewayURL = c.gatewayURL + "/ipfs/" + path.Clean("/" + payload.Hash)[1:]
	}
	return result, nil
}
