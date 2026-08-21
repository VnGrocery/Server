// Package textsearch normalises text so a search matches what people type.
package textsearch

import "strings"

// Fold lowercases text and strips Vietnamese diacritics.
//
// Typing without tone marks is how Vietnamese is normally entered on a phone,
// and a plain substring match meant "Huu Co" found nothing while "Hữu Cơ" sat
// in the list. Both the stored text and the query go through this, so either
// spelling finds the other.
//
// Written as an explicit table rather than with Unicode normalisation so the
// mobile app can carry the identical rules -- Dart has no normaliser, and a
// search that folds differently on each side is worse than one that does not
// fold at all.
func Fold(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	var folded strings.Builder
	folded.Grow(len(text))
	for _, symbol := range strings.ToLower(text) {
		// Text already decomposed carries its tone as a separate combining
		// mark; the base letter is kept and the mark dropped.
		if symbol >= 0x0300 && symbol <= 0x036F {
			continue
		}
		if plain, accented := vietnamese[symbol]; accented {
			folded.WriteRune(plain)
			continue
		}
		folded.WriteRune(symbol)
	}
	return folded.String()
}

// Contains reports whether haystack contains needle, both folded.
func Contains(haystack, needle string) bool {
	return strings.Contains(Fold(haystack), Fold(needle))
}

// vietnamese maps every accented lowercase letter to its plain form. Uppercase
// is handled by lowercasing first.
var vietnamese = buildTable(map[rune]string{
	'a': "àáảãạăằắẳẵặâầấẩẫậ",
	'e': "èéẻẽẹêềếểễệ",
	'i': "ìíỉĩị",
	'o': "òóỏõọôồốổỗộơờớởỡợ",
	'u': "ùúủũụưừứửữự",
	'y': "ỳýỷỹỵ",
	// d with stroke is a letter in its own right, not an accented d, so no
	// amount of decomposition would ever reduce it.
	'd': "đ",
})

func buildTable(groups map[rune]string) map[rune]rune {
	table := make(map[rune]rune, 90)
	for plain, accented := range groups {
		for _, symbol := range accented {
			table[symbol] = plain
		}
	}
	return table
}
