package product

import "testing"

func TestCatalogKeyMatchesTheSameProductAcrossShops(t *testing.T) {
	cases := []struct {
		name       string
		left       [2]string
		right      [2]string
		shouldPair bool
	}{
		{
			name:       "the same product typed the same way",
			left:       [2]string{"Cà chua bi hữu cơ", "vegetables"},
			right:      [2]string{"Cà chua bi hữu cơ", "vegetables"},
			shouldPair: true,
		},
		{
			name:       "one seller typed it without tone marks",
			left:       [2]string{"Cà chua bi hữu cơ", "vegetables"},
			right:      [2]string{"Ca chua bi huu co", "vegetables"},
			shouldPair: true,
		},
		{
			name:       "different case and spacing",
			left:       [2]string{"CÀ CHUA  BI", "vegetables"},
			right:      [2]string{"cà chua bi", "vegetables"},
			shouldPair: true,
		},
		{
			name:       "punctuation is not a difference",
			left:       [2]string{"Cà chua (bi)", "vegetables"},
			right:      [2]string{"Cà chua bi", "vegetables"},
			shouldPair: true,
		},
		{
			name: "different pack sizes are different products",
			// Averaging these would publish a false number under the words
			// "price transparency".
			left:       [2]string{"Gạo ST25 túi 5kg", "fresh_produce"},
			right:      [2]string{"Gạo ST25 túi 1kg", "fresh_produce"},
			shouldPair: false,
		},
		{
			name:       "the same words in a different category are not paired",
			left:       [2]string{"Trứng gà", "fresh_produce"},
			right:      [2]string{"Trứng gà", "meat"},
			shouldPair: false,
		},
		{
			name:       "a longer name is not the same product",
			left:       [2]string{"Cải ngọt", "vegetables"},
			right:      [2]string{"Cải ngọt Đà Lạt", "vegetables"},
			shouldPair: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			left := CatalogKey(tc.left[0], tc.left[1])
			right := CatalogKey(tc.right[0], tc.right[1])
			if (left == right) != tc.shouldPair {
				t.Fatalf("CatalogKey(%q)=%q vs CatalogKey(%q)=%q, wanted pair=%v",
					tc.left[0], left, tc.right[0], right, tc.shouldPair)
			}
		})
	}
}

func TestCatalogKeyRefusesAnEmptyName(t *testing.T) {
	// An unnamed product must not become a key that every other unnamed product
	// also matches.
	if got := CatalogKey("  ", "vegetables"); got != "" {
		t.Fatalf("expected an empty key, got %q", got)
	}
	if got := CatalogKey("!!!", "vegetables"); got != "" {
		t.Fatalf("expected punctuation alone to yield no key, got %q", got)
	}
}
