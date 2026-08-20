package handler

import (
	"strconv"
	"testing"

	shopsvc "vngrocery/internal/service/shop"
)

func TestParseNearQuery(t *testing.T) {
	t.Run("all three omitted means no filter, not an error", func(t *testing.T) {
		near, err := parseNearQuery("", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if near.Valid() {
			t.Fatalf("expected an unset filter, got %+v", near)
		}
	})

	t.Run("reads a usable circle", func(t *testing.T) {
		near, err := parseNearQuery("10.7721", "106.6980", "5")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !near.Valid() || near.RadiusKm != 5 {
			t.Fatalf("unexpected filter %+v", near)
		}
	})

	t.Run("half a request is rejected rather than half-applied", func(t *testing.T) {
		// Ignoring the missing part would look to the caller like the filter
		// ran and simply matched everything.
		for _, args := range [][3]string{
			{"10.7721", "", "5"},
			{"", "106.6980", "5"},
			{"10.7721", "106.6980", ""},
		} {
			if _, err := parseNearQuery(args[0], args[1], args[2]); err == nil {
				t.Fatalf("expected an error for %v", args)
			}
		}
	})

	t.Run("rejects values that are not numbers", func(t *testing.T) {
		if _, err := parseNearQuery("here", "106.6980", "5"); err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("rejects a radius past the cap", func(t *testing.T) {
		radius := strconv.FormatFloat(shopsvc.MaxNearRadiusKm+1, 'f', -1, 64)
		if _, err := parseNearQuery("10.7721", "106.6980", radius); err == nil {
			t.Fatal("expected an error")
		}
	})
}
