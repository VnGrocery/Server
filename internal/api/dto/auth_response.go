package dto

type HealthResponse struct {
	Status string `json:"status"`
}

type MeResponse struct {
	UserID string `json:"userId"`
	Email  string `json:"email,omitempty"`
	Role   string `json:"role,omitempty"`
}

type DeleteResponse struct {
	UserID string `json:"userId"`
	Status string `json:"status"`
}

type LogoutResponse struct {
	Status string `json:"status"`
}

type StatusResponse struct {
	Status string `json:"status"`
}

type PasswordResetResponse struct {
	Status     string `json:"status"`
	ResetToken string `json:"resetToken,omitempty"`
}
