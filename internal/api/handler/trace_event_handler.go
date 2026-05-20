package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/dto"
	"vngrocery/internal/api/middleware"
	"vngrocery/internal/domain"
	traceabilitysvc "vngrocery/internal/service/traceability"
)

type TraceEventService interface {
	CreateTraceEvent(ctx context.Context, input traceabilitysvc.CreateTraceEventInput) (domain.TraceEvent, error)
	ListTraceEvents(ctx context.Context, input traceabilitysvc.ListTraceEventsInput) ([]domain.TraceEvent, error)
}

type TraceEventHandler struct {
	events TraceEventService
}

func NewTraceEventHandler(events TraceEventService) *TraceEventHandler {
	return &TraceEventHandler{events: events}
}

func (h *TraceEventHandler) Create(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}
	var request dto.CreateTraceEventRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}
	event, err := h.events.CreateTraceEvent(c.Request.Context(), traceabilitysvc.CreateTraceEventInput{
		ShopID:       c.Param("shopId"),
		ProductID:    c.Param("productId"),
		BatchID:      c.Param("batchId"),
		ActorUserID:  principal.UserID,
		Type:         request.Type,
		Title:        request.Title,
		Description:  request.Description,
		LocationName: request.LocationName,
		Latitude:     request.Latitude,
		Longitude:    request.Longitude,
		Temperature:  request.Temperature,
		Humidity:     request.Humidity,
		ImageCID:     request.ImageCID,
		ImageHash:    request.ImageHash,
		DataHash:     request.DataHash,
		OccurredAt:   request.OccurredAt,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toTraceEventResponse(event))
}

func (h *TraceEventHandler) List(c *gin.Context) {
	events, err := h.events.ListTraceEvents(c.Request.Context(), traceabilitysvc.ListTraceEventsInput{
		ShopID:    c.Param("shopId"),
		ProductID: c.Param("productId"),
		BatchID:   c.Param("batchId"),
		Type:      c.Query("type"),
		Public:    true,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	items := make([]dto.TraceEventResponse, 0, len(events))
	for _, event := range events {
		items = append(items, toTraceEventResponse(event))
	}
	c.JSON(http.StatusOK, dto.TraceEventListResponse{Items: items})
}

func (h *TraceEventHandler) writeError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, traceabilitysvc.ErrInvalidTraceEvent):
		status = http.StatusBadRequest
	case errors.Is(err, traceabilitysvc.ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(err, traceabilitysvc.ErrNotFound):
		status = http.StatusNotFound
	}
	c.JSON(status, gin.H{"error": err.Error()})
}

func toTraceEventResponse(event domain.TraceEvent) dto.TraceEventResponse {
	return dto.TraceEventResponse{
		EventID:      event.EventID,
		BatchID:      event.BatchID,
		ProductID:    event.ProductID,
		ShopID:       event.ShopID,
		ActorUserID:  event.ActorUserID,
		Type:         event.Type,
		Title:        event.Title,
		Description:  event.Description,
		LocationName: event.LocationName,
		Latitude:     event.Latitude,
		Longitude:    event.Longitude,
		Temperature:  event.Temperature,
		Humidity:     event.Humidity,
		ImageCID:     event.ImageCID,
		ImageHash:    event.ImageHash,
		DataHash:     event.DataHash,
		Status:       event.Status,
		OccurredAt:   event.OccurredAt,
		CreatedAt:    event.CreatedAt,
	}
}
