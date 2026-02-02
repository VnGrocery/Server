package main

import (
	"log"

	"vngrocery/internal/api/handler"
	"vngrocery/internal/api/middleware"
	"vngrocery/internal/api/router"
	authservice "vngrocery/internal/service/auth"
	visionservice "vngrocery/internal/service/vision"
	"vngrocery/pkg/config"
	firebasepkg "vngrocery/pkg/firebase"
	visionpkg "vngrocery/pkg/vision"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	app, err := firebasepkg.NewApp(cfg)
	if err != nil {
		log.Fatalf("failed to initialize Firebase: %v", err)
	}
	defer func() {
		if closeErr := app.Close(); closeErr != nil {
			log.Printf("failed to close Firebase resources: %v", closeErr)
		}
	}()

	authVerifier := authservice.NewVerifier(app.AuthVerifier)
	visionClient := visionpkg.NewOpenAIClient(cfg)
	visionScorer := visionservice.NewService(visionClient)
	authMiddleware := middleware.NewAuthRequired(authVerifier)
	healthHandler := handler.NewHealthHandler()
	authHandler := handler.NewAuthHandler()
	sellerHandler := handler.NewSellerHandler(visionScorer)

	engine := router.New(router.Dependencies{
		HealthHandler:  healthHandler,
		AuthHandler:    authHandler,
		SellerHandler:  sellerHandler,
		AuthMiddleware: authMiddleware,
	})

	if err := engine.Run(":" + cfg.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
