package product

import (
	"reflect"
	"testing"

	"vngrocery/internal/domain"
)

func TestNormalizeSpecs(t *testing.T) {
	got := normalizeSpecs([]domain.SpecItem{
		{Key: "  Xuất xứ ", Value: " Đà Lạt "},
		{Key: "", Value: ""},      // blank row the form shows by default
		{Key: "  ", Value: "   "}, // whitespace only
		{Key: "", Value: "500 g"}, // value without a label is still kept
		{Key: "Bảo quản", Value: ""},
	})

	want := []domain.SpecItem{
		{Key: "Xuất xứ", Value: "Đà Lạt"},
		{Key: "", Value: "500 g"},
		{Key: "Bảo quản", Value: ""},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if normalizeSpecs(nil) != nil {
		t.Error("nil input should stay nil")
	}
	if normalizeSpecs([]domain.SpecItem{{}}) != nil {
		t.Error("all-blank input should collapse to nil, not an empty slice")
	}
}

func TestNormalizeDescBlocksKeepsOnlyRenderableBlocks(t *testing.T) {
	got := normalizeDescBlocks([]domain.DescBlock{
		{Type: "HEADING", Text: " Điểm nổi bật "},
		{Type: "paragraph", Text: ""},                            // empty, dropped
		{Type: "bullets", Items: []string{"Hái sáng nay", "  "}}, // blank item dropped
		{Type: "bullets", Items: []string{"  "}},                 // nothing left, dropped
		{Type: "image", CID: " bafy123 ", Caption: " Luống rau "},
		{Type: "image", CID: ""},      // no image, dropped
		{Type: "marquee", Text: "hi"}, // unknown type would render as a blank gap
	})

	want := []domain.DescBlock{
		{Type: "heading", Text: "Điểm nổi bật"},
		{Type: "bullets", Items: []string{"Hái sáng nay"}},
		{Type: "image", CID: "bafy123", Caption: "Luống rau"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

// Fields that do not belong to a block's type are cleared, so they cannot ride
// along unnoticed into the signed payload.
func TestNormalizeDescBlocksClearsFieldsOffType(t *testing.T) {
	got := normalizeDescBlocks([]domain.DescBlock{
		{Type: "paragraph", Text: "Rau sạch", Items: []string{"x"}, CID: "bafy", Caption: "c"},
	})

	want := []domain.DescBlock{{Type: "paragraph", Text: "Rau sạch"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}
