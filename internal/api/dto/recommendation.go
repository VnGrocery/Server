package dto

// RecommendedProductResponse is a product suggested to one reader, with the
// reasons it was suggested. A suggestion nobody can explain is indistinguishable
// from a random one, so the reasons travel with it.
type RecommendedProductResponse struct {
	ProductID  string   `json:"productId"`
	ShopID     string   `json:"shopId"`
	ShopName   string   `json:"shopName"`
	Name       string   `json:"name"`
	Category   string   `json:"category"`
	Price      float64  `json:"price"`
	Currency   string   `json:"currency"`
	ImageURLs  []string `json:"imageUrls"`
	Score      float64  `json:"score"`
	Reasons    []string `json:"reasons"`
	DistanceKm *float64 `json:"distanceKm,omitempty"`
}

// RecommendedShopResponse is a shop suggested to one reader.
type RecommendedShopResponse struct {
	ShopID     string   `json:"shopId"`
	Name       string   `json:"name"`
	Address    string   `json:"address"`
	TrustScore float64  `json:"trustScore"`
	TrustGrade string   `json:"trustGrade"`
	Rating     float64  `json:"rating"`
	Score      float64  `json:"score"`
	Reasons    []string `json:"reasons"`
	DistanceKm *float64 `json:"distanceKm,omitempty"`
}

// RecommendationsResponse is what one reader is shown, and what it rests on.
type RecommendationsResponse struct {
	Products []RecommendedProductResponse `json:"products"`
	Shops    []RecommendedShopResponse    `json:"shops"`

	// Personalised is false when the reader has done nothing the app can learn
	// from. The list is then ordered by trust and distance, and the client must
	// not present it as personal.
	Personalised bool `json:"personalised"`

	// SignalCount and Categories say what the suggestions were derived from.
	SignalCount int      `json:"signalCount"`
	Categories  []string `json:"categories"`
}
