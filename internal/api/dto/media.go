package dto

type MediaImageUploadResponse struct {
	ImageHash   string `json:"imageHash"`
	ImageCID    string `json:"imageCid,omitempty"`
	GatewayURL  string `json:"gatewayUrl,omitempty"`
	ContentType string `json:"contentType"`
	SizeBytes   int64  `json:"sizeBytes"`
}
