package dto

type SellerScoreResponse struct {
	Score      float64 `json:"score"`
	Category   string  `json:"category"`
	Confidence float64 `json:"confidence"`
	ImageHash  string  `json:"imageHash"`
}
