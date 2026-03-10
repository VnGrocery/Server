package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/dto"
	"vngrocery/internal/domain"
	auditsvc "vngrocery/internal/service/audit"
)

type EventLogUsecase interface {
	List(ctx context.Context, input auditsvc.ListInput) ([]domain.EventLog, error)
}

type EventLogHandler struct {
	events EventLogUsecase
}

func NewEventLogHandler(events EventLogUsecase) *EventLogHandler {
	return &EventLogHandler{events: events}
}

func (h *EventLogHandler) List(c *gin.Context) {
	events, err := h.events.List(c.Request.Context(), auditsvc.ListInput{
		ResourceType: c.Query("resourceType"),
		ResourceID:   c.Query("resourceId"),
		ActorUserID:  c.Query("actorUserId"),
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response := make([]dto.EventLogResponse, 0, len(events))
	for _, event := range events {
		response = append(response, dto.EventLogResponse{
			EventID:         event.EventID,
			ActorUserID:     event.ActorUserID,
			ResourceType:    event.ResourceType,
			ResourceID:      event.ResourceID,
			ResourceVersion: event.ResourceVersion,
			Action:          event.Action,
			Status:          event.Status,
			Sequence:        event.Sequence,
			PreviousEventID: event.PreviousEventID,
			PayloadJSON:     event.PayloadJSON,
			PublicKey:       event.PublicKey,
			KeyAlgorithm:    event.KeyAlgorithm,
			Signature:       event.Signature,
			ContentSHA256:   event.ContentSHA256,
			CreatedAt:       event.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, response)
}
