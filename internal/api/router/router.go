package router

import (
	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/handler"
	"vngrocery/internal/api/middleware"
)

type Dependencies struct {
	HealthHandler  *handler.HealthHandler
	DocsHandler    *handler.DocsHandler
	AuthHandler    *handler.AuthHandler
	SellerHandler  *handler.SellerHandler
	BuyerHandler   *handler.BuyerHandler
	ShopHandler    *handler.ShopHandler
	AuthMiddleware *middleware.AuthRequired
}

func New(deps Dependencies) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())

	engine.GET("/health", deps.HealthHandler.Health)
	engine.GET("/docs", deps.DocsHandler.SwaggerUI)
	engine.GET("/openapi.json", deps.DocsHandler.OpenAPI)

	v1 := engine.Group("/v1")
	{
		v1.GET("/health", deps.HealthHandler.Health)
		v1.POST("/auth/register", deps.AuthHandler.Register)
		v1.POST("/auth/login", deps.AuthHandler.Login)
		v1.POST("/auth/google", deps.AuthHandler.GoogleLogin)
		v1.GET("/shops", deps.ShopHandler.List)
		v1.GET("/shops/:shopId", deps.ShopHandler.GetByID)
		v1.GET("/shops/:shopId/reviews", deps.ShopHandler.ListReviews)
		v1.GET("/me", deps.AuthMiddleware.Handle(), deps.AuthHandler.Me)
		v1.POST("/shops", deps.AuthMiddleware.Handle(), deps.ShopHandler.Create)
		v1.PUT("/shops/:shopId", deps.AuthMiddleware.Handle(), deps.ShopHandler.Update)
		v1.POST("/shops/:shopId/reviews", deps.AuthMiddleware.Handle(), deps.ShopHandler.CreateReview)
		v1.GET("/admin/shops", deps.AuthMiddleware.Handle(), deps.ShopHandler.AdminList)
		v1.PATCH("/admin/shops/:shopId/moderation", deps.AuthMiddleware.Handle(), deps.ShopHandler.Moderate)
		v1.POST("/seller/score", deps.AuthMiddleware.Handle(), deps.SellerHandler.Score)
		v1.POST("/seller/commit", deps.AuthMiddleware.Handle(), deps.SellerHandler.Commit)
		v1.POST("/buyer/check", deps.BuyerHandler.Check)
	}

	return engine
}
