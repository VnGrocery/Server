package product

import (
	"strings"
	"unicode"

	"vngrocery/internal/textsearch"
)

// CatalogKey identifies "the same product" across shops.
//
// Nothing in the data says two shops sell the same thing: a product is a row
// belonging to one shop, with a free-text name and no SKU or barcode. So the
// identity has to be derived, and the only honest material is what the sellers
// typed.
//
// The key is the whole folded name plus the category. Matching the *whole* name
// is what makes this safe to average: "Gạo ST25 túi 5kg" and "Gạo ST25 túi 1kg"
// produce different keys, so two different pack sizes are never averaged into
// one misleading number. It under-matches rather than over-matches -- "Cải ngọt
// Đà Lạt" will not meet "Rau cải ngọt" -- and that is the right way round. A
// missed match shows one fewer comparison; a wrong match publishes a false
// average under the words "price transparency".
func CatalogKey(name, category string) string {
	folded := normalizeForKey(name)
	if folded == "" {
		return ""
	}
	return folded + "|" + normalizeForKey(category)
}

// normalizeForKey folds diacritics and case, drops punctuation, and collapses
// whitespace, so trivial typing differences do not split one product in two.
func normalizeForKey(text string) string {
	folded := textsearch.Fold(text)

	var cleaned strings.Builder
	cleaned.Grow(len(folded))
	lastWasSpace := true
	for _, symbol := range folded {
		switch {
		case unicode.IsLetter(symbol) || unicode.IsDigit(symbol):
			cleaned.WriteRune(symbol)
			lastWasSpace = false
		case !lastWasSpace:
			// Punctuation and runs of spaces both become one separator, so
			// "Cà chua (bi)" and "Cà chua bi" agree.
			cleaned.WriteRune(' ')
			lastWasSpace = true
		}
	}
	return strings.TrimSpace(cleaned.String())
}
