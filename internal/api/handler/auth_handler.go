package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/dto"
	"vngrocery/internal/api/middleware"
	authservice "vngrocery/internal/service/auth"
)

type AuthHandler struct {
	accounts AccountUsecase
}

type AccountUsecase interface {
	Register(ctx context.Context, email, password, displayName string) (string, authservice.Principal, error)
	Login(ctx context.Context, email, password string) (string, authservice.Principal, error)
	GoogleLogin(ctx context.Context, googleIDToken string) (string, authservice.Principal, error)
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

	c.JSON(http.StatusOK, dto.MeResponse{
		UserID: principal.UserID,
		Email:  principal.Email,
	})
}

func (h *AuthHandler) Register(c *gin.Context) {
	var request dto.RegisterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	token, principal, err := h.accounts.Register(c.Request.Context(), request.Email, request.Password, request.DisplayName)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, authservice.ErrInvalidCredentials) || errors.Is(err, authservice.ErrEmailTaken) {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, dto.AuthTokenResponse{
		AccessToken: token,
		UserID:      principal.UserID,
		Email:       principal.Email,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var request dto.LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	token, principal, err := h.accounts.Login(c.Request.Context(), request.Email, request.Password)
	if err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, authservice.ErrInvalidCredentials) {
			status = http.StatusUnauthorized
		} else {
			status = http.StatusInternalServerError
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.AuthTokenResponse{
		AccessToken: token,
		UserID:      principal.UserID,
		Email:       principal.Email,
	})
}

func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	var request dto.GoogleLoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	token, principal, err := h.accounts.GoogleLogin(c.Request.Context(), request.IDToken)
	if err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, authservice.ErrInvalidCredentials) {
			status = http.StatusUnauthorized
		} else {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.AuthTokenResponse{
		AccessToken: token,
		UserID:      principal.UserID,
		Email:       principal.Email,
	})
}
