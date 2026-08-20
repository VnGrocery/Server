package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type DocsHandler struct{}

func NewDocsHandler() *DocsHandler {
	return &DocsHandler{}
}

func (h *DocsHandler) OpenAPI(c *gin.Context) {
	c.JSON(http.StatusOK, buildOpenAPISpec())
}

func (h *DocsHandler) SwaggerUI(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerUIHTML))
}

func buildOpenAPISpec() gin.H {
	return gin.H{
		"openapi": "3.0.3",
		"info": gin.H{
			"title":       "VNGrocery API",
			"version":     "1.0.0",
			"description": "Backend API for authentication, trust and freshness flows, moderation, proof verification, and integrity anchoring.\n\nAPI song ngữ Việt - Anh để frontend, mobile, và ops team dễ đọc và tích hợp.",
		},
		"servers": []gin.H{
			{"url": "/"},
		},
		"components": gin.H{
			"securitySchemes": gin.H{
				"bearerAuth": gin.H{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT",
				},
			},
			"schemas": buildSchemas(),
		},
		"paths": buildPaths(),
	}
}

func buildSchemas() gin.H {
	schemas := gin.H{
		"RegisterRequest": gin.H{
			"type":     "object",
			"required": []string{"email", "password"},
			"properties": gin.H{
				"email":       gin.H{"type": "string", "format": "email"},
				"password":    gin.H{"type": "string"},
				"displayName": gin.H{"type": "string"},
				"firstName":   gin.H{"type": "string"},
				"lastName":    gin.H{"type": "string"},
			},
		},
		"LoginRequest": gin.H{
			"type":     "object",
			"required": []string{"email", "password"},
			"properties": gin.H{
				"email":    gin.H{"type": "string", "format": "email"},
				"password": gin.H{"type": "string"},
			},
		},
		"GoogleLoginRequest": gin.H{
			"type":     "object",
			"required": []string{"idToken"},
			"properties": gin.H{
				"idToken": gin.H{"type": "string"},
			},
		},
		"RefreshTokenRequest": gin.H{
			"type":     "object",
			"required": []string{"refreshToken"},
			"properties": gin.H{
				"refreshToken": gin.H{"type": "string"},
			},
		},
		"LogoutRequest": gin.H{
			"type":     "object",
			"required": []string{"refreshToken"},
			"properties": gin.H{
				"refreshToken": gin.H{"type": "string"},
			},
		},
		"ChangePasswordRequest": gin.H{
			"type":     "object",
			"required": []string{"currentPassword", "newPassword"},
			"properties": gin.H{
				"currentPassword": gin.H{"type": "string"},
				"newPassword":     gin.H{"type": "string"},
			},
		},
		"UpdateMeRequest": gin.H{
			"type":     "object",
			"required": []string{"expectedVersion"},
			"properties": gin.H{
				"expectedVersion": gin.H{"type": "integer", "minimum": 1},
				"displayName":     gin.H{"type": "string"},
				"firstName":       gin.H{"type": "string"},
				"lastName":        gin.H{"type": "string"},
			},
		},
		"ForgotPasswordRequest": gin.H{
			"type":     "object",
			"required": []string{"email"},
			"properties": gin.H{
				"email": gin.H{"type": "string", "format": "email"},
			},
		},
		"ResetPasswordRequest": gin.H{
			"type":     "object",
			"required": []string{"resetToken", "newPassword"},
			"properties": gin.H{
				"resetToken":  gin.H{"type": "string"},
				"newPassword": gin.H{"type": "string"},
			},
		},
		"AuthTokenResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"accessToken":  gin.H{"type": "string"},
				"refreshToken": gin.H{"type": "string"},
				"userId":       gin.H{"type": "string"},
				"email":        gin.H{"type": "string"},
				"role":         gin.H{"type": "string"},
				"displayName":  gin.H{"type": "string"},
				"firstName":    gin.H{"type": "string"},
				"lastName":     gin.H{"type": "string"},
				"publicKey":    gin.H{"type": "string"},
			},
		},
		"LogoutResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"status": gin.H{"type": "string"},
			},
		},
		"StatusResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"status": gin.H{"type": "string"},
			},
		},
		"PasswordResetResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"status":     gin.H{"type": "string"},
				"resetToken": gin.H{"type": "string"},
			},
		},
		"UpdateUserRoleRequest": gin.H{
			"type":     "object",
			"required": []string{"expectedVersion", "role"},
			"properties": gin.H{
				"expectedVersion": gin.H{"type": "integer", "minimum": 1},
				"role":            gin.H{"type": "string"},
			},
		},
		"UpdateUserStatusRequest": gin.H{
			"type":     "object",
			"required": []string{"expectedVersion", "status"},
			"properties": gin.H{
				"expectedVersion": gin.H{"type": "integer", "minimum": 1},
				"status":          gin.H{"type": "string"},
			},
		},
		"AccountKeyRequest": gin.H{
			"type":     "object",
			"required": []string{"expectedVersion"},
			"properties": gin.H{
				"expectedVersion": gin.H{"type": "integer", "minimum": 1},
			},
		},
		"AdminUserResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"userId":      gin.H{"type": "string"},
				"email":       gin.H{"type": "string"},
				"displayName": gin.H{"type": "string"},
				"role":        gin.H{"type": "string"},
				"status":      gin.H{"type": "string"},
				"version":     gin.H{"type": "integer"},
				"createdAt":   gin.H{"type": "string", "format": "date-time"},
				"updatedAt":   gin.H{"type": "string", "format": "date-time"},
			},
		},
		"AdminUserListResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"items": gin.H{
					"type":  "array",
					"items": gin.H{"$ref": "#/components/schemas/AdminUserResponse"},
				},
			},
		},
		"AccountKeyResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"userId":       gin.H{"type": "string"},
				"publicKey":    gin.H{"type": "string"},
				"keyAlgorithm": gin.H{"type": "string"},
				"vaultKeyPath": gin.H{"type": "string"},
				"version":      gin.H{"type": "integer"},
			},
		},
		"MeResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"userId":      gin.H{"type": "string"},
				"email":       gin.H{"type": "string"},
				"role":        gin.H{"type": "string"},
				"displayName": gin.H{"type": "string"},
				"firstName":   gin.H{"type": "string"},
				"lastName":    gin.H{"type": "string"},
				"version":     gin.H{"type": "integer"},
			},
		},
		"DeleteResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"userId": gin.H{"type": "string"},
				"status": gin.H{"type": "string"},
			},
		},
		"UpsertShopRequest": gin.H{
			"type":     "object",
			"required": []string{"name", "address", "latitude", "longitude"},
			"properties": gin.H{
				"expectedVersion": gin.H{"type": "integer", "minimum": 1},
				"name":            gin.H{"type": "string"},
				"description":     gin.H{"type": "string"},
				"address":         gin.H{"type": "string"},
				"latitude":        gin.H{"type": "number"},
				"longitude":       gin.H{"type": "number"},
			},
		},
		"ModerateShopRequest": gin.H{
			"type":     "object",
			"required": []string{"expectedVersion", "status"},
			"properties": gin.H{
				"expectedVersion": gin.H{"type": "integer", "minimum": 1},
				"status":          gin.H{"type": "string"},
				"moderationNote":  gin.H{"type": "string"},
			},
		},
		"CreateShopReviewRequest": gin.H{
			"type":     "object",
			"required": []string{"rating"},
			"properties": gin.H{
				"rating":  gin.H{"type": "integer", "minimum": 1, "maximum": 5},
				"comment": gin.H{"type": "string"},
			},
		},
		"ShopTrustSummaryResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"hasPledges":         gin.H{"type": "boolean"},
				"pledgeCount":        gin.H{"type": "integer"},
				"latestPledgeId":     gin.H{"type": "string"},
				"latestPledgeStatus": gin.H{"type": "string"},
				"latestScore":        gin.H{"type": "number"},
				"latestCategory":     gin.H{"type": "string"},
				"latestConfidence":   gin.H{"type": "number"},
				"lastCommittedAt":    gin.H{"type": "string", "format": "date-time"},
				"score":              gin.H{"type": "number"},
				"grade":              gin.H{"type": "string"},
				"formulaVersion":     gin.H{"type": "string"},
				"pledgeScore":        gin.H{"type": "number"},
				"reviewScore":        gin.H{"type": "number"},
				"buyerCheckScore":    gin.H{"type": "number"},
				"consistencyScore":   gin.H{"type": "number"},
				"recencyScore":       gin.H{"type": "number"},
				"coverageScore":      gin.H{"type": "number"},
				"buyerCheckCount":    gin.H{"type": "integer"},
				"trustedCheckCount":  gin.H{"type": "integer"},
				"highRiskCheckCount": gin.H{"type": "integer"},
				"reasons": gin.H{
					"type":  "array",
					"items": gin.H{"type": "string"},
				},
			},
		},
		"ShopRatingSummaryResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"ratingCount":   gin.H{"type": "integer"},
				"averageRating": gin.H{"type": "number"},
			},
		},
		"ShopResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"shopId":            gin.H{"type": "string"},
				"ownerUserId":       gin.H{"type": "string"},
				"name":              gin.H{"type": "string"},
				"description":       gin.H{"type": "string"},
				"address":           gin.H{"type": "string"},
				"latitude":          gin.H{"type": "number"},
				"longitude":         gin.H{"type": "number"},
				"status":            gin.H{"type": "string"},
				"dataHash":          gin.H{"type": "string"},
				"chainTxHash":       gin.H{"type": "string"},
				"chainBlockNumber":  gin.H{"type": "integer"},
				"chainAnchorStatus": gin.H{"type": "string"},
				"chainAnchorTime":   gin.H{"type": "string", "format": "date-time"},
				"integrityStatus":   gin.H{"type": "string"},
				"moderatedByUserId": gin.H{"type": "string"},
				"moderationNote":    gin.H{"type": "string"},
				"moderatedAt":       gin.H{"type": "string", "format": "date-time"},
				"trustSummary": gin.H{
					"$ref": "#/components/schemas/ShopTrustSummaryResponse",
				},
				"ratingSummary": gin.H{
					"$ref": "#/components/schemas/ShopRatingSummaryResponse",
				},
				"createdAt": gin.H{"type": "string", "format": "date-time"},
				"updatedAt": gin.H{"type": "string", "format": "date-time"},
			},
		},
		"ShopListResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"items": gin.H{
					"type":  "array",
					"items": gin.H{"$ref": "#/components/schemas/ShopResponse"},
				},
				"page":     gin.H{"type": "integer"},
				"pageSize": gin.H{"type": "integer"},
				"total":    gin.H{"type": "integer"},
				"hasNext":  gin.H{"type": "boolean"},
			},
		},
		"ShopReviewResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"reviewId":       gin.H{"type": "string"},
				"shopId":         gin.H{"type": "string"},
				"reviewerUserId": gin.H{"type": "string"},
				"reviewerName":   gin.H{"type": "string", "description": "Display name of the reviewer; empty when the account has none."},
				"rating":         gin.H{"type": "integer"},
				"comment":        gin.H{"type": "string"},
				"status":         gin.H{"type": "string"},
				"createdAt":      gin.H{"type": "string", "format": "date-time"},
				"updatedAt":      gin.H{"type": "string", "format": "date-time"},
			},
		},
		"UpsertProductRequest": gin.H{
			"type":     "object",
			"required": []string{"name", "price", "currency"},
			"properties": gin.H{
				"expectedVersion": gin.H{"type": "integer", "minimum": 1},
				"name":            gin.H{"type": "string"},
				"description":     gin.H{"type": "string"},
				"category":        gin.H{"type": "string"},
				"tags": gin.H{
					"type":  "array",
					"items": gin.H{"type": "string"},
				},
				"imageUrls": gin.H{
					"type":  "array",
					"items": gin.H{"type": "string"},
				},
				"freshnessNote":  gin.H{"type": "string"},
				"freshnessScore": gin.H{"type": "number"},
				"price":          gin.H{"type": "number"},
				"currency":       gin.H{"type": "string"},
			},
		},
		"ProductResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"productId":   gin.H{"type": "string"},
				"shopId":      gin.H{"type": "string"},
				"ownerUserId": gin.H{"type": "string"},
				"name":        gin.H{"type": "string"},
				"description": gin.H{"type": "string"},
				"category":    gin.H{"type": "string"},
				"tags": gin.H{
					"type":  "array",
					"items": gin.H{"type": "string"},
				},
				"imageUrls": gin.H{
					"type":  "array",
					"items": gin.H{"type": "string"},
				},
				"freshnessNote":  gin.H{"type": "string"},
				"freshnessScore": gin.H{"type": "number"},
				"price":          gin.H{"type": "number"},
				"currency":       gin.H{"type": "string"},
				"status":         gin.H{"type": "string"},
				"version":        gin.H{"type": "integer"},
				"createdAt":      gin.H{"type": "string", "format": "date-time"},
				"updatedAt":      gin.H{"type": "string", "format": "date-time"},
			},
		},
		"ProductListResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"items": gin.H{
					"type":  "array",
					"items": gin.H{"$ref": "#/components/schemas/ProductResponse"},
				},
			},
		},
		"CreateProductFreshnessReportRequest": gin.H{
			"type":     "object",
			"required": []string{"score", "category", "confidence", "imageHash"},
			"properties": gin.H{
				"score":      gin.H{"type": "number"},
				"category":   gin.H{"type": "string"},
				"confidence": gin.H{"type": "number"},
				"comment":    gin.H{"type": "string"},
				"imageHash":  gin.H{"type": "string"},
				"imageCid":   gin.H{"type": "string"},
			},
		},
		"ProductFreshnessReportResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"reportId":          gin.H{"type": "string"},
				"productId":         gin.H{"type": "string"},
				"shopId":            gin.H{"type": "string"},
				"reporterUserId":    gin.H{"type": "string"},
				"status":            gin.H{"type": "string"},
				"version":           gin.H{"type": "integer"},
				"score":             gin.H{"type": "number"},
				"category":          gin.H{"type": "string"},
				"confidence":        gin.H{"type": "number"},
				"comment":           gin.H{"type": "string"},
				"imageHash":         gin.H{"type": "string"},
				"imageCid":          gin.H{"type": "string"},
				"moderatedByUserId": gin.H{"type": "string"},
				"moderationNote":    gin.H{"type": "string"},
				"moderatedAt":       gin.H{"type": "string", "format": "date-time"},
				"createdAt":         gin.H{"type": "string", "format": "date-time"},
				"updatedAt":         gin.H{"type": "string", "format": "date-time"},
			},
		},
		"ModerateProductFreshnessReportRequest": gin.H{
			"type":     "object",
			"required": []string{"expectedVersion", "status"},
			"properties": gin.H{
				"expectedVersion": gin.H{"type": "integer"},
				"status":          gin.H{"type": "string"},
				"moderationNote":  gin.H{"type": "string"},
			},
		},
		"ProductFreshnessReportListResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"items": gin.H{
					"type":  "array",
					"items": gin.H{"$ref": "#/components/schemas/ProductFreshnessReportResponse"},
				},
				"pagination": gin.H{"$ref": "#/components/schemas/PaginationResponse"},
			},
		},
		"SellerCommitRequest": gin.H{
			"type":     "object",
			"required": []string{"shopId", "bundleId", "score", "category", "confidence", "imageHash"},
			"properties": gin.H{
				"shopId":     gin.H{"type": "string"},
				"productId":  gin.H{"type": "string"},
				"bundleId":   gin.H{"type": "string"},
				"score":      gin.H{"type": "number"},
				"category":   gin.H{"type": "string"},
				"confidence": gin.H{"type": "number"},
				"imageHash":  gin.H{"type": "string"},
				"imageCid":   gin.H{"type": "string"},
			},
		},
		"SellerCommitResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"pledgeId":             gin.H{"type": "string"},
				"shopId":               gin.H{"type": "string"},
				"productId":            gin.H{"type": "string"},
				"bundleId":             gin.H{"type": "string"},
				"createdByUserId":      gin.H{"type": "string"},
				"status":               gin.H{"type": "string"},
				"score":                gin.H{"type": "number"},
				"category":             gin.H{"type": "string"},
				"confidence":           gin.H{"type": "number"},
				"imageHash":            gin.H{"type": "string"},
				"imageCid":             gin.H{"type": "string"},
				"dataHash":             gin.H{"type": "string"},
				"chainTxHash":          gin.H{"type": "string"},
				"chainBlockNumber":     gin.H{"type": "integer"},
				"chainAnchorStatus":    gin.H{"type": "string"},
				"chainAnchorTime":      gin.H{"type": "string", "format": "date-time"},
				"integrityStatus":      gin.H{"type": "string"},
				"qrVersion":            gin.H{"type": "string"},
				"bundleToken":          gin.H{"type": "string"},
				"bundleTokenExpiresAt": gin.H{"type": "string", "format": "date-time"},
				"committedAt":          gin.H{"type": "string", "format": "date-time"},
				"createdAt":            gin.H{"type": "string", "format": "date-time"},
				"updatedAt":            gin.H{"type": "string", "format": "date-time"},
			},
		},
		"BundleTokenResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"qrVersion":            gin.H{"type": "string"},
				"bundleToken":          gin.H{"type": "string"},
				"bundleTokenExpiresAt": gin.H{"type": "string", "format": "date-time"},
			},
		},
		"PledgeResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"pledgeId":          gin.H{"type": "string"},
				"shopId":            gin.H{"type": "string"},
				"productId":         gin.H{"type": "string"},
				"bundleId":          gin.H{"type": "string"},
				"createdByUserId":   gin.H{"type": "string"},
				"status":            gin.H{"type": "string"},
				"version":           gin.H{"type": "integer"},
				"score":             gin.H{"type": "number"},
				"category":          gin.H{"type": "string"},
				"confidence":        gin.H{"type": "number"},
				"imageHash":         gin.H{"type": "string"},
				"imageCid":          gin.H{"type": "string"},
				"dataHash":          gin.H{"type": "string"},
				"chainTxHash":       gin.H{"type": "string"},
				"chainBlockNumber":  gin.H{"type": "integer"},
				"chainAnchorStatus": gin.H{"type": "string"},
				"chainAnchorTime":   gin.H{"type": "string", "format": "date-time"},
				"integrityStatus":   gin.H{"type": "string"},
				"committedAt":       gin.H{"type": "string", "format": "date-time"},
				"createdAt":         gin.H{"type": "string", "format": "date-time"},
				"updatedAt":         gin.H{"type": "string", "format": "date-time"},
			},
		},
		"PledgeIntegrityResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"pledgeId":          gin.H{"type": "string"},
				"shopId":            gin.H{"type": "string"},
				"dataHash":          gin.H{"type": "string"},
				"providedDataHash":  gin.H{"type": "string", "description": "Optional hash from client, compared against the current pledge dataHash stored in DB."},
				"chainTxHash":       gin.H{"type": "string"},
				"chainBlockNumber":  gin.H{"type": "integer"},
				"chainAnchorStatus": gin.H{"type": "string"},
				"chainAnchorTime":   gin.H{"type": "string", "format": "date-time"},
				"integrityStatus":   gin.H{"type": "string"},
				"onChainMatch":      gin.H{"type": "boolean"},
				"providedHashMatch": gin.H{"type": "boolean"},
				"onChainDataHash":   gin.H{"type": "string"},
				"onChainVersion":    gin.H{"type": "integer"},
				"onChainTimestamp":  gin.H{"type": "string", "format": "date-time"},
				"onChainPresent":    gin.H{"type": "boolean"},
				"mismatchReason":    gin.H{"type": "string"},
				"lastCheckedAt":     gin.H{"type": "string", "format": "date-time"},
				"canReanchor":       gin.H{"type": "boolean"},
				"canRevoke":         gin.H{"type": "boolean"},
			},
		},
		"PledgeProofBundleResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"pledgeId":           gin.H{"type": "string"},
				"shopId":             gin.H{"type": "string"},
				"productId":          gin.H{"type": "string"},
				"bundleId":           gin.H{"type": "string"},
				"score":              gin.H{"type": "number"},
				"category":           gin.H{"type": "string"},
				"confidence":         gin.H{"type": "number"},
				"committedAt":        gin.H{"type": "string", "format": "date-time"},
				"imageHash":          gin.H{"type": "string"},
				"imageCid":           gin.H{"type": "string"},
				"proofStatus":        gin.H{"type": "string"},
				"proofHeadline":      gin.H{"type": "string"},
				"proofSummary":       gin.H{"type": "string"},
				"recommendedActions": gin.H{"type": "array", "items": gin.H{"type": "string"}},
				"integrity":          gin.H{"$ref": "#/components/schemas/PledgeIntegrityResponse"},
			},
		},
		"MediaImageUploadResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"imageHash":   gin.H{"type": "string"},
				"imageCid":    gin.H{"type": "string"},
				"gatewayUrl":  gin.H{"type": "string"},
				"contentType": gin.H{"type": "string"},
				"sizeBytes":   gin.H{"type": "integer"},
			},
		},
		"ModeratePledgeIntegrityRequest": gin.H{
			"type":     "object",
			"required": []string{"expectedVersion"},
			"properties": gin.H{
				"expectedVersion": gin.H{"type": "integer"},
			},
		},
		"PledgeHistoryResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"items": gin.H{
					"type":  "array",
					"items": gin.H{"$ref": "#/components/schemas/PledgeResponse"},
				},
			},
		},
		"SellerScoreResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"score":      gin.H{"type": "number"},
				"category":   gin.H{"type": "string"},
				"confidence": gin.H{"type": "number"},
				"imageHash":  gin.H{"type": "string"},
				"imageCid":   gin.H{"type": "string"},
			},
		},
		"BuyerCheckResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"policyVersion":    gin.H{"type": "string"},
				"checkId":          gin.H{"type": "string"},
				"shopId":           gin.H{"type": "string"},
				"productId":        gin.H{"type": "string"},
				"bundleId":         gin.H{"type": "string"},
				"buyerUserId":      gin.H{"type": "string"},
				"status":           gin.H{"type": "string"},
				"version":          gin.H{"type": "integer"},
				"hasPledge":        gin.H{"type": "boolean"},
				"pledgeId":         gin.H{"type": "string"},
				"trusted":          gin.H{"type": "boolean"},
				"verdict":          gin.H{"type": "string"},
				"pledgedScore":     gin.H{"type": "number"},
				"actualScore":      gin.H{"type": "number"},
				"scoreDelta":       gin.H{"type": "number"},
				"scoreDeltaAbs":    gin.H{"type": "number"},
				"pledgedCategory":  gin.H{"type": "string"},
				"actualCategory":   gin.H{"type": "string"},
				"actualConfidence": gin.H{"type": "number"},
				"locationStatus":   gin.H{"type": "string"},
				"categoryMatch":    gin.H{"type": "boolean"},
				"imageHash":        gin.H{"type": "string"},
				"imageCid":         gin.H{"type": "string"},
				"reasons": gin.H{
					"type":  "array",
					"items": gin.H{"type": "string"},
				},
				"moderatedByUserId": gin.H{"type": "string"},
				"moderationNote":    gin.H{"type": "string"},
				"moderatedAt":       gin.H{"type": "string", "format": "date-time"},
				"createdAt":         gin.H{"type": "string", "format": "date-time"},
				"updatedAt":         gin.H{"type": "string", "format": "date-time"},
			},
		},
		"BuyerCheckListResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"items": gin.H{
					"type":  "array",
					"items": gin.H{"$ref": "#/components/schemas/BuyerCheckResponse"},
				},
				"pagination": gin.H{"$ref": "#/components/schemas/PaginationResponse"},
			},
		},
		"ModerateBuyerCheckRequest": gin.H{
			"type":     "object",
			"required": []string{"expectedVersion", "status"},
			"properties": gin.H{
				"expectedVersion": gin.H{"type": "integer"},
				"status":          gin.H{"type": "string"},
				"moderationNote":  gin.H{"type": "string"},
			},
		},
		"HealthResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"status": gin.H{"type": "string"},
			},
		},
		"EventLogResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"eventId":         gin.H{"type": "string"},
				"actorUserId":     gin.H{"type": "string"},
				"resourceType":    gin.H{"type": "string"},
				"resourceId":      gin.H{"type": "string"},
				"resourceVersion": gin.H{"type": "integer"},
				"action":          gin.H{"type": "string"},
				"status":          gin.H{"type": "string"},
				"sequence":        gin.H{"type": "integer"},
				"previousEventId": gin.H{"type": "string"},
				"payloadJson":     gin.H{"type": "string"},
				"publicKey":       gin.H{"type": "string"},
				"keyAlgorithm":    gin.H{"type": "string"},
				"signature":       gin.H{"type": "string"},
				"contentSha256":   gin.H{"type": "string"},
				"createdAt":       gin.H{"type": "string", "format": "date-time"},
			},
		},
		"PaginationResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"page":       gin.H{"type": "integer"},
				"pageSize":   gin.H{"type": "integer"},
				"totalItems": gin.H{"type": "integer"},
				"totalPages": gin.H{"type": "integer"},
			},
		},
		"EventLogListResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"items": gin.H{
					"type":  "array",
					"items": gin.H{"$ref": "#/components/schemas/EventLogResponse"},
				},
				"pagination": gin.H{"$ref": "#/components/schemas/PaginationResponse"},
			},
		},
		"EventVerificationResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"eventId":              gin.H{"type": "string"},
				"resourceType":         gin.H{"type": "string"},
				"resourceId":           gin.H{"type": "string"},
				"sequence":             gin.H{"type": "integer"},
				"previousEventId":      gin.H{"type": "string"},
				"contentHashValid":     gin.H{"type": "boolean"},
				"signatureValid":       gin.H{"type": "boolean"},
				"chainLinkValid":       gin.H{"type": "boolean"},
				"previousEventPresent": gin.H{"type": "boolean"},
				"verified":             gin.H{"type": "boolean"},
			},
		},
		"ResourceEventVerificationResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"resourceType": gin.H{"type": "string"},
				"resourceId":   gin.H{"type": "string"},
				"eventCount":   gin.H{"type": "integer"},
				"verified":     gin.H{"type": "boolean"},
				"events": gin.H{
					"type":  "array",
					"items": gin.H{"$ref": "#/components/schemas/EventVerificationResponse"},
				},
			},
		},
		"ErrorResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"error": gin.H{"type": "string"},
			},
		},
	}
	schemas["VoucherResponse"] = gin.H{"type": "object", "properties": gin.H{"voucherId": gin.H{"type": "string"}, "shopId": gin.H{"type": "string"}, "code": gin.H{"type": "string"}, "title": gin.H{"type": "string"}, "discountValue": gin.H{"type": "integer"}, "isPercent": gin.H{"type": "boolean"}, "minSpend": gin.H{"type": "integer"}, "expiresAt": gin.H{"type": "string", "format": "date-time"}, "active": gin.H{"type": "boolean"}, "manual": gin.H{"type": "boolean"}}}
	schemas["VoucherCheckRequest"] = gin.H{"type": "object", "required": []string{"code", "shopId", "orderValue"}, "properties": gin.H{"code": gin.H{"type": "string"}, "shopId": gin.H{"type": "string"}, "orderValue": gin.H{"type": "integer"}}}
	schemas["VoucherCheckResponse"] = gin.H{"type": "object", "properties": gin.H{"voucher": gin.H{"$ref": "#/components/schemas/VoucherResponse"}, "valid": gin.H{"type": "boolean"}, "message": gin.H{"type": "string"}, "discountAmount": gin.H{"type": "integer"}, "finalPrice": gin.H{"type": "integer"}}}
	schemas["UserVoucherResponse"] = gin.H{"type": "object", "properties": gin.H{"userVoucherId": gin.H{"type": "string"}, "voucherId": gin.H{"type": "string"}, "used": gin.H{"type": "boolean"}, "usedAt": gin.H{"type": "string", "format": "date-time"}, "voucher": gin.H{"$ref": "#/components/schemas/VoucherResponse"}}}

	return schemas
}

func buildPaths() gin.H {
	jsonBody := func(schema string) gin.H {
		return gin.H{
			"required": true,
			"content": gin.H{
				"application/json": gin.H{
					"schema": gin.H{"$ref": "#/components/schemas/" + schema},
				},
			},
		}
	}
	success := func(code int, schema string) gin.H {
		return gin.H{
			http.StatusText(code): gin.H{
				"description": http.StatusText(code),
				"content": gin.H{
					"application/json": gin.H{
						"schema": gin.H{"$ref": "#/components/schemas/" + schema},
					},
				},
			},
		}
	}
	errorResponse := gin.H{
		"default": gin.H{
			"description": "Error",
			"content": gin.H{
				"application/json": gin.H{
					"schema": gin.H{"$ref": "#/components/schemas/ErrorResponse"},
				},
			},
		},
	}

	paths := gin.H{
		"/health": gin.H{
			"get": gin.H{
				"summary":   "Health check",
				"responses": success(http.StatusOK, "HealthResponse"),
			},
		},
		"/v1/health": gin.H{
			"get": gin.H{
				"summary":   "Versioned health check",
				"responses": success(http.StatusOK, "HealthResponse"),
			},
		},
		"/v1/auth/register": gin.H{
			"post": gin.H{
				"summary":     "Register account",
				"requestBody": jsonBody("RegisterRequest"),
				"responses":   mergeResponses(success(http.StatusCreated, "AuthTokenResponse"), errorResponse),
			},
		},
		"/v1/auth/login": gin.H{
			"post": gin.H{
				"summary":     "Login with email and password",
				"requestBody": jsonBody("LoginRequest"),
				"responses":   mergeResponses(success(http.StatusOK, "AuthTokenResponse"), errorResponse),
			},
		},
		"/v1/auth/google": gin.H{
			"post": gin.H{
				"summary":     "Login with Google ID token",
				"requestBody": jsonBody("GoogleLoginRequest"),
				"responses":   mergeResponses(success(http.StatusOK, "AuthTokenResponse"), errorResponse),
			},
		},
		"/v1/auth/refresh": gin.H{
			"post": gin.H{
				"summary":     "Refresh access token",
				"requestBody": jsonBody("RefreshTokenRequest"),
				"responses":   mergeResponses(success(http.StatusOK, "AuthTokenResponse"), errorResponse),
			},
		},
		"/v1/auth/logout": gin.H{
			"post": gin.H{
				"summary":     "Logout by revoking refresh token",
				"requestBody": jsonBody("LogoutRequest"),
				"responses":   mergeResponses(success(http.StatusOK, "LogoutResponse"), errorResponse),
			},
		},
		"/v1/auth/password/forgot": gin.H{
			"post": gin.H{
				"summary":     "Request password reset",
				"requestBody": jsonBody("ForgotPasswordRequest"),
				"responses":   mergeResponses(success(http.StatusOK, "PasswordResetResponse"), errorResponse),
			},
		},
		"/v1/auth/password/reset": gin.H{
			"post": gin.H{
				"summary":     "Reset password with reset token",
				"requestBody": jsonBody("ResetPasswordRequest"),
				"responses":   mergeResponses(success(http.StatusOK, "StatusResponse"), errorResponse),
			},
		},
		"/v1/me": gin.H{
			"get": gin.H{
				"summary":   "Get current user",
				"security":  []gin.H{{"bearerAuth": []string{}}},
				"responses": mergeResponses(success(http.StatusOK, "MeResponse"), errorResponse),
			},
			"patch": gin.H{
				"summary":     "Update current user profile",
				"security":    []gin.H{{"bearerAuth": []string{}}},
				"requestBody": jsonBody("UpdateMeRequest"),
				"responses": mergeResponses(
					mergeResponses(success(http.StatusOK, "MeResponse"), gin.H{"409": gin.H{"description": "Version conflict"}}),
					errorResponse,
				),
			},
			"delete": gin.H{
				"summary":  "Soft delete current user account",
				"security": []gin.H{{"bearerAuth": []string{}}},
				"parameters": []gin.H{
					requiredQueryParam("expectedVersion", "integer"),
				},
				"responses": mergeResponses(success(http.StatusOK, "DeleteResponse"), errorResponse),
			},
		},
		"/v1/me/password": gin.H{
			"post": gin.H{
				"summary":     "Change current user password",
				"security":    []gin.H{{"bearerAuth": []string{}}},
				"requestBody": jsonBody("ChangePasswordRequest"),
				"responses":   mergeResponses(success(http.StatusOK, "StatusResponse"), errorResponse),
			},
		},
		"/v1/events": gin.H{
			"get": gin.H{
				"summary":  "List signed event logs",
				"security": []gin.H{{"bearerAuth": []string{}}},
				"parameters": []gin.H{
					queryParam("resourceType", "string"),
					queryParam("resourceId", "string"),
					queryParam("actorUserId", "string"),
					queryParam("action", "string"),
					queryParam("status", "string"),
					queryParam("minSequence", "integer"),
					queryParam("maxSequence", "integer"),
					queryParam("createdAfter", "string"),
					queryParam("createdBefore", "string"),
					queryParam("page", "integer"),
					queryParam("pageSize", "integer"),
					realtimeQueryParam(),
				},
				"responses": mergeResponses(success(http.StatusOK, "EventLogListResponse"), errorResponse),
			},
		},
		"/v1/events/verify": gin.H{
			"get": gin.H{
				"summary":  "Verify event chain for a resource",
				"security": []gin.H{{"bearerAuth": []string{}}},
				"parameters": []gin.H{
					queryParam("resourceType", "string"),
					queryParam("resourceId", "string"),
					realtimeQueryParam(),
				},
				"responses": mergeResponses(success(http.StatusOK, "ResourceEventVerificationResponse"), errorResponse),
			},
		},
		"/v1/events/{eventId}/verify": gin.H{
			"get": gin.H{
				"summary":  "Verify event signature and chain link",
				"security": []gin.H{{"bearerAuth": []string{}}},
				"parameters": []gin.H{
					pathParam("eventId"),
					realtimeQueryParam(),
				},
				"responses": mergeResponses(success(http.StatusOK, "EventVerificationResponse"), errorResponse),
			},
		},
		"/v1/shops": gin.H{
			"get": gin.H{
				"summary": "List public shops",
				"parameters": []gin.H{
					queryParam("page", "integer"),
					queryParam("pageSize", "integer"),
					queryParam("q", "string"),
					queryParam("ownerUserId", "string"),
					queryParam("status", "string"),
					realtimeQueryParam(),
				},
				"responses": mergeResponses(success(http.StatusOK, "ShopListResponse"), errorResponse),
			},
			"post": gin.H{
				"summary":     "Create shop",
				"security":    []gin.H{{"bearerAuth": []string{}}},
				"requestBody": jsonBody("UpsertShopRequest"),
				"description": "Mỗi tài khoản chỉ được phép tạo tối đa 1 cửa hàng. Trả 409 nếu đã có cửa hàng active/pending.",
				"responses":   mergeResponses(mergeResponses(success(http.StatusCreated, "ShopResponse"), gin.H{"409": gin.H{"description": "Account already owns a shop"}}), errorResponse),
			},
		},
		"/v1/shops/{shopId}": gin.H{
			"get": gin.H{
				"summary":    "Get shop detail",
				"parameters": []gin.H{pathParam("shopId"), realtimeQueryParam()},
				"responses":  mergeResponses(success(http.StatusOK, "ShopResponse"), errorResponse),
			},
			"put": gin.H{
				"summary":     "Update shop",
				"security":    []gin.H{{"bearerAuth": []string{}}},
				"parameters":  []gin.H{pathParam("shopId")},
				"requestBody": jsonBody("UpsertShopRequest"),
				"responses":   mergeResponses(success(http.StatusOK, "ShopResponse"), errorResponse),
			},
			"delete": gin.H{
				"summary":  "Soft delete shop",
				"security": []gin.H{{"bearerAuth": []string{}}},
				"parameters": []gin.H{
					pathParam("shopId"),
					requiredQueryParam("expectedVersion", "integer"),
				},
				"responses": mergeResponses(success(http.StatusOK, "ShopResponse"), errorResponse),
			},
		},
		"/v1/shops/{shopId}/pledges": gin.H{
			"get": gin.H{
				"summary": "List seller pledge history for buyer UI",
				"parameters": []gin.H{
					pathParam("shopId"),
					queryParam("productId", "string"),
					queryParam("category", "string"),
					realtimeQueryParam(),
				},
				"responses": mergeResponses(success(http.StatusOK, "PledgeHistoryResponse"), errorResponse),
			},
		},
		"/v1/shops/{shopId}/pledges/{pledgeId}/integrity": gin.H{
			"get": gin.H{
				"summary": "Get pledge integrity anchor status",
				"parameters": []gin.H{
					pathParam("shopId"),
					pathParam("pledgeId"),
					queryParam("dataHash", "string"),
					realtimeQueryParam(),
				},
				"responses": mergeResponses(success(http.StatusOK, "PledgeIntegrityResponse"), errorResponse),
			},
		},
		"/v1/shops/{shopId}/pledges/{pledgeId}/proof": gin.H{
			"get": gin.H{
				"summary": "Get buyer-friendly pledge proof bundle",
				"parameters": []gin.H{
					pathParam("shopId"),
					pathParam("pledgeId"),
					queryParam("dataHash", "string"),
					realtimeQueryParam(),
				},
				"responses": mergeResponses(success(http.StatusOK, "PledgeProofBundleResponse"), errorResponse),
			},
		},
		"/v1/media/images": gin.H{
			"post": gin.H{
				"summary":  "Upload image and return reusable media metadata",
				"security": []gin.H{{"bearerAuth": []string{}}},
				"requestBody": gin.H{
					"required": true,
					"content": gin.H{
						"multipart/form-data": gin.H{
							"schema": gin.H{
								"type":     "object",
								"required": []string{"image"},
								"properties": gin.H{
									"image": gin.H{"type": "string", "format": "binary"},
								},
							},
						},
					},
				},
				"responses": mergeResponses(success(http.StatusCreated, "MediaImageUploadResponse"), errorResponse),
			},
		},
		"/v1/shops/{shopId}/products": gin.H{
			"get": gin.H{
				"summary": "List shop products",
				"parameters": []gin.H{
					pathParam("shopId"),
					queryParam("q", "string"),
					queryParam("category", "string"),
					queryParam("tag", "string"),
					queryParam("sort", "string"),
					realtimeQueryParam(),
				},
				"responses": mergeResponses(success(http.StatusOK, "ProductListResponse"), errorResponse),
			},
			"post": gin.H{
				"summary":     "Create product",
				"security":    []gin.H{{"bearerAuth": []string{}}},
				"parameters":  []gin.H{pathParam("shopId")},
				"requestBody": jsonBody("UpsertProductRequest"),
				"responses":   mergeResponses(success(http.StatusCreated, "ProductResponse"), errorResponse),
			},
		},
		"/v1/shops/{shopId}/products/{productId}": gin.H{
			"get": gin.H{
				"summary": "Get product detail",
				"parameters": []gin.H{
					pathParam("shopId"),
					pathParam("productId"),
					realtimeQueryParam(),
				},
				"responses": mergeResponses(success(http.StatusOK, "ProductResponse"), errorResponse),
			},
			"put": gin.H{
				"summary":  "Update product",
				"security": []gin.H{{"bearerAuth": []string{}}},
				"parameters": []gin.H{
					pathParam("shopId"),
					pathParam("productId"),
				},
				"requestBody": jsonBody("UpsertProductRequest"),
				"responses":   mergeResponses(success(http.StatusOK, "ProductResponse"), errorResponse),
			},
			"delete": gin.H{
				"summary":  "Soft delete product",
				"security": []gin.H{{"bearerAuth": []string{}}},
				"parameters": []gin.H{
					pathParam("shopId"),
					pathParam("productId"),
					requiredQueryParam("expectedVersion", "integer"),
				},
				"responses": mergeResponses(success(http.StatusOK, "ProductResponse"), errorResponse),
			},
		},
		"/v1/shops/{shopId}/products/{productId}/freshness-reports": gin.H{
			"get": gin.H{
				"summary": "List product freshness reports",
				"parameters": []gin.H{
					pathParam("shopId"),
					pathParam("productId"),
					realtimeQueryParam(),
				},
				"responses": mergeResponses(success(http.StatusOK, "ProductFreshnessReportListResponse"), errorResponse),
			},
			"post": gin.H{
				"summary":  "Create product freshness report",
				"security": []gin.H{{"bearerAuth": []string{}}},
				"parameters": []gin.H{
					pathParam("shopId"),
					pathParam("productId"),
				},
				"requestBody": jsonBody("CreateProductFreshnessReportRequest"),
				"responses":   mergeResponses(success(http.StatusCreated, "ProductFreshnessReportResponse"), errorResponse),
			},
		},
		"/v1/admin/shops/{shopId}/pledges/{pledgeId}/reanchor": gin.H{
			"post": gin.H{
				"summary":     "Re-anchor pledge integrity proof",
				"security":    []gin.H{{"bearerAuth": []string{}}},
				"parameters":  []gin.H{pathParam("shopId"), pathParam("pledgeId")},
				"requestBody": jsonBody("ModeratePledgeIntegrityRequest"),
				"responses":   mergeResponses(success(http.StatusOK, "PledgeResponse"), errorResponse),
			},
		},
		"/v1/admin/shops/{shopId}/pledges/{pledgeId}/revoke": gin.H{
			"post": gin.H{
				"summary":     "Revoke pledge integrity proof",
				"security":    []gin.H{{"bearerAuth": []string{}}},
				"parameters":  []gin.H{pathParam("shopId"), pathParam("pledgeId")},
				"requestBody": jsonBody("ModeratePledgeIntegrityRequest"),
				"responses":   mergeResponses(success(http.StatusOK, "PledgeResponse"), errorResponse),
			},
		},
		"/v1/shops/{shopId}/reviews": gin.H{
			"get": gin.H{
				"summary":    "List shop reviews",
				"parameters": []gin.H{pathParam("shopId"), realtimeQueryParam()},
				"responses": gin.H{
					"200": gin.H{
						"description": "OK",
						"content": gin.H{
							"application/json": gin.H{
								"schema": gin.H{
									"type":  "array",
									"items": gin.H{"$ref": "#/components/schemas/ShopReviewResponse"},
								},
							},
						},
					},
					"default": errorResponse["default"],
				},
			},
			"post": gin.H{
				"summary":     "Create or update current user review",
				"security":    []gin.H{{"bearerAuth": []string{}}},
				"parameters":  []gin.H{pathParam("shopId")},
				"requestBody": jsonBody("CreateShopReviewRequest"),
				"responses":   mergeResponses(success(http.StatusCreated, "ShopReviewResponse"), errorResponse),
			},
		},
		"/v1/admin/shops": gin.H{
			"get": gin.H{
				"summary":  "Admin list all shops",
				"security": []gin.H{{"bearerAuth": []string{}}},
				"parameters": []gin.H{
					queryParam("page", "integer"),
					queryParam("pageSize", "integer"),
					queryParam("q", "string"),
					queryParam("ownerUserId", "string"),
					queryParam("status", "string"),
					realtimeQueryParam(),
				},
				"responses": mergeResponses(success(http.StatusOK, "ShopListResponse"), errorResponse),
			},
		},
		"/v1/admin/product-freshness-reports/{reportId}/moderation": gin.H{
			"patch": gin.H{
				"summary":     "Moderate product freshness report",
				"security":    []gin.H{{"bearerAuth": []string{}}},
				"parameters":  []gin.H{pathParam("reportId")},
				"requestBody": jsonBody("ModerateProductFreshnessReportRequest"),
				"responses":   mergeResponses(success(http.StatusOK, "ProductFreshnessReportResponse"), errorResponse),
			},
		},
		"/v1/admin/product-freshness-reports": gin.H{
			"get": gin.H{
				"summary":  "Admin list product freshness reports",
				"security": []gin.H{{"bearerAuth": []string{}}},
				"parameters": []gin.H{
					queryParam("reportId", "string"),
					queryParam("shopId", "string"),
					queryParam("productId", "string"),
					queryParam("reporterUserId", "string"),
					queryParam("status", "string"),
					queryParam("createdAfter", "string"),
					queryParam("createdBefore", "string"),
					queryParam("page", "integer"),
					queryParam("pageSize", "integer"),
					realtimeQueryParam(),
				},
				"responses": mergeResponses(success(http.StatusOK, "ProductFreshnessReportListResponse"), errorResponse),
			},
		},
		"/v1/admin/buyer-checks/{checkId}/moderation": gin.H{
			"patch": gin.H{
				"summary":     "Moderate buyer check",
				"security":    []gin.H{{"bearerAuth": []string{}}},
				"parameters":  []gin.H{pathParam("checkId")},
				"requestBody": jsonBody("ModerateBuyerCheckRequest"),
				"responses":   mergeResponses(success(http.StatusOK, "BuyerCheckResponse"), errorResponse),
			},
		},
		"/v1/admin/buyer-checks": gin.H{
			"get": gin.H{
				"summary":  "Admin list buyer checks",
				"security": []gin.H{{"bearerAuth": []string{}}},
				"parameters": []gin.H{
					queryParam("checkId", "string"),
					queryParam("shopId", "string"),
					queryParam("productId", "string"),
					queryParam("buyerUserId", "string"),
					queryParam("status", "string"),
					queryParam("verdict", "string"),
					queryParam("createdAfter", "string"),
					queryParam("createdBefore", "string"),
					queryParam("page", "integer"),
					queryParam("pageSize", "integer"),
					realtimeQueryParam(),
				},
				"responses": mergeResponses(success(http.StatusOK, "BuyerCheckListResponse"), errorResponse),
			},
		},
		"/v1/admin/shops/{shopId}/moderation": gin.H{
			"patch": gin.H{
				"summary":     "Moderate shop status",
				"security":    []gin.H{{"bearerAuth": []string{}}},
				"parameters":  []gin.H{pathParam("shopId")},
				"requestBody": jsonBody("ModerateShopRequest"),
				"responses":   mergeResponses(success(http.StatusOK, "ShopResponse"), errorResponse),
			},
		},
		"/v1/admin/users": gin.H{
			"get": gin.H{
				"summary":  "Admin list users",
				"security": []gin.H{{"bearerAuth": []string{}}},
				"parameters": []gin.H{
					queryParam("status", "string"),
					queryParam("role", "string"),
					realtimeQueryParam(),
				},
				"responses": mergeResponses(success(http.StatusOK, "AdminUserListResponse"), errorResponse),
			},
		},
		"/v1/admin/users/{userId}/role": gin.H{
			"patch": gin.H{
				"summary":  "Admin update user role",
				"security": []gin.H{{"bearerAuth": []string{}}},
				"parameters": []gin.H{
					pathParam("userId"),
				},
				"requestBody": jsonBody("UpdateUserRoleRequest"),
				"responses":   mergeResponses(success(http.StatusOK, "AdminUserResponse"), errorResponse),
			},
		},
		"/v1/admin/users/{userId}/status": gin.H{
			"patch": gin.H{
				"summary":  "Admin moderate user status",
				"security": []gin.H{{"bearerAuth": []string{}}},
				"parameters": []gin.H{
					pathParam("userId"),
				},
				"requestBody": jsonBody("UpdateUserStatusRequest"),
				"responses":   mergeResponses(success(http.StatusOK, "AdminUserResponse"), errorResponse),
			},
		},
		"/v1/admin/users/{userId}/keys/rotate": gin.H{
			"post": gin.H{
				"summary":  "Admin rotate account key",
				"security": []gin.H{{"bearerAuth": []string{}}},
				"parameters": []gin.H{
					pathParam("userId"),
				},
				"requestBody": jsonBody("AccountKeyRequest"),
				"responses":   mergeResponses(success(http.StatusOK, "AccountKeyResponse"), errorResponse),
			},
		},
		"/v1/admin/users/{userId}/keys/recover": gin.H{
			"post": gin.H{
				"summary":  "Admin recover account key",
				"security": []gin.H{{"bearerAuth": []string{}}},
				"parameters": []gin.H{
					pathParam("userId"),
				},
				"requestBody": jsonBody("AccountKeyRequest"),
				"responses":   mergeResponses(success(http.StatusOK, "AccountKeyResponse"), errorResponse),
			},
		},
		"/v1/admin/users/{userId}/keys/backfill": gin.H{
			"post": gin.H{
				"summary":  "Admin backfill missing account key metadata",
				"security": []gin.H{{"bearerAuth": []string{}}},
				"parameters": []gin.H{
					pathParam("userId"),
				},
				"requestBody": jsonBody("AccountKeyRequest"),
				"responses":   mergeResponses(success(http.StatusOK, "AccountKeyResponse"), errorResponse),
			},
		},
		"/v1/seller/score": gin.H{
			"post": gin.H{
				"summary":  "Score seller image",
				"security": []gin.H{{"bearerAuth": []string{}}},
				"requestBody": gin.H{
					"required": true,
					"content": gin.H{
						"multipart/form-data": gin.H{
							"schema": gin.H{
								"type":     "object",
								"required": []string{"image"},
								"properties": gin.H{
									"image": gin.H{"type": "string", "format": "binary"},
								},
							},
						},
					},
				},
				"responses": mergeResponses(success(http.StatusOK, "SellerScoreResponse"), errorResponse),
			},
		},
		"/v1/seller/commit": gin.H{
			"post": gin.H{
				"summary":     "Commit seller pledge",
				"security":    []gin.H{{"bearerAuth": []string{}}},
				"requestBody": jsonBody("SellerCommitRequest"),
				"responses":   mergeResponses(success(http.StatusCreated, "SellerCommitResponse"), errorResponse),
			},
		},
		"/v1/shops/{shopId}/pledges/{pledgeId}/bundle-token": gin.H{
			"post": gin.H{
				"summary":    "Re-issue bundle token for an existing pledge",
				"security":   []gin.H{{"bearerAuth": []string{}}},
				"parameters": []gin.H{pathParam("shopId"), pathParam("pledgeId")},
				"responses":  mergeResponses(success(http.StatusOK, "BundleTokenResponse"), errorResponse),
			},
		},
		"/v1/buyer/check": gin.H{
			"post": gin.H{
				"summary":  "Check buyer image against seller pledge by bundle",
				"security": []gin.H{{"bearerAuth": []string{}}},
				"requestBody": gin.H{
					"required": true,
					"content": gin.H{
						"multipart/form-data": gin.H{
							"schema": gin.H{
								"type":     "object",
								"required": []string{"image", "bundleId", "bundleToken"},
								"properties": gin.H{
									"pledgeId":       gin.H{"type": "string"},
									"bundleId":       gin.H{"type": "string"},
									"bundleToken":    gin.H{"type": "string"},
									"locationStatus": gin.H{"type": "string"},
									"image":          gin.H{"type": "string", "format": "binary"},
								},
							},
						},
					},
				},
				"responses": mergeResponses(success(http.StatusOK, "BuyerCheckResponse"), errorResponse),
			},
		},
	}
	paths["/v1/me/shop"] = gin.H{"get": gin.H{"summary": "Get the authenticated seller shop", "security": []gin.H{{"bearerAuth": []string{}}}, "responses": mergeResponses(success(http.StatusOK, "ShopResponse"), errorResponse)}}
	paths["/v1/seller/shops/{shopId}/products"] = gin.H{"get": gin.H{"summary": "List all products owned by the authenticated seller", "security": []gin.H{{"bearerAuth": []string{}}}, "parameters": []gin.H{pathParam("shopId")}, "responses": mergeResponses(success(http.StatusOK, "ProductListResponse"), errorResponse)}}
	paths["/v1/vouchers/check"] = gin.H{"post": gin.H{"summary": "Validate a voucher and calculate its discount", "requestBody": jsonBody("VoucherCheckRequest"), "responses": mergeResponses(success(http.StatusOK, "VoucherCheckResponse"), errorResponse)}}
	paths["/v1/me/vouchers"] = gin.H{"get": gin.H{"summary": "List the authenticated user's voucher wallet", "security": []gin.H{{"bearerAuth": []string{}}}, "responses": errorResponse}, "post": gin.H{"summary": "Save a voucher to the authenticated user's wallet", "security": []gin.H{{"bearerAuth": []string{}}}, "responses": mergeResponses(success(http.StatusCreated, "UserVoucherResponse"), errorResponse)}}
	paths["/v1/me/vouchers/{userVoucherId}/use"] = gin.H{"post": gin.H{"summary": "Mark a wallet voucher as used", "security": []gin.H{{"bearerAuth": []string{}}}, "parameters": []gin.H{pathParam("userVoucherId")}, "responses": mergeResponses(success(http.StatusOK, "UserVoucherResponse"), errorResponse)}}

	annotatePathDocs(paths)
	return paths
}

func annotatePathDocs(paths gin.H) {
	type operationDoc struct {
		summary     string
		description string
	}

	docs := map[string]map[string]operationDoc{
		"/v1/auth/register": {
			"post": {
				summary:     "Đăng ký tài khoản | Register account",
				description: "Tạo tài khoản mới bằng email và mật khẩu.\n\nCreate a new account with email and password. Nếu email nằm trong BOOTSTRAP_ADMIN_EMAILS thì tài khoản mới có thể được gán role admin ngay lúc tạo. Private key không bao giờ trả về cho client.",
			},
		},
		"/v1/auth/login": {
			"post": {
				summary:     "Đăng nhập email | Login with email",
				description: "Đăng nhập bằng email và mật khẩu.\n\nAuthenticate with email and password. Trả về access token, refresh token, và public key nếu tài khoản hợp lệ.",
			},
		},
		"/v1/auth/google": {
			"post": {
				summary:     "Đăng nhập Google | Login with Google",
				description: "Đăng nhập bằng Google ID token.\n\nAuthenticate with Google. Nếu đây là lần đầu của account, backend sẽ tạo user mới và có thể bootstrap admin role theo email whitelist.",
			},
		},
		"/v1/auth/refresh": {
			"post": {
				summary:     "Làm mới token | Refresh token",
				description: "Đổi refresh token lấy access token mới.\n\nExchange a valid refresh token for a new access token and rotated refresh token.",
			},
		},
		"/v1/auth/logout": {
			"post": {
				summary:     "Đăng xuất | Logout",
				description: "Thu hồi refresh token để đăng xuất an toàn.\n\nRevoke the provided refresh token. Access token đã cấp vẫn tồn tại đến khi hết hạn.",
			},
		},
		"/v1/me": {
			"get": {
				summary:     "Thông tin tài khoản hiện tại | Current account",
				description: "Lấy thông tin cơ bản của người dùng đang đăng nhập.\n\nReturn the current authenticated user profile summary.",
			},
			"patch": {
				summary:     "Cập nhật hồ sơ cá nhân | Update current profile",
				description: "Cập nhật tên hiển thị, họ và tên của tài khoản hiện tại với expectedVersion để tránh ghi đè.\n\nUpdate display name, first name, and last name using optimistic concurrency.",
			},
			"delete": {
				summary:     "Xóa mềm tài khoản | Soft delete account",
				description: "Không xóa vật lý dữ liệu, chỉ chuyển trạng thái sang deleted.\n\nSoft-delete the current account. Data is retained for transparency, audit, and future reconciliation.",
			},
		},
		"/v1/events": {
			"get": {
				summary:     "Liệt kê event log đã ký | List signed event logs",
				description: "Trả về danh sách event log đã ký để phục vụ audit và proof.\n\nUse filters like resourceType, resourceId, actorUserId, action, status, sequence, and time range for troubleshooting or admin review.",
			},
		},
		"/v1/events/verify": {
			"get": {
				summary:     "Xác minh chuỗi event theo resource | Verify resource event chain",
				description: "Kiểm tra toàn bộ event của một resource có hợp lệ không.\n\nVerify event content hash, signature, and chain linkage for all events attached to a resource.",
			},
		},
		"/v1/events/{eventId}/verify": {
			"get": {
				summary:     "Xác minh một event | Verify single event",
				description: "Kiểm tra hash, chữ ký, và liên kết previousEventId của một event cụ thể.\n\nUse this when admin or support needs to inspect one event in detail.",
			},
		},
		"/v1/shops": {
			"get": {
				summary:     "Danh sách cửa hàng | List shops",
				description: "Lấy danh sách cửa hàng công khai cho buyer app.\n\nReturns shop cards with trust summary and rating summary. Suitable for discovery screens.",
			},
			"post": {
				summary:     "Tạo cửa hàng | Create shop",
				description: "Mỗi tài khoản chỉ được phép tạo tối đa 1 cửa hàng. Trả 409 Conflict nếu đã có cửa hàng active/pending.\n\nCreate a new shop. Each account can own at most one non-deleted shop. Returns 409 if the account already owns a shop.",
			},
		},
		"/v1/shops/{shopId}": {
			"get": {
				summary:     "Chi tiết cửa hàng | Shop detail",
				description: "Trả về thông tin đầy đủ của cửa hàng, trust score, và rating summary.\n\nFrontend should use this to render shop detail, trust cards, and navigation entry points.",
			},
			"put": {
				summary:     "Cập nhật cửa hàng | Update shop",
				description: "Cập nhật thông tin cửa hàng với expectedVersion để tránh ghi đè.\n\nOptimistic concurrency is enforced through expectedVersion.",
			},
			"delete": {
				summary:     "Xóa mềm cửa hàng | Soft delete shop",
				description: "Không xóa vật lý document shop, chỉ đổi status sang deleted.\n\nThe record stays available for audit and blockchain integrity history.",
			},
		},
		"/v1/shops/{shopId}/pledges": {
			"get": {
				summary:     "Lịch sử cam kết | Pledge history",
				description: "Danh sách các cam kết của seller theo shop, có thể lọc theo productId hoặc category.\n\nBuyer UI should use this to show commitment history and trust timeline.",
			},
		},
		"/v1/shops/{shopId}/pledges/{pledgeId}/integrity": {
			"get": {
				summary:     "Tình trạng integrity | Integrity status",
				description: "Trả về tình trạng neo hash lên chain và kết quả đối chiếu giữa DB và on-chain.\n\nTechnical endpoint for admin, support, and integrity debugging.",
			},
		},
		"/v1/shops/{shopId}/pledges/{pledgeId}/proof": {
			"get": {
				summary:     "Proof thân thiện với người dùng | Buyer-friendly proof",
				description: "Trả về gói proof để frontend hiển thị theo ngôn ngữ dễ hiểu.\n\nPrefer proofStatus, proofHeadline, proofSummary, and recommendedActions instead of low-level blockchain wording.",
			},
		},
		"/v1/media/images": {
			"post": {
				summary:     "Upload ảnh dùng lại được | Upload reusable image",
				description: "Upload ảnh và nhận imageHash, imageCid, gatewayUrl, contentType, sizeBytes.\n\nMobile should call this first for retryable media flows before seller commit or buyer check.",
			},
		},
		"/v1/shops/{shopId}/products": {
			"get": {
				summary:     "Danh sách sản phẩm | List products",
				description: "Lấy danh sách sản phẩm của shop.\n\nSupports buyer-facing browse screens with category, tag, and sort filters.",
			},
			"post": {
				summary:     "Tạo sản phẩm | Create product",
				description: "Seller tạo sản phẩm mới trong shop của mình.\n\nProduct lifecycle uses status and version so changes remain auditable.",
			},
		},
		"/v1/shops/{shopId}/products/{productId}": {
			"get": {
				summary:     "Chi tiết sản phẩm | Product detail",
				description: "Trả về dữ liệu đầy đủ của sản phẩm, phù hợp cho màn hình product detail.\n\nIncludes freshness metadata and publication status.",
			},
			"put": {
				summary:     "Cập nhật sản phẩm | Update product",
				description: "Cập nhật sản phẩm với expectedVersion.\n\nUse this for seller edit flows and lifecycle changes.",
			},
			"delete": {
				summary:     "Xóa mềm sản phẩm | Soft delete product",
				description: "Không xóa vật lý sản phẩm. Backend chỉ đổi status sang deleted.\n\nThe historical record remains available for proof and audit.",
			},
		},
		"/v1/shops/{shopId}/products/{productId}/freshness-reports": {
			"get": {
				summary:     "Danh sách báo cáo độ tươi | List freshness reports",
				description: "Trả về các báo cáo độ tươi của product.\n\nUseful for buyer context and seller product quality history.",
			},
			"post": {
				summary:     "Tạo báo cáo độ tươi | Create freshness report",
				description: "Buyer gửi báo cáo độ tươi cho một sản phẩm.\n\nThis is user-generated evidence and may later be moderated or weighted into trust logic.",
			},
		},
		"/v1/shops/{shopId}/reviews": {
			"get": {
				summary:     "Danh sách review cửa hàng | List shop reviews",
				description: "Lấy review hiện có của cửa hàng.\n\nBuyer-facing endpoint for public review display.",
			},
			"post": {
				summary:     "Tạo hoặc sửa review | Create or update review",
				description: "Người dùng đăng nhập tạo hoặc cập nhật review của mình cho shop.\n\nReview records also use soft lifecycle instead of hard delete.",
			},
		},
		"/v1/admin/shops": {
			"get": {
				summary:     "Admin xem tất cả cửa hàng | Admin list shops",
				description: "Danh sách cửa hàng cho moderation và kiểm tra nội bộ.\n\nAdmin can filter by owner and status.",
			},
		},
		"/v1/admin/shops/{shopId}/moderation": {
			"patch": {
				summary:     "Admin duyệt cửa hàng | Moderate shop",
				description: "Admin cập nhật moderation status của cửa hàng.\n\nThis endpoint should be used for approve, reject, or archive style moderation workflows.",
			},
		},
		"/v1/admin/product-freshness-reports/{reportId}/moderation": {
			"patch": {
				summary:     "Admin duyệt báo cáo độ tươi | Moderate freshness report",
				description: "Admin đánh dấu report là accepted, flagged, hoặc rejected theo policy.\n\nUse when the team needs to separate trusted reports from noise or abuse.",
			},
		},
		"/v1/admin/product-freshness-reports": {
			"get": {
				summary:     "Admin xem danh sách báo cáo độ tươi | Admin list freshness reports",
				description: "Danh sách chung báo cáo độ tươi toàn hệ thống cho moderation và kiểm tra chất lượng.\n\nSupports filtering by reportId, shop, product, reporter, status, and time range.",
			},
		},
		"/v1/admin/buyer-checks/{checkId}/moderation": {
			"patch": {
				summary:     "Admin duyệt buyer check | Moderate buyer check",
				description: "Admin cập nhật moderation state của buyer check.\n\nUseful for abuse handling, dispute review, and trust-score hygiene.",
			},
		},
		"/v1/admin/buyer-checks": {
			"get": {
				summary:     "Admin xem danh sách buyer check | Admin list buyer checks",
				description: "Danh sách chung buyer check toàn hệ thống để lọc rủi ro và xử lý theo lô.\n\nSupports filtering by checkId, shop, product, buyer, verdict, status, and time range.",
			},
		},
		"/v1/admin/users": {
			"get": {
				summary:     "Admin xem danh sách user | Admin list users",
				description: "Danh sách user cho moderation và quản trị role.\n\nSupports filtering by status and role.",
			},
		},
		"/v1/admin/users/{userId}/role": {
			"patch": {
				summary:     "Admin đổi role user | Update user role",
				description: "Admin cập nhật role của user, ví dụ từ user sang admin.\n\nThis should be the main long-term role management mechanism, not environment bootstrap.",
			},
		},
		"/v1/admin/users/{userId}/status": {
			"patch": {
				summary:     "Admin đổi trạng thái user | Moderate user status",
				description: "Admin cập nhật status của user như active, suspended, hoặc deleted.\n\nUse expectedVersion to avoid overwriting concurrent state changes.",
			},
		},
		"/v1/admin/users/{userId}/keys/rotate": {
			"post": {
				summary:     "Admin đổi khóa tài khoản | Rotate account key",
				description: "Tạo cặp khóa mới cho account và cập nhật metadata không nhạy cảm.\n\nPrivate key stays in Vault and is never returned to the client.",
			},
		},
		"/v1/admin/users/{userId}/keys/recover": {
			"post": {
				summary:     "Admin khôi phục khóa | Recover account key",
				description: "Khôi phục metadata key hoặc secret khi Firestore và Vault bị lệch nhau.\n\nUse for operational recovery, not for regular user flows.",
			},
		},
		"/v1/admin/users/{userId}/keys/backfill": {
			"post": {
				summary:     "Admin bổ sung metadata key | Backfill account key",
				description: "Bổ sung metadata key cho account cũ thiếu thông tin key management.\n\nUseful during migration or legacy account repair.",
			},
		},
		"/v1/seller/score": {
			"post": {
				summary:     "AI chấm điểm ảnh seller | Score seller image",
				description: "Nhận ảnh từ seller và trả về score, category, confidence.\n\nUse this before seller commit. If IPFS is enabled, imageCid can also be returned.",
			},
		},
		"/v1/seller/commit": {
			"post": {
				summary:     "Seller tạo cam kết | Commit seller pledge",
				description: "Seller xác nhận cam kết chất lượng cho shop hoặc product.\n\nDepending on runtime config, this may also prepare integrity hash, upload media metadata, and anchor proof on Besu.",
			},
		},
		"/v1/shops/{shopId}/pledges/{pledgeId}/bundle-token": {
			"post": {
				summary:     "Seller cấp lại token bó hàng | Re-issue bundle token",
				description: "Cấp lại bundle token đã ký cho pledge đang tồn tại khi token cũ hết hạn hoặc buyer cần quét lại.\n\nOnly owner of the shop can issue token for its pledge.",
			},
		},
		"/v1/buyer/check": {
			"post": {
				summary:     "Buyer kiểm tra bằng ảnh | Buyer image check",
				description: "Buyer upload ảnh để kiểm tra chất lượng hiện tại.\n\nbundleToken là bắt buộc để ràng buộc đúng bó hàng và chống replay. If pledgeId is provided, the system compares against the seller pledge.",
			},
		},
	}

	for path, methods := range docs {
		pathItem, ok := paths[path].(gin.H)
		if !ok {
			continue
		}
		for method, doc := range methods {
			op, ok := pathItem[method].(gin.H)
			if !ok {
				continue
			}
			op["summary"] = doc.summary
			op["description"] = doc.description
		}
	}
}

func mergeResponses(base, extra gin.H) gin.H {
	merged := gin.H{}
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range extra {
		merged[key] = value
	}
	return merged
}

func pathParam(name string) gin.H {
	return gin.H{
		"name":     name,
		"in":       "path",
		"required": true,
		"schema":   gin.H{"type": "string"},
	}
}

func queryParam(name, schemaType string) gin.H {
	return gin.H{
		"name":     name,
		"in":       "query",
		"required": false,
		"schema":   gin.H{"type": schemaType},
	}
}

func realtimeQueryParam() gin.H {
	return gin.H{
		"name":        "realtime",
		"in":          "query",
		"required":    false,
		"description": "Set to 1/true to bypass Redis cache and fetch fresh data directly from database",
		"schema":      gin.H{"type": "boolean"},
	}
}

func requiredQueryParam(name, schemaType string) gin.H {
	return gin.H{
		"name":     name,
		"in":       "query",
		"required": true,
		"schema":   gin.H{"type": schemaType},
	}
}

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>VNGrocery API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
  <style>
    body { margin: 0; background: #f6f8fb; }
    .topbar { display: none; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    const specUrl = new URL('/openapi.json', window.location.href).toString();
    window.ui = SwaggerUIBundle({
      url: specUrl,
      dom_id: '#swagger-ui',
      deepLinking: true,
      persistAuthorization: true,
      displayRequestDuration: true,
      validatorUrl: null
    });
  </script>
</body>
</html>`
