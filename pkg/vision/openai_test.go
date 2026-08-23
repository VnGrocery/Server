package vision

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	visionservice "vngrocery/internal/service/vision"
	"vngrocery/pkg/config"
)

func TestOpenAIClientScoreImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/responses" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected auth header: %s", got)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		if !strings.Contains(string(body), "\"json_schema\"") {
			t.Fatalf("expected structured output request, got %s", string(body))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output_text":"{\"score\":8.5,\"category\":\"fresh_produce\",\"confidence\":0.93}"}`))
	}))
	defer server.Close()

	client := NewOpenAIClient(config.Config{
		OpenAIAPIKey:  "test-key",
		OpenAIBaseURL: server.URL,
		OpenAIModel:   "gpt-4o-mini",
	})

	result, err := client.ScoreImage(context.Background(), visionservice.ImagePayload{
		Filename:    "store.jpg",
		ContentType: "image/jpeg",
		Data:        []byte{0xff, 0xd8, 0xff},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.Score != 8.5 {
		t.Fatalf("unexpected score: %v", result.Score)
	}
}

// A refused key, a spent quota or an OpenAI outage is not our server failing.
// Every one of these used to come back as a plain error, which the API layer
// reported as 500 - so the seller was told the app had broken.
func TestOpenAIClientClassifiesProviderErrors(t *testing.T) {
	cases := []struct {
		name   string
		status int
		want   error
	}{
		{"rejected key", http.StatusUnauthorized, visionservice.ErrProviderUnavailable},
		{"spent quota", http.StatusTooManyRequests, visionservice.ErrProviderUnavailable},
		{"provider outage", http.StatusBadGateway, visionservice.ErrProviderUnavailable},
		{"image refused", http.StatusBadRequest, visionservice.ErrInvalidImage},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(testCase.status)
				_, _ = w.Write([]byte(`{"error":{"message":"nope"}}`))
			}))
			defer server.Close()

			client := NewOpenAIClient(config.Config{
				OpenAIAPIKey:  "test-key",
				OpenAIBaseURL: server.URL,
				OpenAIModel:   "gpt-4o-mini",
			})

			_, err := client.ScoreImage(context.Background(), visionservice.ImagePayload{
				Filename:    "store.jpg",
				ContentType: "image/jpeg",
				Data:        []byte{0xff, 0xd8, 0xff},
			})
			if !errors.Is(err, testCase.want) {
				t.Fatalf("expected %v, got %v", testCase.want, err)
			}
		})
	}
}
