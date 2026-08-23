package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/dto"
	"vngrocery/internal/api/middleware"
	"vngrocery/internal/domain"
	commentsvc "vngrocery/internal/service/comment"
)

type CommentService interface {
	Create(ctx context.Context, input commentsvc.CreateInput) (domain.ProductComment, error)
	List(ctx context.Context, input commentsvc.ListInput) ([]commentsvc.View, commentsvc.Summary, error)
	ListForShop(ctx context.Context, input commentsvc.ShopQueueInput) ([]commentsvc.View, commentsvc.Summary, error)
	Moderate(ctx context.Context, input commentsvc.ModerateInput) (domain.ProductComment, error)
	Delete(ctx context.Context, input commentsvc.DeleteInput) (domain.ProductComment, error)
}

type CommentHandler struct {
	comments CommentService
}

func NewCommentHandler(comments CommentService) *CommentHandler {
	return &CommentHandler{comments: comments}
}

func (h *CommentHandler) List(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}
	views, summary, err := h.comments.List(c.Request.Context(), commentsvc.ListInput{
		ShopID:      c.Param("shopId"),
		ProductID:   c.Param("productId"),
		ActorUserID: principal.UserID,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	items := make([]dto.ProductCommentResponse, 0, len(views))
	for _, view := range views {
		items = append(items, toProductCommentResponse(view.Comment, view.AuthorName))
	}
	c.JSON(http.StatusOK, dto.ProductCommentListResponse{
		Items:         items,
		Moderation:    summary.ModerationOn,
		ApprovedCount: summary.ApprovedCount,
		PendingCount:  summary.PendingCount,
		RejectedCount: summary.RejectedCount,
		CanComment:    summary.CanComment,
	})
}

// ListForShop serves the owner's moderation queue.
func (h *CommentHandler) ListForShop(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}
	views, summary, err := h.comments.ListForShop(c.Request.Context(), commentsvc.ShopQueueInput{
		ShopID:      c.Param("shopId"),
		OwnerUserID: principal.UserID,
		Status:      c.Query("status"),
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	items := make([]dto.ProductCommentResponse, 0, len(views))
	for _, view := range views {
		items = append(items, toProductCommentResponse(view.Comment, view.AuthorName))
	}
	c.JSON(http.StatusOK, dto.ProductCommentListResponse{
		Items:         items,
		Moderation:    summary.ModerationOn,
		ApprovedCount: summary.ApprovedCount,
		PendingCount:  summary.PendingCount,
		RejectedCount: summary.RejectedCount,
	})
}

func (h *CommentHandler) Create(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}
	var request dto.CreateProductCommentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}
	comment, err := h.comments.Create(c.Request.Context(), commentsvc.CreateInput{
		ShopID:       c.Param("shopId"),
		ProductID:    c.Param("productId"),
		AuthorUserID: principal.UserID,
		Body:         request.Body,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	// The author knows their own name, so it is left empty here; the list
	// endpoint fills it in for everybody else.
	c.JSON(http.StatusCreated, toProductCommentResponse(comment, ""))
}

func (h *CommentHandler) Moderate(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}
	var request dto.ModerateProductCommentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}
	comment, err := h.comments.Moderate(c.Request.Context(), commentsvc.ModerateInput{
		ShopID:          c.Param("shopId"),
		CommentID:       c.Param("commentId"),
		OwnerUserID:     principal.UserID,
		ExpectedVersion: request.ExpectedVersion,
		Approve:         request.Approve,
		Reason:          request.Reason,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProductCommentResponse(comment, ""))
}

func (h *CommentHandler) Delete(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}
	var request dto.DeleteProductCommentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}
	comment, err := h.comments.Delete(c.Request.Context(), commentsvc.DeleteInput{
		ShopID:          c.Param("shopId"),
		CommentID:       c.Param("commentId"),
		ActorUserID:     principal.UserID,
		ExpectedVersion: request.ExpectedVersion,
		Reason:          request.Reason,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProductCommentResponse(comment, ""))
}

func (h *CommentHandler) writeError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, commentsvc.ErrInvalidComment):
		status = http.StatusBadRequest
	case errors.Is(err, commentsvc.ErrCheckRequired),
		errors.Is(err, commentsvc.ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(err, commentsvc.ErrNotFound):
		status = http.StatusNotFound
	}
	c.JSON(status, gin.H{"error": err.Error()})
}

func toProductCommentResponse(comment domain.ProductComment, authorName string) dto.ProductCommentResponse {
	return dto.ProductCommentResponse{
		CommentID:         comment.CommentID,
		ShopID:            comment.ShopID,
		ProductID:         comment.ProductID,
		AuthorUserID:      comment.AuthorUserID,
		AuthorName:        authorName,
		Body:              comment.Body,
		Status:            comment.Status,
		CheckID:           comment.CheckID,
		Verdict:           comment.Verdict,
		ModeratedByUserID: comment.ModeratedByUserID,
		ModerationReason:  comment.ModerationReason,
		ModeratedAt:       comment.ModeratedAt,
		Version:           comment.Version,
		CreatedAt:         comment.CreatedAt,
		UpdatedAt:         comment.UpdatedAt,
	}
}
