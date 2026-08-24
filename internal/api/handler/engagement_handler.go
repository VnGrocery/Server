package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/dto"
	"vngrocery/internal/api/middleware"
	engagementsvc "vngrocery/internal/service/engagement"
)

type EngagementHandler struct{ service *engagementsvc.Service }

func NewEngagementHandler(service *engagementsvc.Service) *EngagementHandler {
	return &EngagementHandler{service: service}
}

// Get serves the totals for one shop or product.
func (h *EngagementHandler) Get(c *gin.Context) {
	principal, _ := middleware.GetPrincipal(c)
	state, err := h.service.Get(c.Request.Context(), principal.UserID, c.Query("targetType"), c.Query("targetId"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toEngagementResponse(state))
}

// Toggle adds the caller's mark or takes it back. The same request does both,
// because the button the reader pressed is the same button either way.
func (h *EngagementHandler) Toggle(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	var request dto.ToggleEngagementRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	state, err := h.service.Toggle(c.Request.Context(), principal.UserID, request.TargetType, request.TargetID, request.Kind)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toEngagementResponse(state))
}

func (h *EngagementHandler) writeError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, engagementsvc.ErrInvalidTarget), errors.Is(err, engagementsvc.ErrInvalidKind):
		status = http.StatusBadRequest
	case errors.Is(err, engagementsvc.ErrNotFound):
		status = http.StatusNotFound
	}
	c.JSON(status, gin.H{"error": err.Error()})
}

func toEngagementResponse(state engagementsvc.State) dto.EngagementResponse {
	mine := state.Mine
	if mine == nil {
		mine = []string{}
	}
	return dto.EngagementResponse{
		TargetType:       state.TargetType,
		TargetID:         state.TargetID,
		Follows:          state.Follows,
		Likes:            state.Likes,
		Loves:            state.Loves,
		Mine:             mine,
		DataHash:         state.DataHash,
		ChainTxHash:      state.ChainTxHash,
		ChainBlockNumber: state.ChainBlockNumber,
		ChainAnchorTime:  state.ChainAnchorTime,
		AnchorStatus:     state.AnchorStatus,
	}
}
