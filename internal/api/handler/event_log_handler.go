package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/api/googleapi"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

	"vngrocery/internal/api/dto"
	auditsvc "vngrocery/internal/service/audit"
)

type EventLogUsecase interface {
	List(ctx context.Context, input auditsvc.ListInput) (auditsvc.ListResult, error)
	VerifyEvent(ctx context.Context, input auditsvc.VerifyEventInput) (auditsvc.EventVerificationResult, error)
	VerifyResource(ctx context.Context, input auditsvc.VerifyResourceInput) (auditsvc.VerifyResourceResult, error)
}

type EventLogHandler struct {
	events EventLogUsecase
}

func NewEventLogHandler(events EventLogUsecase) *EventLogHandler {
	return &EventLogHandler{events: events}
}

func (h *EventLogHandler) List(c *gin.Context) {
	page, pageSize, err := parsePagination(c.Query("page"), c.Query("pageSize"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	minSequence, err := parseOptionalPositiveInt(c.Query("minSequence"), "minSequence")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	maxSequence, err := parseOptionalPositiveInt(c.Query("maxSequence"), "maxSequence")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	createdAfter, err := parseOptionalTime(c.Query("createdAfter"), "createdAfter")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	createdBefore, err := parseOptionalTime(c.Query("createdBefore"), "createdBefore")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.events.List(c.Request.Context(), auditsvc.ListInput{
		ResourceType:  c.Query("resourceType"),
		ResourceID:    c.Query("resourceId"),
		ActorUserID:   c.Query("actorUserId"),
		Action:        c.Query("action"),
		Status:        c.Query("status"),
		MinSequence:   minSequence,
		MaxSequence:   maxSequence,
		CreatedAfter:  createdAfter,
		CreatedBefore: createdBefore,
		Page:          page,
		PageSize:      pageSize,
	})
	if err != nil {
		statusCode, message := mapEventLogListError(err)
		c.JSON(statusCode, gin.H{"error": message})
		return
	}

	response := make([]dto.EventLogResponse, 0, len(result.Items))
	for _, event := range result.Items {
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
			OccurredAt:      event.OccurredAt,
			PayloadJSON:     event.PayloadJSON,
			PublicKey:       event.PublicKey,
			KeyAlgorithm:    event.KeyAlgorithm,
			Signature:       event.Signature,
			ContentSHA256:   event.ContentSHA256,
			CreatedAt:       event.CreatedAt,
		})
	}

	totalPages := 0
	if result.Total > 0 {
		totalPages = (result.Total + result.PageSize - 1) / result.PageSize
	}

	c.JSON(http.StatusOK, dto.EventLogListResponse{
		Items: response,
		Pagination: dto.PaginationResponse{
			Page:       result.Page,
			PageSize:   result.PageSize,
			TotalItems: result.Total,
			TotalPages: totalPages,
		},
	})
}

func (h *EventLogHandler) VerifyEvent(c *gin.Context) {
	result, err := h.events.VerifyEvent(c.Request.Context(), auditsvc.VerifyEventInput{
		EventID: c.Param("eventId"),
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.EventVerificationResponse{
		EventID:              result.EventID,
		ResourceType:         result.ResourceType,
		ResourceID:           result.ResourceID,
		Sequence:             result.Sequence,
		PreviousEventID:      result.PreviousEventID,
		ContentHashValid:     result.ContentHashValid,
		SignatureValid:       result.SignatureValid,
		ChainLinkValid:       result.ChainLinkValid,
		PreviousEventPresent: result.PreviousEventPresent,
		Verified:             result.Verified,
	})
}

func (h *EventLogHandler) VerifyResource(c *gin.Context) {
	result, err := h.events.VerifyResource(c.Request.Context(), auditsvc.VerifyResourceInput{
		ResourceType: c.Query("resourceType"),
		ResourceID:   c.Query("resourceId"),
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	events := make([]dto.EventVerificationResponse, 0, len(result.Events))
	for _, event := range result.Events {
		events = append(events, dto.EventVerificationResponse{
			EventID:              event.EventID,
			ResourceType:         event.ResourceType,
			ResourceID:           event.ResourceID,
			Sequence:             event.Sequence,
			PreviousEventID:      event.PreviousEventID,
			ContentHashValid:     event.ContentHashValid,
			SignatureValid:       event.SignatureValid,
			ChainLinkValid:       event.ChainLinkValid,
			PreviousEventPresent: event.PreviousEventPresent,
			Verified:             event.Verified,
		})
	}

	c.JSON(http.StatusOK, dto.ResourceEventVerificationResponse{
		ResourceType: result.ResourceType,
		ResourceID:   result.ResourceID,
		EventCount:   result.EventCount,
		Verified:     result.Verified,
		Events:       events,
	})
}

func parseOptionalPositiveInt(raw, field string) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, errors.New(field + " must be a positive integer")
	}
	return parsed, nil
}

func parseOptionalTime(raw, field string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, nil
	}

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, errors.New(field + " must be an RFC3339 timestamp")
	}
	return parsed, nil
}

func mapEventLogListError(err error) (int, string) {
	if isQuotaExceededError(err) {
		return http.StatusTooManyRequests, "event log quota exceeded, narrow filters and retry later"
	}
	if isPermissionDeniedError(err) {
		return http.StatusForbidden, "you do not have permission to view these event logs"
	}
	if isTimeoutError(err) {
		return http.StatusGatewayTimeout, "event log query timed out, please retry with narrower filters"
	}
	return http.StatusBadRequest, err.Error()
}

func isQuotaExceededError(err error) bool {
	for current := err; current != nil; current = errors.Unwrap(current) {
		if st, ok := grpcstatus.FromError(current); ok && st.Code() == codes.ResourceExhausted {
			return true
		}
		var gErr *googleapi.Error
		if errors.As(current, &gErr) && gErr.Code == http.StatusTooManyRequests {
			return true
		}
	}
	return false
}

func isPermissionDeniedError(err error) bool {
	for current := err; current != nil; current = errors.Unwrap(current) {
		if st, ok := grpcstatus.FromError(current); ok && st.Code() == codes.PermissionDenied {
			return true
		}
		var gErr *googleapi.Error
		if errors.As(current, &gErr) && gErr.Code == http.StatusForbidden {
			return true
		}
	}
	return false
}

func isTimeoutError(err error) bool {
	for current := err; current != nil; current = errors.Unwrap(current) {
		if errors.Is(current, context.DeadlineExceeded) {
			return true
		}
		if st, ok := grpcstatus.FromError(current); ok && st.Code() == codes.DeadlineExceeded {
			return true
		}
	}
	return false
}
