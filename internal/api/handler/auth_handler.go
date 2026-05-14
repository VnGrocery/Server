package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/dto"
	"vngrocery/internal/api/middleware"
	"vngrocery/internal/domain"
	authservice "vngrocery/internal/service/auth"
)

type AuthHandler struct {
	accounts AccountUsecase
}

type AccountUsecase interface {
	Register(ctx context.Context, email, password, displayName, firstName, lastName string) (authservice.AuthResult, error)
	Login(ctx context.Context, email, password string) (authservice.AuthResult, error)
	GoogleLogin(ctx context.Context, googleIDToken string) (authservice.AuthResult, error)
	Refresh(ctx context.Context, refreshToken string) (authservice.AuthResult, error)
	Logout(ctx context.Context, refreshToken string) error
	ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error
	RequestPasswordReset(ctx context.Context, email string) (authservice.PasswordResetResult, error)
	ResetPassword(ctx context.Context, resetToken, newPassword string) error
	Delete(ctx context.Context, userID string, expectedVersion int) (authservice.DeleteResult, error)
	Me(ctx context.Context, userID string) (domain.User, error)
}

func NewAuthHandler(accounts AccountUsecase) *AuthHandler {
	return &AuthHandler{accounts: accounts}
}

func (h *AuthHandler) Me(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "authenticated principal was not found in request context",
		})
		return
	}

	user, err := h.accounts.Me(c.Request.Context(), principal.UserID)
	if err != nil {
		c.JSON(authStatus(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.MeResponse{
		UserID:      principal.UserID,
		Email:       principal.Email,
		Role:        principal.Role,
		DisplayName: user.DisplayName,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
	})
}

func (h *AuthHandler) Register(c *gin.Context) {
	var request dto.RegisterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	result, err := h.accounts.Register(c.Request.Context(), request.Email, request.Password, request.DisplayName, request.FirstName, request.LastName)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, authservice.ErrInvalidCredentials) || errors.Is(err, authservice.ErrEmailTaken) {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, toAuthTokenResponse(result))
}

func (h *AuthHandler) Login(c *gin.Context) {
	var request dto.LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	result, err := h.accounts.Login(c.Request.Context(), request.Email, request.Password)
	if err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, authservice.ErrInvalidCredentials) {
			status = http.StatusUnauthorized
		} else if errors.Is(err, authservice.ErrAccountDeleted) {
			status = http.StatusForbidden
		} else {
			status = http.StatusInternalServerError
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, toAuthTokenResponse(result))
}

func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	var request dto.GoogleLoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	result, err := h.accounts.GoogleLogin(c.Request.Context(), request.IDToken)
	if err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, authservice.ErrInvalidCredentials) {
			status = http.StatusUnauthorized
		} else if errors.Is(err, authservice.ErrAccountDeleted) {
			status = http.StatusForbidden
		} else {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, toAuthTokenResponse(result))
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var request dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	result, err := h.accounts.Refresh(c.Request.Context(), request.RefreshToken)
	if err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, authservice.ErrAccountDeleted) {
			status = http.StatusForbidden
		} else if !errors.Is(err, authservice.ErrInvalidRefreshToken) {
			status = http.StatusInternalServerError
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, toAuthTokenResponse(result))
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var request dto.LogoutRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	if err := h.accounts.Logout(c.Request.Context(), request.RefreshToken); err != nil {
		status := http.StatusUnauthorized
		if !errors.Is(err, authservice.ErrInvalidRefreshToken) {
			status = http.StatusInternalServerError
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.LogoutResponse{Status: "logged_out"})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}

	var request dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	if err := h.accounts.ChangePassword(c.Request.Context(), principal.UserID, request.CurrentPassword, request.NewPassword); err != nil {
		c.JSON(authStatus(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.StatusResponse{Status: "password_changed"})
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var request dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	result, err := h.accounts.RequestPasswordReset(c.Request.Context(), request.Email)
	if err != nil {
		c.JSON(authStatus(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.PasswordResetResponse{Status: "reset_requested", ResetToken: result.ResetToken})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var request dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	if err := h.accounts.ResetPassword(c.Request.Context(), request.ResetToken, request.NewPassword); err != nil {
		c.JSON(authStatus(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.StatusResponse{Status: "password_reset"})
}

func (h *AuthHandler) DeleteMe(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}

	expectedVersion, parseErr := parsePositiveIntQuery(c.Query("expectedVersion"), "expectedVersion")
	if parseErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": parseErr.Error()})
		return
	}

	result, err := h.accounts.Delete(c.Request.Context(), principal.UserID, expectedVersion)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, authservice.ErrInvalidCredentials) {
			status = http.StatusBadRequest
		} else if errors.Is(err, authservice.ErrVersionConflict) {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.DeleteResponse{
		UserID: result.UserID,
		Status: result.Status,
	})
}

func toAuthTokenResponse(result authservice.AuthResult) dto.AuthTokenResponse {
	return dto.AuthTokenResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		UserID:       result.Principal.UserID,
		Email:        result.Principal.Email,
		Role:         result.Principal.Role,
		DisplayName:  result.DisplayName,
		FirstName:    result.FirstName,
		LastName:     result.LastName,
		PublicKey:    result.PublicKey,
	}
}

func authStatus(err error) int {
	switch {
	case errors.Is(err, authservice.ErrInvalidCredentials), errors.Is(err, authservice.ErrInvalidResetToken):
		return http.StatusBadRequest
	case errors.Is(err, authservice.ErrAccountDeleted):
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}
