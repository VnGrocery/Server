package router

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/handler"
	"vngrocery/internal/api/middleware"
	cachepkg "vngrocery/pkg/cache"
)

type Dependencies struct {
	HealthHandler             *handler.HealthHandler
	DocsHandler               *handler.DocsHandler
	AuthHandler               *handler.AuthHandler
	AdminUserHandler          *handler.AdminUserHandler
	EventLogHandler           *handler.EventLogHandler
	MediaHandler              *handler.MediaHandler
	ProductHandler            *handler.ProductHandler
	ProductBatchHandler       *handler.ProductBatchHandler
	SellerHandler             *handler.SellerHandler
	BuyerHandler              *handler.BuyerHandler
	ShopHandler               *handler.ShopHandler
	AuthMiddleware            *middleware.AuthRequired
	AdminMiddleware           *middleware.AdminRequired
	Metrics                   *middleware.Metrics
	RateLimitStore            middleware.RateLimitStore
	RateLimitMaxRequests      int
	RateLimitWindow           time.Duration
	AdminRateLimitMaxRequests int
	AdminRateLimitWindow      time.Duration
}

func New(deps Dependencies) *gin.Engine {
	engine := gin.New()
	metrics := deps.Metrics
	if metrics == nil {
		metrics = middleware.NewMetrics()
	}
	maxRequests := deps.RateLimitMaxRequests
	if maxRequests <= 0 {
		maxRequests = 120
	}
	window := deps.RateLimitWindow
	if window <= 0 {
		window = time.Minute
	}
	globalRateLimit := middleware.RateLimitWithStore(deps.RateLimitStore, maxRequests, window, metrics)
	engine.Use(gin.Recovery(), middleware.StructuredLogger(metrics), func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/v1/admin/") {
			c.Next()
			return
		}
		globalRateLimit(c)
	})
	engine.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept, Origin, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "false")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		if cachepkg.ParseRealtimeFlag(c.Query("realtime")) {
			c.Request = c.Request.WithContext(cachepkg.WithBypass(c.Request.Context(), true))
			c.Header("Cache-Control", "no-store")
		}
		c.Next()
	})

	engine.GET("/health", deps.HealthHandler.Health)
	engine.GET("/metrics", metrics.Handler())
	engine.GET("/docs", deps.DocsHandler.SwaggerUI)
	engine.GET("/openapi.json", deps.DocsHandler.OpenAPI)

	v1 := engine.Group("/v1")
	{
		v1.GET("/health", deps.HealthHandler.Health)
		v1.POST("/auth/register", deps.AuthHandler.Register)
		v1.POST("/auth/login", deps.AuthHandler.Login)
		v1.POST("/auth/google", deps.AuthHandler.GoogleLogin)
		v1.POST("/auth/refresh", deps.AuthHandler.Refresh)
		v1.POST("/auth/logout", deps.AuthHandler.Logout)
		v1.POST("/auth/password/forgot", deps.AuthHandler.ForgotPassword)
		v1.POST("/auth/password/reset", deps.AuthHandler.ResetPassword)
		v1.DELETE("/me", deps.AuthMiddleware.Handle(), deps.AuthHandler.DeleteMe)
		v1.GET("/events", deps.AuthMiddleware.Handle(), deps.EventLogHandler.List)
		v1.GET("/events/verify", deps.AuthMiddleware.Handle(), deps.EventLogHandler.VerifyResource)
		v1.GET("/events/:eventId/verify", deps.AuthMiddleware.Handle(), deps.EventLogHandler.VerifyEvent)
		v1.GET("/shops", deps.ShopHandler.List)
		v1.GET("/shops/:shopId", deps.ShopHandler.GetByID)
		v1.GET("/shops/:shopId/pledges", deps.ShopHandler.ListPledges)
		v1.GET("/shops/:shopId/pledges/:pledgeId/integrity", deps.ShopHandler.GetPledgeIntegrity)
		v1.GET("/shops/:shopId/pledges/:pledgeId/proof", deps.ShopHandler.GetPledgeProof)
		v1.GET("/shops/:shopId/products", deps.ProductHandler.List)
		v1.GET("/shops/:shopId/products/:productId", deps.ProductHandler.GetByID)
		v1.GET("/shops/:shopId/products/:productId/batches", deps.ProductBatchHandler.List)
		v1.GET("/shops/:shopId/products/:productId/batches/:batchId", deps.ProductBatchHandler.GetByID)
		v1.GET("/shops/:shopId/products/:productId/freshness-reports", deps.ProductHandler.ListFreshnessReports)
		v1.GET("/shops/:shopId/reviews", deps.ShopHandler.ListReviews)
		v1.GET("/me", deps.AuthMiddleware.Handle(), deps.AuthHandler.Me)
		v1.PATCH("/me", deps.AuthMiddleware.Handle(), deps.AuthHandler.UpdateMe)
		v1.POST("/me/password", deps.AuthMiddleware.Handle(), deps.AuthHandler.ChangePassword)
		v1.POST("/shops", deps.AuthMiddleware.Handle(), deps.ShopHandler.Create)
		v1.POST("/shops/:shopId/products", deps.AuthMiddleware.Handle(), deps.ProductHandler.Create)
		v1.POST("/shops/:shopId/products/bulk", deps.AuthMiddleware.Handle(), deps.ProductHandler.BulkUpsert)
		v1.POST("/shops/:shopId/products/:productId/batches", deps.AuthMiddleware.Handle(), deps.ProductBatchHandler.Create)
		v1.PUT("/shops/:shopId/products/:productId/batches/:batchId", deps.AuthMiddleware.Handle(), deps.ProductBatchHandler.Update)
		v1.DELETE("/shops/:shopId/products/:productId/batches/:batchId", deps.AuthMiddleware.Handle(), deps.ProductBatchHandler.Delete)
		v1.POST("/shops/:shopId/products/:productId/freshness-reports", deps.AuthMiddleware.Handle(), deps.ProductHandler.CreateFreshnessReport)
		v1.PUT("/shops/:shopId", deps.AuthMiddleware.Handle(), deps.ShopHandler.Update)
		v1.PUT("/shops/:shopId/products/:productId", deps.AuthMiddleware.Handle(), deps.ProductHandler.Update)
		v1.DELETE("/shops/:shopId", deps.AuthMiddleware.Handle(), deps.ShopHandler.Delete)
		v1.DELETE("/shops/:shopId/products/:productId", deps.AuthMiddleware.Handle(), deps.ProductHandler.Delete)
		v1.POST("/shops/:shopId/reviews", deps.AuthMiddleware.Handle(), deps.ShopHandler.CreateReview)
		v1.DELETE("/shops/:shopId/reviews/me", deps.AuthMiddleware.Handle(), deps.ShopHandler.DeleteReview)
		v1.GET("/admin/shops", deps.AuthMiddleware.Handle(), deps.AdminMiddleware.Handle(), deps.ShopHandler.AdminList)
		v1.PATCH("/admin/shops/:shopId/moderation", deps.AuthMiddleware.Handle(), deps.AdminMiddleware.Handle(), deps.ShopHandler.Moderate)
		v1.POST("/admin/shops/:shopId/pledges/:pledgeId/reanchor", deps.AuthMiddleware.Handle(), deps.AdminMiddleware.Handle(), deps.ShopHandler.ReanchorPledgeIntegrity)
		v1.POST("/admin/shops/:shopId/pledges/:pledgeId/revoke", deps.AuthMiddleware.Handle(), deps.AdminMiddleware.Handle(), deps.ShopHandler.RevokePledgeIntegrity)
		v1.PATCH("/admin/products/:productId/moderation", deps.AuthMiddleware.Handle(), deps.AdminMiddleware.Handle(), deps.ProductHandler.Moderate)
		v1.PATCH("/admin/product-freshness-reports/:reportId/moderation", deps.AuthMiddleware.Handle(), deps.AdminMiddleware.Handle(), deps.ProductHandler.ModerateFreshnessReport)
		v1.PATCH("/admin/buyer-checks/:checkId/moderation", deps.AuthMiddleware.Handle(), deps.AdminMiddleware.Handle(), deps.BuyerHandler.Moderate)
		v1.GET("/admin/buyer-checks", deps.AuthMiddleware.Handle(), deps.AdminMiddleware.Handle(), deps.BuyerHandler.ListAdmin)
		v1.GET("/admin/product-freshness-reports", deps.AuthMiddleware.Handle(), deps.AdminMiddleware.Handle(), deps.ProductHandler.ListFreshnessReportsAdmin)
		v1.GET("/admin/users", deps.AuthMiddleware.Handle(), deps.AdminMiddleware.Handle(), deps.AdminUserHandler.List)
		v1.PATCH("/admin/users/:userId/role", deps.AuthMiddleware.Handle(), deps.AdminMiddleware.Handle(), deps.AdminUserHandler.UpdateRole)
		v1.PATCH("/admin/users/:userId/status", deps.AuthMiddleware.Handle(), deps.AdminMiddleware.Handle(), deps.AdminUserHandler.UpdateStatus)
		v1.POST("/admin/users/:userId/keys/rotate", deps.AuthMiddleware.Handle(), deps.AdminMiddleware.Handle(), deps.AdminUserHandler.RotateAccountKey)
		v1.POST("/admin/users/:userId/keys/recover", deps.AuthMiddleware.Handle(), deps.AdminMiddleware.Handle(), deps.AdminUserHandler.RecoverAccountKey)
		v1.POST("/admin/users/:userId/keys/backfill", deps.AuthMiddleware.Handle(), deps.AdminMiddleware.Handle(), deps.AdminUserHandler.BackfillAccountKey)
		v1.POST("/seller/score", deps.AuthMiddleware.Handle(), deps.SellerHandler.Score)
		v1.POST("/seller/commit", deps.AuthMiddleware.Handle(), deps.SellerHandler.Commit)
		v1.POST("/shops/:shopId/pledges/:pledgeId/bundle-token", deps.AuthMiddleware.Handle(), deps.SellerHandler.ReissueBundleToken)
		v1.POST("/buyer/check", deps.AuthMiddleware.Handle(), deps.BuyerHandler.Check)
		if deps.MediaHandler != nil {
			v1.POST("/media/images", deps.AuthMiddleware.Handle(), deps.MediaHandler.UploadImage)
		}
	}

	return engine
}
