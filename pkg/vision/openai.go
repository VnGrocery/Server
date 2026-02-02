package vision

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	visionservice "vngrocery/internal/service/vision"
	"vngrocery/pkg/config"
)

const openAIResponsesPath = "/responses"

type OpenAIClient struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewOpenAIClient(cfg config.Config) *OpenAIClient {
	return &OpenAIClient{
		baseURL: strings.TrimRight(cfg.OpenAIBaseURL, "/"),
		apiKey:  cfg.OpenAIAPIKey,
		model:   cfg.OpenAIModel,
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

func (c *OpenAIClient) ScoreImage(ctx context.Context, image visionservice.ImagePayload) (visionservice.ScoreResult, error) {
	requestBody := openAIResponseRequest{
		Model: c.model,
		Input: []openAIInputItem{
			{
				Role: "system",
				Content: []openAIContentItem{
					{Type: "input_text", Text: sellerScoringInstruction},
				},
			},
			{
				Role: "user",
				Content: []openAIContentItem{
					{Type: "input_text", Text: "Analyze this grocery shop image and return a strict JSON assessment."},
					{Type: "input_image", ImageURL: dataURL(image.ContentType, image.Data), Detail: "high"},
				},
			},
		},
		Text: openAITextConfig{
			Format: openAIFormat{
				Type:        "json_schema",
				Name:        "seller_score",
				Description: "Structured quality score for a grocery shop image.",
				Strict:      true,
				Schema: map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"properties": map[string]any{
						"score": map[string]any{
							"type":    "number",
							"minimum": 0,
							"maximum": 10,
						},
						"category": map[string]any{
							"type": "string",
						},
						"confidence": map[string]any{
							"type":    "number",
							"minimum": 0,
							"maximum": 1,
						},
					},
					"required": []string{"score", "category", "confidence"},
				},
			},
		},
	}

	payload, err := json.Marshal(requestBody)
	if err != nil {
		return visionservice.ScoreResult{}, fmt.Errorf("failed to marshal OpenAI request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+openAIResponsesPath, bytes.NewReader(payload))
	if err != nil {
		return visionservice.ScoreResult{}, fmt.Errorf("failed to create OpenAI request: %w", err)
	}

	request.Header.Set("Authorization", "Bearer "+c.apiKey)
	request.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return visionservice.ScoreResult{}, fmt.Errorf("%w: failed to call OpenAI Responses API: %v", visionservice.ErrProviderUnavailable, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return visionservice.ScoreResult{}, fmt.Errorf("failed to read OpenAI response body: %w", err)
	}

	if response.StatusCode >= http.StatusBadRequest {
		return visionservice.ScoreResult{}, fmt.Errorf("OpenAI API returned status %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed openAIResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return visionservice.ScoreResult{}, fmt.Errorf("failed to decode OpenAI response: %w", err)
	}

	if strings.TrimSpace(parsed.OutputText) == "" {
		return visionservice.ScoreResult{}, errors.New("OpenAI response did not contain output_text")
	}

	var result visionservice.ScoreResult
	if err := json.Unmarshal([]byte(parsed.OutputText), &result); err != nil {
		return visionservice.ScoreResult{}, fmt.Errorf("failed to decode structured OpenAI output: %w", err)
	}

	return result, nil
}

func dataURL(contentType string, data []byte) string {
	return fmt.Sprintf("data:%s;base64,%s", contentType, base64.StdEncoding.EncodeToString(data))
}

const sellerScoringInstruction = "You are a food retail quality assessor. Analyze the uploaded grocery shop image and return JSON only. " +
	"Estimate overall shop quality on a 0 to 10 scale, where higher means cleaner, fresher, better organized, and more trustworthy. " +
	"Set category to a short snake_case label that best describes the observed scene quality or merchandising state. " +
	"Set confidence to a value between 0 and 1."

type openAIResponseRequest struct {
	Model string            `json:"model"`
	Input []openAIInputItem `json:"input"`
	Text  openAITextConfig  `json:"text"`
}

type openAIInputItem struct {
	Role    string              `json:"role"`
	Content []openAIContentItem `json:"content"`
}

type openAIContentItem struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

type openAITextConfig struct {
	Format openAIFormat `json:"format"`
}

type openAIFormat struct {
	Type        string         `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Strict      bool           `json:"strict"`
	Schema      map[string]any `json:"schema"`
}

type openAIResponse struct {
	OutputText string `json:"output_text"`
}
