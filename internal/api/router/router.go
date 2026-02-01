package router

import (
	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/handler"
	"vngrocery/internal/api/middleware"
)

type Dependencies struct {
	HealthHandler  *handler.HealthHandler
	AuthHandler    *handler.AuthHandler
	AuthMiddleware *middleware.AuthRequired
}

func New(deps Dependencies) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())

	engine.GET("/health", deps.HealthHandler.Health)

	v1 := engine.Group("/v1")
	{
		v1.GET("/health", deps.HealthHandler.Health)
		v1.GET("/me", deps.AuthMiddleware.Handle(), deps.AuthHandler.Me)
	}

	return engine
}
