package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/dto"
	"vngrocery/internal/api/middleware"
	"vngrocery/internal/domain"
	productsvc "vngrocery/internal/service/product"
)

type ProductService interface {
	Create(ctx context.Context, input productsvc.CreateInput) (domain.Product, error)
	Update(ctx context.Context, input productsvc.UpdateInput) (domain.Product, error)
	Delete(ctx context.Context, input productsvc.DeleteInput) (domain.Product, error)
	GetByID(ctx context.Context, shopID, productID string) (domain.Product, error)
	List(ctx context.Context, input productsvc.ListInput) ([]domain.Product, error)
}

type ProductHandler struct {
	products ProductService
}

func NewProductHandler(products ProductService) *ProductHandler {
	return &ProductHandler{products: products}
}

func (h *ProductHandler) Create(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}

	var request dto.UpsertProductRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	product, err := h.products.Create(c.Request.Context(), productsvc.CreateInput{
		ShopID:      c.Param("shopId"),
		OwnerUserID: principal.UserID,
		Name:        request.Name,
		Description: request.Description,
		Price:       request.Price,
		Currency:    request.Currency,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toProductResponse(product))
}

func (h *ProductHandler) Update(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}

	var request dto.UpsertProductRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	product, err := h.products.Update(c.Request.Context(), productsvc.UpdateInput{
		ProductID:       c.Param("productId"),
		ShopID:          c.Param("shopId"),
		OwnerUserID:     principal.UserID,
		ExpectedVersion: request.ExpectedVersion,
		Name:            request.Name,
		Description:     request.Description,
		Price:           request.Price,
		Currency:        request.Currency,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProductResponse(product))
}

func (h *ProductHandler) Delete(c *gin.Context) {
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

	product, err := h.products.Delete(c.Request.Context(), productsvc.DeleteInput{
		ProductID:       c.Param("productId"),
		ShopID:          c.Param("shopId"),
		OwnerUserID:     principal.UserID,
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProductResponse(product))
}

func (h *ProductHandler) GetByID(c *gin.Context) {
	product, err := h.products.GetByID(c.Request.Context(), c.Param("shopId"), c.Param("productId"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProductResponse(product))
}

func (h *ProductHandler) List(c *gin.Context) {
	products, err := h.products.List(c.Request.Context(), productsvc.ListInput{
		ShopID: c.Param("shopId"),
	})
	if err != nil {
		h.writeError(c, err)
		return
	}

	items := make([]dto.ProductResponse, 0, len(products))
	for _, product := range products {
		items = append(items, toProductResponse(product))
	}
	c.JSON(http.StatusOK, dto.ProductListResponse{Items: items})
}

func (h *ProductHandler) writeError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, productsvc.ErrInvalidProduct):
		status = http.StatusBadRequest
	case errors.Is(err, productsvc.ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(err, productsvc.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, productsvc.ErrVersionConflict):
		status = http.StatusConflict
	}
	c.JSON(status, gin.H{"error": err.Error()})
}

func toProductResponse(product domain.Product) dto.ProductResponse {
	return dto.ProductResponse{
		ProductID:   product.ProductID,
		ShopID:      product.ShopID,
		OwnerUserID: product.OwnerUserID,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Currency:    product.Currency,
		Status:      product.Status,
		Version:     product.Version,
		CreatedAt:   product.CreatedAt,
		UpdatedAt:   product.UpdatedAt,
	}
}
