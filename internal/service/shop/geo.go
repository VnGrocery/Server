package shop

import "math"

// NearFilter narrows a listing to shops within RadiusKm of a point.
//
// Without it the app had to download every shop in the country and work out
// what was close by itself.
type NearFilter struct {
	Latitude  float64
	Longitude float64
	RadiusKm  float64
}

// MaxNearRadiusKm caps how far "near me" can reach. Past this the answer is no
// longer a list of places anyone would travel to for fresh produce, and the
// filter stops doing any useful narrowing.
const MaxNearRadiusKm float64 = 50

// Valid reports whether the filter describes a usable circle.
//
// (0, 0) is a point in the Atlantic and is what a client sends when it has no
// location at all, so it is treated as "no filter" rather than as a place.
func (f NearFilter) Valid() bool {
	if f.RadiusKm <= 0 || f.RadiusKm > MaxNearRadiusKm {
		return false
	}
	if f.Latitude < -90 || f.Latitude > 90 {
		return false
	}
	if f.Longitude < -180 || f.Longitude > 180 {
		return false
	}
	return f.Latitude != 0 || f.Longitude != 0
}

// DistanceKm is the great-circle distance between two points.
//
// Straight-line rather than travel distance: this only has to decide which
// shops are close, and a routing service would be a network call per shop.
func DistanceKm(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0088

	dLat := radians(lat2 - lat1)
	dLon := radians(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(radians(lat1))*math.Cos(radians(lat2))

	return earthRadiusKm * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func radians(degrees float64) float64 { return degrees * math.Pi / 180 }
