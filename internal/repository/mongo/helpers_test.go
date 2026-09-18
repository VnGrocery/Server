package mongo

import (
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"vngrocery/internal/domain"
)

// Struct slices used to fail decoding with "unsupported slice type", which
// would have broken every product read as soon as one product carried specs.
func TestEncodeDecodeStructSlicesRoundTrip(t *testing.T) {
	product := domain.Product{
		ProductID: "p1",
		Name:      "Rau cải",
		Specs: []domain.SpecItem{
			{Key: "Xuất xứ", Value: "Đà Lạt, Lâm Đồng"},
			{Key: "Trọng lượng", Value: "500 g / bó"},
		},
		DescBlocks: []domain.DescBlock{
			{Type: domain.DescBlockHeading, Text: "Điểm nổi bật"},
			{Type: domain.DescBlockBullets, Items: []string{"Hái sáng nay", "Không thuốc"}},
			{Type: domain.DescBlockImage, CID: "bafy123", Caption: "Luống rau"},
		},
	}

	doc, err := encodeDocument(product)
	if err != nil {
		t.Fatalf("encodeDocument: %v", err)
	}

	// Keys must come from the firestore tags, not the driver's own naming.
	specs, ok := doc["specs"].([]any)
	if !ok {
		t.Fatalf("specs encoded as %T, want []any", doc["specs"])
	}
	first, err := asDocument(specs[0])
	if err != nil {
		t.Fatalf("asDocument: %v", err)
	}
	if first["key"] != "Xuất xứ" || first["value"] != "Đà Lạt, Lâm Đồng" {
		t.Fatalf("spec row encoded as %v", first)
	}

	var decoded domain.Product
	if err := decodeDocument(doc, &decoded); err != nil {
		t.Fatalf("decodeDocument: %v", err)
	}
	if !reflect.DeepEqual(decoded.Specs, product.Specs) {
		t.Errorf("specs round-trip: got %v, want %v", decoded.Specs, product.Specs)
	}
	if !reflect.DeepEqual(decoded.DescBlocks, product.DescBlocks) {
		t.Errorf("descBlocks round-trip: got %v, want %v", decoded.DescBlocks, product.DescBlocks)
	}
}

// What actually comes back from the driver is primitive.A of bson.M, not the
// []any that encodeDocument produced.
func TestDecodeStructSliceFromDriverShapes(t *testing.T) {
	doc := bson.M{
		"productId": "p1",
		"specs": primitive.A{
			bson.M{"key": "Bảo quản", "value": "Ngăn mát 2-5 °C"},
		},
		"descBlocks": primitive.A{
			bson.D{{Key: "type", Value: "paragraph"}, {Key: "text", Value: "Rau trồng tại vườn."}},
		},
	}

	var decoded domain.Product
	if err := decodeDocument(doc, &decoded); err != nil {
		t.Fatalf("decodeDocument: %v", err)
	}
	if len(decoded.Specs) != 1 || decoded.Specs[0].Key != "Bảo quản" {
		t.Fatalf("specs: got %v", decoded.Specs)
	}
	if len(decoded.DescBlocks) != 1 || decoded.DescBlocks[0].Text != "Rau trồng tại vườn." {
		t.Fatalf("descBlocks: got %v", decoded.DescBlocks)
	}
}

// Products written before this feature have no specs key at all.
func TestDecodeProductWithoutNewFields(t *testing.T) {
	var decoded domain.Product
	if err := decodeDocument(bson.M{"productId": "p1", "name": "Cà chua"}, &decoded); err != nil {
		t.Fatalf("decodeDocument: %v", err)
	}
	if decoded.Specs != nil || decoded.DescBlocks != nil {
		t.Fatalf("absent fields should stay nil, got %v / %v", decoded.Specs, decoded.DescBlocks)
	}
}
