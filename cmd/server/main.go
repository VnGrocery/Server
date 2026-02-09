package main

import (
	"log"
	"time"

	"vngrocery/internal/api/handler"
	"vngrocery/internal/api/middleware"
	"vngrocery/internal/api/router"
	firestorerepo "vngrocery/internal/repository/firestore"
	authservice "vngrocery/internal/service/auth"
	buyerservice "vngrocery/internal/service/buyer"
	sellerservice "vngrocery/internal/service/seller"
	shopservice "vngrocery/internal/service/shop"
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
	if err := cfg.ValidateVision(); err != nil {
		log.Fatalf("failed to validate vision config: %v", err)
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

	jwtService := authservice.NewJWTService(cfg.JWTSecret, "vngrocery")
	var visionScorer visionservice.ImageScorer = visionservice.NewService(nil)
	if cfg.HasVisionProvider() {
		visionClient := visionpkg.NewOpenAIClient(cfg)
		visionScorer = visionservice.NewService(visionClient)
	}
	pledgeRepository := firestorerepo.NewPledgeRepository(app.Firestore)
	shopRepository := firestorerepo.NewShopRepository(app.Firestore)
	userRepository := firestorerepo.NewUserRepository(app.Firestore)
	authUserRepository := firestorerepo.NewAuthUserRepository(app.Firestore)
	accountService := authservice.NewAccountService(authUserRepository, userRepository, jwtService, 24*time.Hour, cfg.GoogleClientID)
	shopManager := shopservice.NewService(shopRepository)
	sellerCommitService := sellerservice.NewService(pledgeRepository, shopRepository)
	buyerCheckService := buyerservice.NewService(pledgeRepository, visionScorer)
	authMiddleware := middleware.NewAuthRequired(jwtService)
	healthHandler := handler.NewHealthHandler()
	authHandler := handler.NewAuthHandler(accountService)
	sellerHandler := handler.NewSellerHandler(visionScorer, sellerCommitService)
	buyerHandler := handler.NewBuyerHandler(buyerCheckService)
	shopHandler := handler.NewShopHandler(shopManager)

	engine := router.New(router.Dependencies{
		HealthHandler:  healthHandler,
		AuthHandler:    authHandler,
		SellerHandler:  sellerHandler,
		BuyerHandler:   buyerHandler,
		ShopHandler:    shopHandler,
		AuthMiddleware: authMiddleware,
	})

	if err := engine.Run(":" + cfg.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
