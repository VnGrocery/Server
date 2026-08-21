package textsearch

import "testing"

func TestFold(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"strips tone marks", "Hữu Cơ", "huu co"},
		{"maps d with stroke, which never decomposes", "Đà Lạt", "da lat"},
		{"lowercase d with stroke too", "đậu", "dau"},
		{"every accented vowel base", "ăâêôơư", "aaeoou"},
		{"a full shop name", "Rau Sạch Cô Ba", "rau sach co ba"},
		{"already plain text is untouched", "rau sach", "rau sach"},
		{"trims surrounding space", "  Cần Giờ  ", "can gio"},
		{"empty stays empty", "   ", ""},
		{"leaves other scripts and digits alone", "VietGAP 2024", "vietgap 2024"},
		// Text can arrive with the tone as a separate combining mark rather
		// than precomposed; both spellings have to fold the same way.
		{"decomposed input folds like precomposed", "Hụ̄u", "huu"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Fold(tc.in); got != tc.want {
				t.Fatalf("Fold(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestFoldCoversEveryVietnameseLetter(t *testing.T) {
	// A missed letter is a silent hole: that word simply stops being findable.
	const everyAccented = "àáảãạăằắẳẵặâầấẩẫậèéẻẽẹêềếểễệìíỉĩị" +
		"òóỏõọôồốổỗộơờớởỡợùúủũụưừứửữựỳýỷỹỵđ"

	for _, symbol := range everyAccented {
		folded := Fold(string(symbol))
		if folded == "" {
			t.Fatalf("%q folded away entirely", symbol)
		}
		if []rune(folded)[0] > 127 {
			t.Fatalf("%q folded to %q, which is still accented", symbol, folded)
		}
	}
}

func TestContains(t *testing.T) {
	const shop = "Rau Hữu Cơ Quận 3"

	t.Run("typed without tone marks", func(t *testing.T) {
		// The bug: this is how the name gets typed on a phone, and it found
		// nothing.
		if !Contains(shop, "huu co") {
			t.Fatal("expected a match for the unaccented spelling")
		}
	})

	t.Run("typed with tone marks", func(t *testing.T) {
		if !Contains(shop, "Hữu Cơ") {
			t.Fatal("expected a match for the accented spelling")
		}
	})

	t.Run("mixed case and a partial word", func(t *testing.T) {
		if !Contains(shop, "QUAN 3") {
			t.Fatal("expected a case-insensitive match")
		}
	})

	t.Run("a genuine miss is still a miss", func(t *testing.T) {
		if Contains(shop, "hai san") {
			t.Fatal("folding must not make everything match everything")
		}
	})

	t.Run("an empty query matches anything", func(t *testing.T) {
		if !Contains(shop, "  ") {
			t.Fatal("an empty search should not exclude results")
		}
	})
}
