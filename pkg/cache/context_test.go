package cache

import (
	"context"
	"testing"
)

func TestParseRealtimeFlag(t *testing.T) {
	positives := []string{"1", "true", "TRUE", " yes ", "on", "Y"}
	for _, input := range positives {
		if !ParseRealtimeFlag(input) {
			t.Fatalf("expected true for input %q", input)
		}
	}

	negatives := []string{"", "0", "false", "no", "off", "abc"}
	for _, input := range negatives {
		if ParseRealtimeFlag(input) {
			t.Fatalf("expected false for input %q", input)
		}
	}
}

func TestBypassContext(t *testing.T) {
	if ShouldBypass(nil) {
		t.Fatalf("nil context must not bypass cache")
	}
	if ShouldBypass(context.Background()) {
		t.Fatalf("background context must not bypass cache")
	}

	ctx := WithBypass(context.Background(), true)
	if !ShouldBypass(ctx) {
		t.Fatalf("expected bypass cache=true")
	}
}
