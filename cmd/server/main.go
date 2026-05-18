package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"log"
	"strconv"
	"strings"
	"time"

	"vngrocery/internal/api/handler"
	"vngrocery/internal/api/middleware"
	"vngrocery/internal/api/router"
	"vngrocery/internal/domain"
	"vngrocery/internal/repository"
	cacherepo "vngrocery/internal/repository/cache"
	firestorerepo "vngrocery/internal/repository/firestore"
	mongorepo "vngrocery/internal/repository/mongo"
	auditservice "vngrocery/internal/service/audit"
	authservice "vngrocery/internal/service/auth"
	bundletokenservice "vngrocery/internal/service/bundletoken"
	buyerservice "vngrocery/internal/service/buyer"
	integrityservice "vngrocery/internal/service/integrity"
	productservice "vngrocery/internal/service/product"
	sellerservice "vngrocery/internal/service/seller"
	shopservice "vngrocery/internal/service/shop"
	useradminservice "vngrocery/internal/service/useradmin"
	visionservice "vngrocery/internal/service/vision"
	alertpkg "vngrocery/pkg/alert"
	besupkg "vngrocery/pkg/besu"
	cachepkg "vngrocery/pkg/cache"
	"vngrocery/pkg/config"
	firebasepkg "vngrocery/pkg/firebase"
	ipfspkg "vngrocery/pkg/ipfs"
	mongopkg "vngrocery/pkg/mongodb"
	vaultpkg "vngrocery/pkg/vault"
	visionpkg "vngrocery/pkg/vision"
)

type vaultAccountKeyStore struct {
	client *vaultpkg.Client
}

type localAccountKeyStore struct{}

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

func (s localAccountKeyStore) CreateAccountKey(ctx context.Context, userID string) (authservice.AccountKey, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return authservice.AccountKey{}, err
	}
	return authservice.AccountKey{
		PublicKey:  base64.StdEncoding.EncodeToString(publicKey),
		Algorithm:  "Ed25519",
		VaultPath:  "local/dev/" + strings.TrimSpace(userID),
		PrivateKey: base64.StdEncoding.EncodeToString(privateKey),
	}, nil
}

func (s localAccountKeyStore) DeleteAccountKey(ctx context.Context, vaultPath string) error {
	return nil
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	if err := cfg.ValidateVision(); err != nil {
		log.Fatalf("failed to validate vision config: %v", err)
	}

	jwtService := authservice.NewJWTService(cfg.JWTSecret, "vngrocery")
	var visionScorer visionservice.ImageScorer = visionservice.NewService(nil)
	if cfg.HasVisionProvider() {
		visionClient := visionpkg.NewOpenAIClient(cfg)
		visionScorer = visionservice.NewService(visionClient)
	}
	var rateLimitStore middleware.RateLimitStore
	var pledgeRepository repository.PledgeRepository
	var buyerCheckRepository repository.BuyerCheckRepository
	var productRepository repository.ProductRepository
	var productFreshnessReportRepository repository.ProductFreshnessReportRepository
	var shopRepository repository.ShopRepository
	var shopReviewRepository repository.ShopReviewRepository
	var userRepository repository.UserRepository
	var authUserRepository repository.AuthUserRepository
	var refreshTokenRepository repository.RefreshTokenRepository
	var passwordResetTokenRepository repository.PasswordResetTokenRepository
	var bundleTokenUseRepository repository.BundleTokenUseRepository
	var eventLogRepository repository.EventLogRepository
	var firebaseApp *firebasepkg.App
	var mongoApp *mongopkg.App
	var cacheStore *cachepkg.Store
	if cfg.CacheBackend == "redis" {
		redisDB, err := strconv.Atoi(cfg.RedisDB)
		if err != nil || redisDB < 0 {
			log.Fatalf("invalid REDIS_DB: %q", cfg.RedisDB)
		}
		redisKV := cachepkg.NewRedisKV(cfg.RedisAddr, cfg.RedisPassword, redisDB)
		pingCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := redisKV.Ping(pingCtx); err != nil {
			log.Fatalf("failed to connect redis cache: %v", err)
		}
		defer func() {
			if closeErr := redisKV.Close(); closeErr != nil {
				log.Printf("failed to close Redis resources: %v", closeErr)
			}
		}()
		cacheStore = cachepkg.NewStore(redisKV, cfg.CachePrefix, time.Duration(mustParseInt(cfg.CacheTTLSeconds, 30))*time.Second)
	}

	if cfg.UseMongo() {
		mongoApp, err = mongopkg.NewApp(cfg)
		if err != nil {
			log.Fatalf("failed to initialize MongoDB: %v", err)
		}
		defer func() {
			if closeErr := mongoApp.Close(); closeErr != nil {
				log.Printf("failed to close MongoDB resources: %v", closeErr)
			}
		}()

		pledgeRepository = mongorepo.NewPledgeRepository(mongoApp.Database)
		buyerCheckRepository = mongorepo.NewBuyerCheckRepository(mongoApp.Database)
		productRepository = mongorepo.NewProductRepository(mongoApp.Database)
		productFreshnessReportRepository = mongorepo.NewProductFreshnessReportRepository(mongoApp.Database)
		shopRepository = mongorepo.NewShopRepository(mongoApp.Database)
		shopReviewRepository = mongorepo.NewShopReviewRepository(mongoApp.Database)
		userRepository = mongorepo.NewUserRepository(mongoApp.Database)
		authUserRepository = mongorepo.NewAuthUserRepository(mongoApp.Database)
		refreshTokenRepository = mongorepo.NewRefreshTokenRepository(mongoApp.Database)
		passwordResetTokenRepository = mongorepo.NewPasswordResetTokenRepository(mongoApp.Database)
		bundleTokenUseRepository = mongorepo.NewBundleTokenUseRepository(mongoApp.Database)
		eventLogRepository = mongorepo.NewEventLogRepository(mongoApp.Database)
		rateLimitStore = middleware.NewMemoryRateLimitStore()
		if cfg.RateLimitBackend == "firestore" {
			log.Printf("RATE_LIMIT_BACKEND=firestore ignored because MongoDB is active; using memory rate limit store")
		}
	} else {
		firebaseApp, err = firebasepkg.NewApp(cfg)
		if err != nil {
			log.Fatalf("failed to initialize Firebase: %v", err)
		}
		defer func() {
			if closeErr := firebaseApp.Close(); closeErr != nil {
				log.Printf("failed to close Firebase resources: %v", closeErr)
			}
		}()

		pledgeRepository = firestorerepo.NewPledgeRepository(firebaseApp.Firestore)
		buyerCheckRepository = firestorerepo.NewBuyerCheckRepository(firebaseApp.Firestore)
		productRepository = firestorerepo.NewProductRepository(firebaseApp.Firestore)
		productFreshnessReportRepository = firestorerepo.NewProductFreshnessReportRepository(firebaseApp.Firestore)
		shopRepository = firestorerepo.NewShopRepository(firebaseApp.Firestore)
		shopReviewRepository = firestorerepo.NewShopReviewRepository(firebaseApp.Firestore)
		userRepository = firestorerepo.NewUserRepository(firebaseApp.Firestore)
		authUserRepository = firestorerepo.NewAuthUserRepository(firebaseApp.Firestore)
		refreshTokenRepository = firestorerepo.NewRefreshTokenRepository(firebaseApp.Firestore)
		passwordResetTokenRepository = firestorerepo.NewPasswordResetTokenRepository(firebaseApp.Firestore)
		bundleTokenUseRepository = firestorerepo.NewBundleTokenUseRepository(firebaseApp.Firestore)
		eventLogRepository = firestorerepo.NewEventLogRepository(firebaseApp.Firestore)
		if cfg.RateLimitBackend == "firestore" {
			rateLimitStore = middleware.NewFirestoreRateLimitStore(firebaseApp.Firestore, cfg.RateLimitCollection)
		} else {
			rateLimitStore = middleware.NewMemoryRateLimitStore()
		}
	}
	if cacheStore != nil && cacheStore.IsEnabled() {
		shopRepository = cacherepo.NewShopRepository(shopRepository, cacheStore)
		productRepository = cacherepo.NewProductRepository(productRepository, cacheStore)
		eventLogRepository = cacherepo.NewEventLogRepository(eventLogRepository, cacheStore)
		log.Printf("cache enabled: backend=%s ttl=%ss prefix=%s", cfg.CacheBackend, cfg.CacheTTLSeconds, cfg.CachePrefix)
	}
	metrics := middleware.NewMetrics()
	var accountKeys authservice.AccountKeyStore
	var auditSigner auditservice.Signer
	var integrityManager *integrityservice.Service
	var ipfsClient *ipfspkg.Client
	if cfg.IPFSEnabled {
		ipfsClient = ipfspkg.NewClient(cfg.IPFSAPIURL, cfg.IPFSGatewayURL)
	}
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
	if accountKeys == nil {
		accountKeys = localAccountKeyStore{}
		log.Printf("VAULT_ENABLED=false; using local in-process account key store for development")
	}
	auditQueryService := auditservice.NewService(eventLogRepository, authUserRepository, auditSigner)
	var auditLogger *auditservice.Service
	if auditSigner != nil {
		auditLogger = auditQueryService
	}
	integrityManager = integrityservice.NewService(pledgeRepository, nil, auditLogger)
	integrityManager.SetShopRepository(shopRepository)
	if cfg.BesuEnabled {
		besuClient := besupkg.NewClient(besupkg.Config{
			RPCURL:          cfg.BesuRPCURL,
			ContractAddress: cfg.BesuContractAddress,
			FromAddress:     cfg.BesuFromAddress,
			PrivateKey:      cfg.BesuPrivateKey,
			ChainID:         cfg.BesuChainID,
			GasLimit:        mustParseUint(cfg.BesuGasLimit, 250000),
			ReceiptTimeout:  time.Duration(mustParseInt(cfg.BesuReceiptTimeoutSec, 30)) * time.Second,
		})
		integrityManager = integrityservice.NewService(pledgeRepository, besuClientAdapter{client: besuClient}, auditLogger)
		integrityManager.SetShopRepository(shopRepository)
		integrityManager.StartBackground(context.Background(), integrityservice.WorkerConfig{
			PendingInterval: time.Duration(mustParseInt(cfg.BesuPendingIntervalSec, 3)) * time.Second,
			VerifyInterval:  time.Duration(mustParseInt(cfg.BesuVerifyIntervalSec, 8)) * time.Second,
			PendingBatch:    mustParseInt(cfg.BesuPendingBatchSize, 25),
			VerifyBatch:     mustParseInt(cfg.BesuVerifyBatchSize, 50),
		})
	}
	alertNotifier := alertpkg.NewMultiNotifier(
		alertpkg.NewWebhookClient(cfg.AlertWebhookURL, time.Duration(mustParseInt(cfg.AlertTimeoutSec, 5))*time.Second),
		alertpkg.NewSlackClient(cfg.AlertSlackWebhookURL, time.Duration(mustParseInt(cfg.AlertTimeoutSec, 5))*time.Second),
		alertpkg.NewTelegramClient(cfg.AlertTelegramBotToken, cfg.AlertTelegramChatID, time.Duration(mustParseInt(cfg.AlertTimeoutSec, 5))*time.Second),
		alertpkg.NewSMTPClient(cfg.AlertSMTPHost, cfg.AlertSMTPPort, cfg.AlertSMTPUsername, cfg.AlertSMTPPassword, cfg.AlertSMTPFrom, cfg.AlertSMTPTo),
	)
	integrityManager.SetNotifier(alertAdapter{client: alertNotifier})
	integrityManager.SetObserver(metrics)
	accountService := authservice.NewAccountService(
		authUserRepository,
		userRepository,
		refreshTokenRepository,
		passwordResetTokenRepository,
		accountKeys,
		auditLogger,
		nil,
		jwtService,
		24*time.Hour,
		30*24*time.Hour,
		cfg.GoogleClientID,
		splitCommaSeparated(cfg.BootstrapAdminEmails)...,
	)
	productManager := productservice.NewService(productRepository, productFreshnessReportRepository, shopRepository, userRepository, auditLogger)
	userAdminService := useradminservice.NewService(userRepository, authUserRepository, accountKeys, auditLogger)
	shopManager := shopservice.NewService(shopRepository, pledgeRepository, buyerCheckRepository, shopReviewRepository, userRepository, auditLogger)
	sellerCommitService := sellerservice.NewService(pledgeRepository, shopRepository, productRepository, auditLogger)
	shopManager.SetPledgeIntegrityReader(integrityAdapter{service: integrityManager})
	shopManager.SetShopIntegrityManager(integrityManager)
	sellerCommitService.SetIntegrityManager(integrityManager)
	buyerCheckService := buyerservice.NewService(pledgeRepository, buyerCheckRepository, userRepository, visionScorer, auditLogger)
	bundleTokenService := bundletokenservice.NewService(cfg.JWTSecret, "vngrocery", 30*time.Minute, bundleTokenUseRepository)
	bundleTokenService.SetObserver(metrics)
	bundleTokenService.StartCleanup(context.Background(), 10*time.Minute, 500)
	buyerCheckService.SetBundleTokenVerifier(bundleTokenService)
	buyerCheckService.SetObserver(metrics)
	authMiddleware := middleware.NewAuthRequired(jwtService)
	adminMiddleware := middleware.NewAdminRequired(userRepository)
	healthHandler := handler.NewHealthHandler()
	docsHandler := handler.NewDocsHandler()
	authHandler := handler.NewAuthHandler(accountService)
	adminUserHandler := handler.NewAdminUserHandler(userAdminService)
	eventLogHandler := handler.NewEventLogHandler(auditQueryService)
	productHandler := handler.NewProductHandler(productManager)
	sellerHandler := handler.NewSellerHandler(visionScorer, sellerCommitService)
	sellerHandler.SetBundleTokenIssuer(bundleTokenService)
	buyerHandler := handler.NewBuyerHandler(buyerCheckService)
	uploadCfg := handler.NewMediaUploadConfigForRuntime(
		int64(mustParseInt(cfg.MediaMaxImageBytes, 10<<20)),
		splitCommaSeparated(cfg.MediaAllowedTypes),
	)
	sellerHandler.SetUploadConfig(uploadCfg)
	buyerHandler.SetUploadConfig(uploadCfg)
	if ipfsClient != nil {
		uploader := ipfsUploadAdapter{client: ipfsClient}
		sellerHandler.SetUploader(uploader)
		buyerHandler.SetUploader(uploader)
	}
	mediaHandler := handler.NewMediaHandler(ipfsUploadAdapterOrNil(ipfsClient), uploadCfg)
	shopHandler := handler.NewShopHandler(shopManager)

	engine := router.New(router.Dependencies{
		HealthHandler:             healthHandler,
		DocsHandler:               docsHandler,
		AuthHandler:               authHandler,
		AdminUserHandler:          adminUserHandler,
		EventLogHandler:           eventLogHandler,
		MediaHandler:              mediaHandler,
		ProductHandler:            productHandler,
		SellerHandler:             sellerHandler,
		BuyerHandler:              buyerHandler,
		ShopHandler:               shopHandler,
		AuthMiddleware:            authMiddleware,
		AdminMiddleware:           adminMiddleware,
		Metrics:                   metrics,
		RateLimitStore:            rateLimitStore,
		RateLimitMaxRequests:      mustParseInt(cfg.RateLimitMaxRequests, 120),
		RateLimitWindow:           time.Duration(mustParseInt(cfg.RateLimitWindowSec, 60)) * time.Second,
		AdminRateLimitMaxRequests: mustParseInt(cfg.AdminRateLimitMaxRequests, 30),
		AdminRateLimitWindow:      time.Duration(mustParseInt(cfg.AdminRateLimitWindowSec, 60)) * time.Second,
	})

	if err := engine.Run(":" + cfg.Port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

type besuClientAdapter struct {
	client *besupkg.Client
}

type ipfsUploadAdapter struct {
	client *ipfspkg.Client
}

func ipfsUploadAdapterOrNil(client *ipfspkg.Client) handler.ImageUploader {
	if client == nil {
		return nil
	}
	return ipfsUploadAdapter{client: client}
}

func (a ipfsUploadAdapter) AddBytes(ctx context.Context, filename string, data []byte) (handler.ImageUploadResult, error) {
	result, err := a.client.AddBytes(ctx, filename, data)
	if err != nil {
		return handler.ImageUploadResult{}, err
	}
	return handler.ImageUploadResult{
		CID:        result.CID,
		GatewayURL: result.GatewayURL,
	}, nil
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

func splitCommaSeparated(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}

type alertAdapter struct {
	client alertpkg.IntegrityNotifier
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
