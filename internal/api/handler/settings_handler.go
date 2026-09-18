package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/middleware"
	"vngrocery/internal/domain"
	settingssvc "vngrocery/internal/service/settings"
)

type SettingsService interface {
	Get(ctx context.Context) (domain.RuntimeSettings, error)
	Update(ctx context.Context, input settingssvc.UpdateInput) (domain.RuntimeSettings, error)
}

type SettingsHandler struct {
	settings SettingsService
}

func NewSettingsHandler(settings SettingsService) *SettingsHandler {
	return &SettingsHandler{settings: settings}
}

type settingsResponse struct {
	FreshnessReportPerHour int    `json:"freshnessReportPerHour"`
	BuyerCheckPerHour      int    `json:"buyerCheckPerHour"`
	RateLimitWindowMinutes int    `json:"rateLimitWindowMinutes"`
	Version                int    `json:"version"`
	UpdatedByUserID        string `json:"updatedByUserId,omitempty"`
	// A pointer because omitempty does not drop a zero time.Time, and a
	// never-edited settings document would report being saved in year 0001.
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

type updateSettingsRequest struct {
	FreshnessReportPerHour int `json:"freshnessReportPerHour"`
	BuyerCheckPerHour      int `json:"buyerCheckPerHour"`
	RateLimitWindowMinutes int `json:"rateLimitWindowMinutes"`
}

func (h *SettingsHandler) Get(c *gin.Context) {
	settings, err := h.settings.Get(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toSettingsResponse(settings))
}

func (h *SettingsHandler) Update(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}

	var request updateSettingsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	settings, err := h.settings.Update(c.Request.Context(), settingssvc.UpdateInput{
		ActorUserID:            principal.UserID,
		FreshnessReportPerHour: request.FreshnessReportPerHour,
		BuyerCheckPerHour:      request.BuyerCheckPerHour,
		RateLimitWindowMinutes: request.RateLimitWindowMinutes,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, settingssvc.ErrInvalidSettings) {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toSettingsResponse(settings))
}

func toSettingsResponse(settings domain.RuntimeSettings) settingsResponse {
	response := settingsResponse{
		FreshnessReportPerHour: settings.FreshnessReportPerHour,
		BuyerCheckPerHour:      settings.BuyerCheckPerHour,
		RateLimitWindowMinutes: settings.RateLimitWindowMinutes,
		Version:                settings.Version,
		UpdatedByUserID:        settings.UpdatedByUserID,
	}
	if !settings.UpdatedAt.IsZero() {
		updatedAt := settings.UpdatedAt.UTC()
		response.UpdatedAt = &updatedAt
	}
	return response
}
