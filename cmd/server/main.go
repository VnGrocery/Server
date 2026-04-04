package main

import (
	"context"
	"log"
	"strconv"
	"time"

	"vngrocery/internal/api/handler"
	"vngrocery/internal/api/middleware"
	"vngrocery/internal/api/router"
	"vngrocery/internal/domain"
	firestorerepo "vngrocery/internal/repository/firestore"
	auditservice "vngrocery/internal/service/audit"
	authservice "vngrocery/internal/service/auth"
	buyerservice "vngrocery/internal/service/buyer"
	integrityservice "vngrocery/internal/service/integrity"
	productservice "vngrocery/internal/service/product"
	sellerservice "vngrocery/internal/service/seller"
	shopservice "vngrocery/internal/service/shop"
	useradminservice "vngrocery/internal/service/useradmin"
	visionservice "vngrocery/internal/service/vision"
	alertpkg "vngrocery/pkg/alert"
	besupkg "vngrocery/pkg/besu"
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
	buyerCheckRepository := firestorerepo.NewBuyerCheckRepository(app.Firestore)
	productRepository := firestorerepo.NewProductRepository(app.Firestore)
	productFreshnessReportRepository := firestorerepo.NewProductFreshnessReportRepository(app.Firestore)
	shopRepository := firestorerepo.NewShopRepository(app.Firestore)
	shopReviewRepository := firestorerepo.NewShopReviewRepository(app.Firestore)
	userRepository := firestorerepo.NewUserRepository(app.Firestore)
	authUserRepository := firestorerepo.NewAuthUserRepository(app.Firestore)
	refreshTokenRepository := firestorerepo.NewRefreshTokenRepository(app.Firestore)
	passwordResetTokenRepository := firestorerepo.NewPasswordResetTokenRepository(app.Firestore)
	eventLogRepository := firestorerepo.NewEventLogRepository(app.Firestore)
	metrics := middleware.NewMetrics()
	var rateLimitStore middleware.RateLimitStore
	if cfg.RateLimitBackend == "firestore" {
		rateLimitStore = middleware.NewFirestoreRateLimitStore(app.Firestore, cfg.RateLimitCollection)
	} else {
		rateLimitStore = middleware.NewMemoryRateLimitStore()
	}
	var accountKeys authservice.AccountKeyStore
	var auditSigner auditservice.Signer
	var integrityManager *integrityservice.Service
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
	integrityManager = integrityservice.NewService(pledgeRepository, nil, auditLogger)
	if cfg.BesuEnabled {
		besuClient := besupkg.NewClient(besupkg.Config{
			RPCURL:          cfg.BesuRPCURL,
			ContractAddress: cfg.BesuContractAddress,
			FromAddress:     cfg.BesuFromAddress,
			PrivateKey:      cfg.BesuPrivateKey,
			ChainID:         cfg.BesuChainID,
			GasLimit:        mustParseUint(cfg.BesuGasLimit, 250000),
			ReceiptTimeout:  time.Duration(mustParseInt(cfg.BesuReceiptTimeoutSec, 15)) * time.Second,
		})
		integrityManager = integrityservice.NewService(pledgeRepository, besuClientAdapter{client: besuClient}, auditLogger)
		integrityManager.StartBackground(context.Background(), integrityservice.WorkerConfig{
			PendingInterval: time.Duration(mustParseInt(cfg.BesuPendingIntervalSec, 10)) * time.Second,
			VerifyInterval:  time.Duration(mustParseInt(cfg.BesuVerifyIntervalSec, 60)) * time.Second,
			PendingBatch:    mustParseInt(cfg.BesuPendingBatchSize, 25),
			VerifyBatch:     mustParseInt(cfg.BesuVerifyBatchSize, 50),
		})
	}
	if cfg.AlertWebhookURL != "" {
		integrityManager.SetNotifier(alertAdapter{
			client: alertpkg.NewWebhookClient(cfg.AlertWebhookURL, time.Duration(mustParseInt(cfg.AlertTimeoutSec, 5))*time.Second),
		})
	}
	integrityManager.SetObserver(metrics)
	accountService := authservice.NewAccountService(authUserRepository, userRepository, refreshTokenRepository, passwordResetTokenRepository, accountKeys, auditLogger, nil, jwtService, 24*time.Hour, 30*24*time.Hour, cfg.GoogleClientID)
	productManager := productservice.NewService(productRepository, productFreshnessReportRepository, shopRepository, userRepository, auditLogger)
	userAdminService := useradminservice.NewService(userRepository, authUserRepository, accountKeys, auditLogger)
	shopManager := shopservice.NewService(shopRepository, pledgeRepository, buyerCheckRepository, shopReviewRepository, userRepository, auditLogger)
	sellerCommitService := sellerservice.NewService(pledgeRepository, shopRepository, productRepository, auditLogger)
	shopManager.SetPledgeIntegrityReader(integrityAdapter{service: integrityManager})
	sellerCommitService.SetIntegrityManager(integrityManager)
	buyerCheckService := buyerservice.NewService(pledgeRepository, buyerCheckRepository, userRepository, visionScorer, auditLogger)
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
		HealthHandler:        healthHandler,
		DocsHandler:          docsHandler,
		AuthHandler:          authHandler,
		AdminUserHandler:     adminUserHandler,
		EventLogHandler:      eventLogHandler,
		ProductHandler:       productHandler,
		SellerHandler:        sellerHandler,
		BuyerHandler:         buyerHandler,
		ShopHandler:          shopHandler,
		AuthMiddleware:       authMiddleware,
		Metrics:              metrics,
		RateLimitStore:       rateLimitStore,
		RateLimitMaxRequests: mustParseInt(cfg.RateLimitMaxRequests, 120),
		RateLimitWindow:      time.Duration(mustParseInt(cfg.RateLimitWindowSec, 60)) * time.Second,
	})

	if err := engine.Run(":" + cfg.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

type besuClientAdapter struct {
	client *besupkg.Client
}

func (a besuClientAdapter) CommitHash(ctx context.Context, recordID, dataHash string, timestamp time.Time, version int) (integrityservice.CommitResult, error) {
	result, err := a.client.CommitHash(ctx, recordID, dataHash, timestamp, version)
	if err != nil {
		return integrityservice.CommitResult{}, err
	}
	return integrityservice.CommitResult{
		TxHash:      result.TxHash,
		BlockNumber: result.BlockNumber,
		BlockTime:   result.BlockTime,
		Mined:       result.Mined,
	}, nil
}

func (a besuClientAdapter) RevokeHash(ctx context.Context, recordID string, version int) (integrityservice.CommitResult, error) {
	result, err := a.client.RevokeHash(ctx, recordID, version)
	if err != nil {
		return integrityservice.CommitResult{}, err
	}
	return integrityservice.CommitResult{
		TxHash:      result.TxHash,
		BlockNumber: result.BlockNumber,
		BlockTime:   result.BlockTime,
		Mined:       result.Mined,
	}, nil
}

func (a besuClientAdapter) Verify(ctx context.Context, recordID, dataHash string) (bool, error) {
	return a.client.Verify(ctx, recordID, dataHash)
}

func (a besuClientAdapter) GetLatest(ctx context.Context, recordID string) (integrityservice.LatestRecord, error) {
	result, err := a.client.GetLatest(ctx, recordID)
	if err != nil {
		return integrityservice.LatestRecord{}, err
	}
	return integrityservice.LatestRecord{
		DataHash:  result.DataHash,
		Timestamp: result.Timestamp,
		Version:   result.Version,
		IsRevoked: result.IsRevoked,
		IsPresent: result.IsPresent,
	}, nil
}

func (a besuClientAdapter) Receipt(ctx context.Context, txHash string) (integrityservice.CommitResult, error) {
	result, err := a.client.Receipt(ctx, txHash)
	if err != nil {
		return integrityservice.CommitResult{}, err
	}
	return integrityservice.CommitResult{
		TxHash:      result.TxHash,
		BlockNumber: result.BlockNumber,
		BlockTime:   result.BlockTime,
		Mined:       result.Mined,
	}, nil
}

type integrityAdapter struct {
	service *integrityservice.Service
}

func (a integrityAdapter) GetPledgeIntegrity(ctx context.Context, pledge domain.Pledge) (shopservice.PledgeIntegrityView, error) {
	result, err := a.service.GetPledgeIntegrity(ctx, pledge)
	if err != nil {
		return shopservice.PledgeIntegrityView{}, err
	}
	return shopservice.PledgeIntegrityView{
		PledgeID:          result.PledgeID,
		ShopID:            result.ShopID,
		DataHash:          result.DataHash,
		ProvidedDataHash:  result.ProvidedDataHash,
		ChainTxHash:       result.ChainTxHash,
		ChainBlockNumber:  result.ChainBlockNumber,
		ChainAnchorStatus: result.ChainAnchorStatus,
		ChainAnchorTime:   result.ChainAnchorTime,
		IntegrityStatus:   result.IntegrityStatus,
		OnChainMatch:      result.OnChainMatch,
		ProvidedHashMatch: result.ProvidedHashMatch,
		OnChainDataHash:   result.OnChainDataHash,
		OnChainVersion:    result.OnChainVersion,
		OnChainTimestamp:  result.OnChainTimestamp,
		OnChainPresent:    result.OnChainPresent,
		MismatchReason:    result.MismatchReason,
		LastCheckedAt:     result.LastCheckedAt,
		CanReanchor:       result.CanReanchor,
		CanRevoke:         result.CanRevoke,
	}, nil
}

func (a integrityAdapter) VerifyPledgeHash(ctx context.Context, pledge domain.Pledge, dataHash string) (shopservice.PledgeIntegrityView, error) {
	result, err := a.service.VerifyPledgeHash(ctx, pledge, dataHash)
	if err != nil {
		return shopservice.PledgeIntegrityView{}, err
	}
	return shopservice.PledgeIntegrityView{
		PledgeID:          result.PledgeID,
		ShopID:            result.ShopID,
		DataHash:          result.DataHash,
		ProvidedDataHash:  result.ProvidedDataHash,
		ChainTxHash:       result.ChainTxHash,
		ChainBlockNumber:  result.ChainBlockNumber,
		ChainAnchorStatus: result.ChainAnchorStatus,
		ChainAnchorTime:   result.ChainAnchorTime,
		IntegrityStatus:   result.IntegrityStatus,
		OnChainMatch:      result.OnChainMatch,
		ProvidedHashMatch: result.ProvidedHashMatch,
		OnChainDataHash:   result.OnChainDataHash,
		OnChainVersion:    result.OnChainVersion,
		OnChainTimestamp:  result.OnChainTimestamp,
		OnChainPresent:    result.OnChainPresent,
		MismatchReason:    result.MismatchReason,
		LastCheckedAt:     result.LastCheckedAt,
		CanReanchor:       result.CanReanchor,
		CanRevoke:         result.CanRevoke,
	}, nil
}

func (a integrityAdapter) ReanchorPledge(ctx context.Context, pledge domain.Pledge) (domain.Pledge, error) {
	return a.service.ReanchorPledge(ctx, pledge)
}

func (a integrityAdapter) RevokePledge(ctx context.Context, pledge domain.Pledge) (domain.Pledge, error) {
	return a.service.RevokePledge(ctx, pledge)
}

func mustParseInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func mustParseUint(raw string, fallback uint64) uint64 {
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || value == 0 {
		return fallback
	}
	return value
}

type alertAdapter struct {
	client *alertpkg.WebhookClient
}

func (a alertAdapter) NotifyIntegrityMismatch(ctx context.Context, payload integrityservice.IntegrityAlertPayload) error {
	return a.client.NotifyIntegrityMismatch(ctx, alertpkg.IntegrityMismatchPayload{
		Event:            "pledge.integrity_mismatch_detected",
		PledgeID:         payload.PledgeID,
		ShopID:           payload.ShopID,
		CreatedByUserID:  payload.CreatedByUserID,
		DataHash:         payload.DataHash,
		ChainTxHash:      payload.ChainTxHash,
		IntegrityStatus:  payload.IntegrityStatus,
		DetectedAt:       payload.DetectedAt,
		OnChainDataHash:  payload.OnChainDataHash,
		OnChainVersion:   payload.OnChainVersion,
		OnChainTimestamp: payload.OnChainTimestamp,
	})
}
