package main

import (
	"context"
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
	vaultpkg "vngrocery/pkg/vault"
	visionpkg "vngrocery/pkg/vision"
)

type vaultAccountKeyStore struct {
	client *vaultpkg.Client
}

func (s vaultAccountKeyStore) CreateAccountKey(ctx context.Context, userID string) (authservice.AccountKey, error) {
	key, err := s.client.CreateAccountKey(ctx, userID)
	if err != nil {
		return authservice.AccountKey{}, err
	}

	return authservice.AccountKey{
		PublicKey:  key.PublicKey,
		Algorithm:  key.Algorithm,
		VaultPath:  key.VaultPath,
		PrivateKey: key.PrivateKey,
	}, nil
}

func (s vaultAccountKeyStore) DeleteAccountKey(ctx context.Context, vaultPath string) error {
	return s.client.DeleteAccountKey(ctx, vaultPath)
}

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
	shopReviewRepository := firestorerepo.NewShopReviewRepository(app.Firestore)
	userRepository := firestorerepo.NewUserRepository(app.Firestore)
	authUserRepository := firestorerepo.NewAuthUserRepository(app.Firestore)
	var accountKeys authservice.AccountKeyStore
	if cfg.VaultEnabled {
		accountKeys = vaultAccountKeyStore{client: vaultpkg.NewClient(vaultpkg.Config{
			Address:        cfg.VaultAddr,
			Token:          cfg.VaultToken,
			KVMount:        cfg.VaultKVMount,
			KeysPathPrefix: cfg.VaultKeysPathPrefix,
		})}
	}
	accountService := authservice.NewAccountService(authUserRepository, userRepository, accountKeys, nil, jwtService, 24*time.Hour, cfg.GoogleClientID)
	shopManager := shopservice.NewService(shopRepository, pledgeRepository, shopReviewRepository, userRepository)
	sellerCommitService := sellerservice.NewService(pledgeRepository, shopRepository)
	buyerCheckService := buyerservice.NewService(pledgeRepository, visionScorer)
	authMiddleware := middleware.NewAuthRequired(jwtService)
	healthHandler := handler.NewHealthHandler()
	docsHandler := handler.NewDocsHandler()
	authHandler := handler.NewAuthHandler(accountService)
	sellerHandler := handler.NewSellerHandler(visionScorer, sellerCommitService)
	buyerHandler := handler.NewBuyerHandler(buyerCheckService)
	shopHandler := handler.NewShopHandler(shopManager)

	engine := router.New(router.Dependencies{
		HealthHandler:  healthHandler,
		DocsHandler:    docsHandler,
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
