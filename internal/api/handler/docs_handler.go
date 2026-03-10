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
			"description": "Backend API for authentication, shop management, trust checks, moderation, and reviews.",
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
	return gin.H{
		"RegisterRequest": gin.H{
			"type":     "object",
			"required": []string{"email", "password"},
			"properties": gin.H{
				"email":       gin.H{"type": "string", "format": "email"},
				"password":    gin.H{"type": "string"},
				"displayName": gin.H{"type": "string"},
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
		"AuthTokenResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"accessToken": gin.H{"type": "string"},
				"userId":      gin.H{"type": "string"},
				"email":       gin.H{"type": "string"},
				"publicKey":   gin.H{"type": "string"},
			},
		},
		"MeResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"userId": gin.H{"type": "string"},
				"email":  gin.H{"type": "string"},
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
				"price":           gin.H{"type": "number"},
				"currency":        gin.H{"type": "string"},
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
				"price":       gin.H{"type": "number"},
				"currency":    gin.H{"type": "string"},
				"status":      gin.H{"type": "string"},
				"version":     gin.H{"type": "integer"},
				"createdAt":   gin.H{"type": "string", "format": "date-time"},
				"updatedAt":   gin.H{"type": "string", "format": "date-time"},
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
		"SellerCommitRequest": gin.H{
			"type":     "object",
			"required": []string{"shopId", "score", "category", "confidence"},
			"properties": gin.H{
				"shopId":     gin.H{"type": "string"},
				"score":      gin.H{"type": "number"},
				"category":   gin.H{"type": "string"},
				"confidence": gin.H{"type": "number"},
			},
		},
		"SellerCommitResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"pledgeId":        gin.H{"type": "string"},
				"shopId":          gin.H{"type": "string"},
				"createdByUserId": gin.H{"type": "string"},
				"status":          gin.H{"type": "string"},
				"score":           gin.H{"type": "number"},
				"category":        gin.H{"type": "string"},
				"confidence":      gin.H{"type": "number"},
				"createdAt":       gin.H{"type": "string", "format": "date-time"},
				"updatedAt":       gin.H{"type": "string", "format": "date-time"},
			},
		},
		"SellerScoreResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"score":      gin.H{"type": "number"},
				"category":   gin.H{"type": "string"},
				"confidence": gin.H{"type": "number"},
			},
		},
		"BuyerCheckResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"policyVersion":    gin.H{"type": "string"},
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
				"categoryMatch":    gin.H{"type": "boolean"},
				"reasons": gin.H{
					"type":  "array",
					"items": gin.H{"type": "string"},
				},
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
		"ErrorResponse": gin.H{
			"type": "object",
			"properties": gin.H{
				"error": gin.H{"type": "string"},
			},
		},
	}
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

	return gin.H{
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
		"/v1/me": gin.H{
			"get": gin.H{
				"summary":   "Get current user",
				"security":  []gin.H{{"bearerAuth": []string{}}},
				"responses": mergeResponses(success(http.StatusOK, "MeResponse"), errorResponse),
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
		"/v1/events": gin.H{
			"get": gin.H{
				"summary":  "List signed event logs",
				"security": []gin.H{{"bearerAuth": []string{}}},
				"parameters": []gin.H{
					queryParam("resourceType", "string"),
					queryParam("resourceId", "string"),
					queryParam("actorUserId", "string"),
				},
				"responses": gin.H{
					"200": gin.H{
						"description": "OK",
						"content": gin.H{
							"application/json": gin.H{
								"schema": gin.H{
									"type":  "array",
									"items": gin.H{"$ref": "#/components/schemas/EventLogResponse"},
								},
							},
						},
					},
					"default": errorResponse["default"],
				},
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
				},
				"responses": mergeResponses(success(http.StatusOK, "ShopListResponse"), errorResponse),
			},
			"post": gin.H{
				"summary":     "Create shop",
				"security":    []gin.H{{"bearerAuth": []string{}}},
				"requestBody": jsonBody("UpsertShopRequest"),
				"responses":   mergeResponses(success(http.StatusCreated, "ShopResponse"), errorResponse),
			},
		},
		"/v1/shops/{shopId}": gin.H{
			"get": gin.H{
				"summary":    "Get shop detail",
				"parameters": []gin.H{pathParam("shopId")},
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
		"/v1/shops/{shopId}/products": gin.H{
			"get": gin.H{
				"summary":    "List shop products",
				"parameters": []gin.H{pathParam("shopId")},
				"responses":  mergeResponses(success(http.StatusOK, "ProductListResponse"), errorResponse),
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
		"/v1/shops/{shopId}/reviews": gin.H{
			"get": gin.H{
				"summary":    "List shop reviews",
				"parameters": []gin.H{pathParam("shopId")},
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
				},
				"responses": mergeResponses(success(http.StatusOK, "ShopListResponse"), errorResponse),
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
		"/v1/buyer/check": gin.H{
			"post": gin.H{
				"summary": "Check buyer image against pledge",
				"requestBody": gin.H{
					"required": true,
					"content": gin.H{
						"multipart/form-data": gin.H{
							"schema": gin.H{
								"type":     "object",
								"required": []string{"pledgeId", "image"},
								"properties": gin.H{
									"pledgeId": gin.H{"type": "string"},
									"image":    gin.H{"type": "string", "format": "binary"},
								},
							},
						},
					},
				},
				"responses": mergeResponses(success(http.StatusOK, "BuyerCheckResponse"), errorResponse),
			},
		},
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
