package shop

import (
	"math"
	"testing"
)

// Bến Thành market, District 1.
const (
	benThanhLat = 10.7721
	benThanhLng = 106.6980
)

func TestDistanceKm(t *testing.T) {
	t.Run("is zero at the same point", func(t *testing.T) {
		if got := DistanceKm(benThanhLat, benThanhLng, benThanhLat, benThanhLng); got > 1e-9 {
			t.Fatalf("expected 0, got %v", got)
		}
	})

	t.Run("matches a known distance across the city", func(t *testing.T) {
		// Bến Thành to Thủ Đức: 0.0741 deg of latitude is 8.24 km and 0.0823 deg
		// of longitude at this latitude is 8.99 km, so the hypotenuse is 12.2 km.
		got := DistanceKm(benThanhLat, benThanhLng, 10.8462, 106.7803)
		if math.Abs(got-12.2) > 0.1 {
			t.Fatalf("expected ~12.2 km, got %v", got)
		}
	})

	t.Run("is symmetric", func(t *testing.T) {
		there := DistanceKm(benThanhLat, benThanhLng, 10.4114, 106.9548)
		back := DistanceKm(10.4114, 106.9548, benThanhLat, benThanhLng)
		if math.Abs(there-back) > 1e-9 {
			t.Fatalf("%v != %v", there, back)
		}
	})
}

func TestNearFilterValid(t *testing.T) {
	cases := []struct {
		name  string
		near  NearFilter
		valid bool
	}{
		{"a usable circle", NearFilter{benThanhLat, benThanhLng, 5}, true},
		// What a client with no location sends; it is a point in the Atlantic,
		// not a place anyone is standing.
		{"origin of the coordinate system", NearFilter{0, 0, 5}, false},
		{"no radius", NearFilter{benThanhLat, benThanhLng, 0}, false},
		{"negative radius", NearFilter{benThanhLat, benThanhLng, -1}, false},
		{"radius past the cap", NearFilter{benThanhLat, benThanhLng, MaxNearRadiusKm + 1}, false},
		{"radius at the cap", NearFilter{benThanhLat, benThanhLng, MaxNearRadiusKm}, true},
		{"latitude out of range", NearFilter{91, benThanhLng, 5}, false},
		{"longitude out of range", NearFilter{benThanhLat, 181, 5}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.near.Valid(); got != tc.valid {
				t.Fatalf("expected valid=%v, got %v", tc.valid, got)
			}
		})
	}
}
