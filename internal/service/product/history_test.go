package product

import (
	"encoding/base64"
	"encoding/hex"
	"testing"
	"time"
)

func TestHexSHA(t *testing.T) {
	t.Run("converts the stored base64 digest to hex", func(t *testing.T) {
		// The log stores base64; hex is what people expect to read and paste.
		raw := []byte{0xde, 0xad, 0xbe, 0xef, 0x01, 0x02, 0x03, 0x04}
		stored := base64.StdEncoding.EncodeToString(raw)

		if got := hexSHA(stored); got != hex.EncodeToString(raw) {
			t.Fatalf("expected %s, got %s", hex.EncodeToString(raw), got)
		}
	})

	t.Run("leaves something that is not base64 alone", func(t *testing.T) {
		// Better to show an unexpected format than to swallow it.
		if got := hexSHA("not base64!!"); got != "not base64!!" {
			t.Fatalf("expected the input back, got %s", got)
		}
	})

	t.Run("an empty digest stays empty", func(t *testing.T) {
		if got := hexSHA("  "); got != "" {
			t.Fatalf("expected empty, got %q", got)
		}
	})
}

func TestShorten(t *testing.T) {
	full := "deadbeef01020304"

	if got := shorten(full); got != "deadbe" {
		t.Fatalf("expected six characters, got %s", got)
	}
	if got := shorten("abc"); got != "abc" {
		t.Fatalf("a short hash should be returned whole, got %s", got)
	}
}

func TestDiffSnapshots(t *testing.T) {
	before := &productSnapshot{
		Name:     "Cai kale",
		Price:    68000,
		Currency: "VND",
		Status:   "published",
		Tags:     []string{"huu co"},
	}

	t.Run("lists only the fields that actually changed", func(t *testing.T) {
		after := *before
		after.Price = 72000

		changes := diffSnapshots(before, &after)

		if len(changes) != 1 {
			t.Fatalf("expected one change, got %d: %+v", len(changes), changes)
		}
		if changes[0].Field != "price" || changes[0].Before != "68000" || changes[0].After != "72000" {
			t.Fatalf("unexpected change %+v", changes[0])
		}
	})

	t.Run("prints prices without a trailing decimal tail", func(t *testing.T) {
		// "68000 -> 72000", not "68000.000000 -> 72000.000000".
		if got := formatAmount(68000); got != "68000" {
			t.Fatalf("expected 68000, got %s", got)
		}
		if got := formatAmount(68000.5); got != "68000.5" {
			t.Fatalf("expected 68000.5, got %s", got)
		}
	})

	t.Run("an unchanged product yields no changes", func(t *testing.T) {
		after := *before
		if changes := diffSnapshots(before, &after); len(changes) != 0 {
			t.Fatalf("expected no changes, got %+v", changes)
		}
	})

	t.Run("the first entry has nothing to compare against", func(t *testing.T) {
		if changes := diffSnapshots(nil, before); changes != nil {
			t.Fatalf("expected nil, got %+v", changes)
		}
	})
}

func TestPriceWindow(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	day := func(daysAgo int) time.Time { return now.AddDate(0, 0, -daysAgo) }

	t.Run("keeps the points inside the window", func(t *testing.T) {
		points := []PricePoint{
			{At: day(20), Price: 100},
			{At: day(5), Price: 120},
		}

		got := priceWindow(points, now, 30)

		if len(got) != 2 {
			t.Fatalf("expected 2 points, got %d", len(got))
		}
	})

	t.Run("carries the price in effect when the window opened", func(t *testing.T) {
		// Otherwise a product last edited months ago charts as though it had no
		// price until its next change.
		points := []PricePoint{
			{At: day(200), Price: 100},
			{At: day(5), Price: 120},
		}

		got := priceWindow(points, now, 30)

		if len(got) != 2 {
			t.Fatalf("expected a carried point plus the real one, got %d", len(got))
		}
		if got[0].Price != 100 {
			t.Fatalf("expected the older price carried forward, got %v", got[0].Price)
		}
		if !got[0].At.Equal(day(30)) {
			t.Fatalf("expected the carried point at the window start, got %v", got[0].At)
		}
	})

	t.Run("a product changed only long ago still has a line to draw", func(t *testing.T) {
		points := []PricePoint{{At: day(400), Price: 90}}

		got := priceWindow(points, now, 30)

		if len(got) != 1 || got[0].Price != 90 {
			t.Fatalf("unexpected series %+v", got)
		}
	})

	t.Run("orders the series oldest first whatever order it arrives in", func(t *testing.T) {
		points := []PricePoint{
			{At: day(2), Price: 130},
			{At: day(9), Price: 120},
		}

		got := priceWindow(points, now, 30)

		if !got[0].At.Before(got[1].At) {
			t.Fatalf("expected oldest first, got %+v", got)
		}
	})

	t.Run("no points means no series rather than a zero price", func(t *testing.T) {
		if got := priceWindow(nil, now, 30); got != nil {
			t.Fatalf("expected nil, got %+v", got)
		}
	})
}
