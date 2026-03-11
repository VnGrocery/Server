package main

import (
	"context"
	"log"
	"time"

	"vngrocery/internal/api/handler"
	"vngrocery/internal/api/middleware"
	"vngrocery/internal/api/router"
	firestorerepo "vngrocery/internal/repository/firestore"
	auditservice "vngrocery/internal/service/audit"
	authservice "vngrocery/internal/service/auth"
	buyerservice "vngrocery/internal/service/buyer"
	productservice "vngrocery/internal/service/product"
	sellerservice "vngrocery/internal/service/seller"
	shopservice "vngrocery/internal/service/shop"
	useradminservice "vngrocery/internal/service/useradmin"
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
	productRepository := firestorerepo.NewProductRepository(app.Firestore)
	shopRepository := firestorerepo.NewShopRepository(app.Firestore)
	shopReviewRepository := firestorerepo.NewShopReviewRepository(app.Firestore)
	userRepository := firestorerepo.NewUserRepository(app.Firestore)
	authUserRepository := firestorerepo.NewAuthUserRepository(app.Firestore)
	refreshTokenRepository := firestorerepo.NewRefreshTokenRepository(app.Firestore)
	passwordResetTokenRepository := firestorerepo.NewPasswordResetTokenRepository(app.Firestore)
	eventLogRepository := firestorerepo.NewEventLogRepository(app.Firestore)
	var accountKeys authservice.AccountKeyStore
	var auditSigner auditservice.Signer
	if cfg.VaultEnabled {
		vaultClient := vaultpkg.NewClient(vaultpkg.Config{
			Address:        cfg.VaultAddr,
			Token:          cfg.VaultToken,
			KVMount:        cfg.VaultKVMount,
			KeysPathPrefix: cfg.VaultKeysPathPrefix,
		})
		accountKeys = vaultAccountKeyStore{client: vaultClient}
		auditSigner = vaultClient
	}
	auditQueryService := auditservice.NewService(eventLogRepository, authUserRepository, auditSigner)
	var auditLogger *auditservice.Service
	if auditSigner != nil {
		auditLogger = auditQueryService
	}
	accountService := authservice.NewAccountService(authUserRepository, userRepository, refreshTokenRepository, passwordResetTokenRepository, accountKeys, auditLogger, nil, jwtService, 24*time.Hour, 30*24*time.Hour, cfg.GoogleClientID)
	productManager := productservice.NewService(productRepository, shopRepository, auditLogger)
	userAdminService := useradminservice.NewService(userRepository, auditLogger)
	shopManager := shopservice.NewService(shopRepository, pledgeRepository, shopReviewRepository, userRepository, auditLogger)
	sellerCommitService := sellerservice.NewService(pledgeRepository, shopRepository, auditLogger)
	buyerCheckService := buyerservice.NewService(pledgeRepository, visionScorer)
	authMiddleware := middleware.NewAuthRequired(jwtService)
	healthHandler := handler.NewHealthHandler()
	docsHandler := handler.NewDocsHandler()
	authHandler := handler.NewAuthHandler(accountService)
	adminUserHandler := handler.NewAdminUserHandler(userAdminService)
	eventLogHandler := handler.NewEventLogHandler(auditQueryService)
	productHandler := handler.NewProductHandler(productManager)
	sellerHandler := handler.NewSellerHandler(visionScorer, sellerCommitService)
	buyerHandler := handler.NewBuyerHandler(buyerCheckService)
	shopHandler := handler.NewShopHandler(shopManager)

	engine := router.New(router.Dependencies{
		HealthHandler:    healthHandler,
		DocsHandler:      docsHandler,
		AuthHandler:      authHandler,
		AdminUserHandler: adminUserHandler,
		EventLogHandler:  eventLogHandler,
		ProductHandler:   productHandler,
		SellerHandler:    sellerHandler,
		BuyerHandler:     buyerHandler,
		ShopHandler:      shopHandler,
		AuthMiddleware:   authMiddleware,
	})

	if err := engine.Run(":" + cfg.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
