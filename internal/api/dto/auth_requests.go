package dto

type RegisterRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type GoogleLoginRequest struct {
	IDToken string `json:"idToken"`
}

type AuthTokenResponse struct {
	AccessToken string `json:"accessToken"`
	UserID      string `json:"userId"`
	Email       string `json:"email,omitempty"`
	PublicKey   string `json:"publicKey,omitempty"`
}
