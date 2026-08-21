package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/dto"
	"vngrocery/internal/api/middleware"
	recommendsvc "vngrocery/internal/service/recommend"
)

// RecommendationService suggests shops and products for one reader.
type RecommendationService interface {
	Suggest(ctx context.Context, input recommendsvc.Input) (recommendsvc.Result, error)
}

type RecommendationHandler struct {
	recommendations RecommendationService
}

func NewRecommendationHandler(recommendations RecommendationService) *RecommendationHandler {
	return &RecommendationHandler{recommendations: recommendations}
}

// List returns suggestions for the signed-in reader.
func (h *RecommendationHandler) List(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}

	near, err := parseNearQuery(c.Query("lat"), c.Query("lng"), c.Query("radiusKm"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	limit, err := parseOptionalPositiveIntQuery(c.Query("limit"), "limit")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.recommendations.Suggest(c.Request.Context(), recommendsvc.Input{
		UserID: principal.UserID,
		Near:   near,
		Limit:  limit,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	products := make([]dto.RecommendedProductResponse, 0, len(result.Products))
	for _, item := range result.Products {
		products = append(products, dto.RecommendedProductResponse{
			ProductID:  item.Product.ProductID,
			ShopID:     item.Product.ShopID,
			ShopName:   item.Shop.Name,
			Name:       item.Product.Name,
			Category:   item.Product.Category,
			Price:      item.Product.Price,
			Currency:   item.Product.Currency,
			ImageURLs:  item.Product.ImageURLs,
			Score:      item.Score,
			Reasons:    item.Reasons,
			DistanceKm: item.DistanceKm,
		})
	}

	shops := make([]dto.RecommendedShopResponse, 0, len(result.Shops))
	for _, item := range result.Shops {
		shops = append(shops, dto.RecommendedShopResponse{
			ShopID:     item.Shop.ShopID,
			Name:       item.Shop.Name,
			Address:    item.Shop.Address,
			TrustScore: item.Trust.Score,
			TrustGrade: item.Trust.Grade,
			Rating:     item.Rating.AverageRating,
			Score:      item.Score,
			Reasons:    item.Reasons,
			DistanceKm: item.DistanceKm,
		})
	}

	c.JSON(http.StatusOK, dto.RecommendationsResponse{
		Products:     products,
		Shops:        shops,
		Personalised: result.Personalised,
		SignalCount:  result.SignalCount,
		Categories:   result.Categories,
	})
}
