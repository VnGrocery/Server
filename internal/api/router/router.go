package router

import (
	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/handler"
	"vngrocery/internal/api/middleware"
)

type Dependencies struct {
	HealthHandler  *handler.HealthHandler
	AuthHandler    *handler.AuthHandler
	SellerHandler  *handler.SellerHandler
	BuyerHandler   *handler.BuyerHandler
	AuthMiddleware *middleware.AuthRequired
}

func New(deps Dependencies) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())

	engine.GET("/health", deps.HealthHandler.Health)

	v1 := engine.Group("/v1")
	{
		v1.GET("/health", deps.HealthHandler.Health)
		v1.POST("/auth/register", deps.AuthHandler.Register)
		v1.POST("/auth/login", deps.AuthHandler.Login)
		v1.POST("/auth/google", deps.AuthHandler.GoogleLogin)
		v1.GET("/me", deps.AuthMiddleware.Handle(), deps.AuthHandler.Me)
		v1.POST("/seller/score", deps.AuthMiddleware.Handle(), deps.SellerHandler.Score)
		v1.POST("/seller/commit", deps.AuthMiddleware.Handle(), deps.SellerHandler.Commit)
		v1.POST("/buyer/check", deps.BuyerHandler.Check)
	}

	return engine
}
