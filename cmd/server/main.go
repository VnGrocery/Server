package main

import (
	"log"

	"vngrocery/internal/api/handler"
	"vngrocery/internal/api/middleware"
	"vngrocery/internal/api/router"
	authservice "vngrocery/internal/service/auth"
	"vngrocery/pkg/config"
	firebasepkg "vngrocery/pkg/firebase"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Loi load config ne: %v", err)
	}

	app, err := firebasepkg.NewApp(cfg)
	if err != nil {
		log.Fatalf("Loi khoi tao Firebase roi: %v", err)
	}
	defer func() {
		if closeErr := app.Close(); closeErr != nil {
			log.Printf("Dong ket noi Firebase bi loi nha: %v", closeErr)
		}
	}()

	authVerifier := authservice.NewVerifier(app.AuthVerifier)
	authMiddleware := middleware.NewAuthRequired(authVerifier)
	healthHandler := handler.NewHealthHandler()
	authHandler := handler.NewAuthHandler()

	engine := router.New(router.Dependencies{
		HealthHandler:  healthHandler,
		AuthHandler:    authHandler,
		AuthMiddleware: authMiddleware,
	})

	if err := engine.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Server toang luc start: %v", err)
	}
}
