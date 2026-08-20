package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"vngrocery/internal/api/dto"
	"vngrocery/internal/api/middleware"
	"vngrocery/internal/domain"
	productsvc "vngrocery/internal/service/product"
)

type ProductService interface {
	History(ctx context.Context, input productsvc.HistoryInput) (productsvc.ProductHistory, error)
	Create(ctx context.Context, input productsvc.CreateInput) (domain.Product, error)
	Update(ctx context.Context, input productsvc.UpdateInput) (domain.Product, error)
	Delete(ctx context.Context, input productsvc.DeleteInput) (domain.Product, error)
	Moderate(ctx context.Context, input productsvc.ModerateInput) (domain.Product, error)
	BulkUpsert(ctx context.Context, input productsvc.BulkUpsertInput) ([]domain.Product, error)
	GetByID(ctx context.Context, shopID, productID string) (domain.Product, error)
	List(ctx context.Context, input productsvc.ListInput) ([]domain.Product, error)
	CreateFreshnessReport(ctx context.Context, input productsvc.FreshnessReportInput) (domain.ProductFreshnessReport, error)
	ModerateFreshnessReport(ctx context.Context, input productsvc.ModerateFreshnessReportInput) (domain.ProductFreshnessReport, error)
	ListFreshnessReports(ctx context.Context, shopID, productID string) ([]domain.ProductFreshnessReport, error)
	ListFreshnessReportsAdmin(ctx context.Context, input productsvc.ListFreshnessReportAdminInput) (productsvc.ListFreshnessReportAdminResult, error)
}

type ProductHandler struct {
	products ProductService
}

func NewProductHandler(products ProductService) *ProductHandler {
	return &ProductHandler{products: products}
}

func (h *ProductHandler) Create(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}

	var request dto.UpsertProductRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	product, err := h.products.Create(c.Request.Context(), productsvc.CreateInput{
		ShopID:         c.Param("shopId"),
		OwnerUserID:    principal.UserID,
		Name:           request.Name,
		Description:    request.Description,
		Category:       request.Category,
		Tags:           request.Tags,
		ImageURLs:      request.ImageURLs,
		FreshnessNote:  request.FreshnessNote,
		FreshnessScore: request.FreshnessScore,
		Price:          request.Price,
		Currency:       request.Currency,
		Status:         request.Status,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toProductResponse(product))
}

func (h *ProductHandler) Update(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}

	var request dto.UpsertProductRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	product, err := h.products.Update(c.Request.Context(), productsvc.UpdateInput{
		ProductID:       c.Param("productId"),
		ShopID:          c.Param("shopId"),
		OwnerUserID:     principal.UserID,
		ExpectedVersion: request.ExpectedVersion,
		Name:            request.Name,
		Description:     request.Description,
		Category:        request.Category,
		Tags:            request.Tags,
		ImageURLs:       request.ImageURLs,
		FreshnessNote:   request.FreshnessNote,
		FreshnessScore:  request.FreshnessScore,
		Price:           request.Price,
		Currency:        request.Currency,
		Status:          request.Status,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProductResponse(product))
}

func (h *ProductHandler) Delete(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}

	expectedVersion, parseErr := parsePositiveIntQuery(c.Query("expectedVersion"), "expectedVersion")
	if parseErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": parseErr.Error()})
		return
	}

	product, err := h.products.Delete(c.Request.Context(), productsvc.DeleteInput{
		ProductID:       c.Param("productId"),
		ShopID:          c.Param("shopId"),
		OwnerUserID:     principal.UserID,
		ExpectedVersion: expectedVersion,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProductResponse(product))
}

func (h *ProductHandler) Moderate(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}

	var request dto.ModerateProductRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	product, err := h.products.Moderate(c.Request.Context(), productsvc.ModerateInput{
		ProductID:       c.Param("productId"),
		ModeratorUserID: principal.UserID,
		ExpectedVersion: request.ExpectedVersion,
		Status:          request.Status,
		ModerationNote:  request.ModerationNote,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProductResponse(product))
}

func (h *ProductHandler) BulkUpsert(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}

	var request dto.BulkUpsertProductRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	items := make([]productsvc.BulkUpsertItemInput, 0, len(request.Items))
	for _, item := range request.Items {
		items = append(items, productsvc.BulkUpsertItemInput{
			ProductID:       item.ProductID,
			ExpectedVersion: item.ExpectedVersion,
			Name:            item.Name,
			Description:     item.Description,
			Category:        item.Category,
			Tags:            item.Tags,
			ImageURLs:       item.ImageURLs,
			FreshnessNote:   item.FreshnessNote,
			FreshnessScore:  item.FreshnessScore,
			Price:           item.Price,
			Currency:        item.Currency,
			Status:          item.Status,
		})
	}
	products, err := h.products.BulkUpsert(c.Request.Context(), productsvc.BulkUpsertInput{
		ShopID:      c.Param("shopId"),
		OwnerUserID: principal.UserID,
		Items:       items,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	response := dto.ProductListResponse{Items: make([]dto.ProductResponse, 0, len(products))}
	for _, product := range products {
		response.Items = append(response.Items, toProductResponse(product))
	}
	c.JSON(http.StatusOK, response)
}

func (h *ProductHandler) GetByID(c *gin.Context) {
	product, err := h.products.GetByID(c.Request.Context(), c.Param("shopId"), c.Param("productId"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProductResponse(product))
}

func (h *ProductHandler) List(c *gin.Context) {
	products, err := h.products.List(c.Request.Context(), productsvc.ListInput{
		ShopID:   c.Param("shopId"),
		Query:    c.Query("q"),
		Category: c.Query("category"),
		Tag:      c.Query("tag"),
		Sort:     c.Query("sort"),
	})
	if err != nil {
		h.writeError(c, err)
		return
	}

	items := make([]dto.ProductResponse, 0, len(products))
	for _, product := range products {
		items = append(items, toProductResponse(product))
	}
	c.JSON(http.StatusOK, dto.ProductListResponse{Items: items})
}

func (h *ProductHandler) ListSeller(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}
	products, err := h.products.List(c.Request.Context(), productsvc.ListInput{ShopID: c.Param("shopId"), Query: c.Query("q"), Category: c.Query("category"), Tag: c.Query("tag"), Sort: c.Query("sort"), OwnerUserID: principal.UserID, IncludeAllStatuses: true})
	if err != nil {
		h.writeError(c, err)
		return
	}
	items := make([]dto.ProductResponse, 0, len(products))
	for _, product := range products {
		items = append(items, toProductResponse(product))
	}
	c.JSON(http.StatusOK, dto.ProductListResponse{Items: items})
}

func (h *ProductHandler) CreateFreshnessReport(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}

	var request dto.CreateProductFreshnessReportRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	report, err := h.products.CreateFreshnessReport(c.Request.Context(), productsvc.FreshnessReportInput{
		ShopID:         c.Param("shopId"),
		ProductID:      c.Param("productId"),
		ReporterUserID: principal.UserID,
		Score:          request.Score,
		Category:       request.Category,
		Confidence:     request.Confidence,
		Comment:        request.Comment,
		ImageHash:      request.ImageHash,
		ImageCID:       request.ImageCID,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toProductFreshnessReportResponse(report))
}

func (h *ProductHandler) ListFreshnessReports(c *gin.Context) {
	reports, err := h.products.ListFreshnessReports(c.Request.Context(), c.Param("shopId"), c.Param("productId"))
	if err != nil {
		h.writeError(c, err)
		return
	}

	items := make([]dto.ProductFreshnessReportResponse, 0, len(reports))
	for _, report := range reports {
		items = append(items, toProductFreshnessReportResponse(report))
	}
	c.JSON(http.StatusOK, dto.ProductFreshnessReportListResponse{Items: items})
}

func (h *ProductHandler) ListFreshnessReportsAdmin(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}

	page, pageSize, err := parsePagination(c.Query("page"), c.Query("pageSize"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	createdAfter, err := parseOptionalTime(c.Query("createdAfter"), "createdAfter")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	createdBefore, err := parseOptionalTime(c.Query("createdBefore"), "createdBefore")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.products.ListFreshnessReportsAdmin(c.Request.Context(), productsvc.ListFreshnessReportAdminInput{
		ActorUserID:    principal.UserID,
		ReportID:       c.Query("reportId"),
		ShopID:         c.Query("shopId"),
		ProductID:      c.Query("productId"),
		ReporterUserID: c.Query("reporterUserId"),
		Status:         c.Query("status"),
		CreatedAfter:   createdAfter,
		CreatedBefore:  createdBefore,
		Page:           page,
		PageSize:       pageSize,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}

	items := make([]dto.ProductFreshnessReportResponse, 0, len(result.Items))
	for _, report := range result.Items {
		items = append(items, toProductFreshnessReportResponse(report))
	}
	totalPages := 0
	if result.Total > 0 {
		totalPages = (result.Total + result.PageSize - 1) / result.PageSize
	}
	c.JSON(http.StatusOK, dto.ProductFreshnessReportListResponse{
		Items: items,
		Pagination: dto.PaginationResponse{
			Page:       result.Page,
			PageSize:   result.PageSize,
			TotalItems: result.Total,
			TotalPages: totalPages,
		},
	})
}

func (h *ProductHandler) ModerateFreshnessReport(c *gin.Context) {
	principal, ok := middleware.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "authenticated principal was not found in request context"})
		return
	}

	var request dto.ModerateProductFreshnessReportRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	report, err := h.products.ModerateFreshnessReport(c.Request.Context(), productsvc.ModerateFreshnessReportInput{
		ReportID:        c.Param("reportId"),
		ModeratorUserID: principal.UserID,
		ExpectedVersion: request.ExpectedVersion,
		Status:          request.Status,
		ModerationNote:  request.ModerationNote,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProductFreshnessReportResponse(report))
}

func (h *ProductHandler) writeError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, productsvc.ErrInvalidProduct):
		status = http.StatusBadRequest
	case errors.Is(err, productsvc.ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(err, productsvc.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, productsvc.ErrVersionConflict):
		status = http.StatusConflict
	}
	c.JSON(status, gin.H{"error": err.Error()})
}

func toProductFreshnessReportResponse(report domain.ProductFreshnessReport) dto.ProductFreshnessReportResponse {
	return dto.ProductFreshnessReportResponse{
		ReportID:          report.ReportID,
		ProductID:         report.ProductID,
		ShopID:            report.ShopID,
		ReporterUserID:    report.ReporterUserID,
		Status:            report.Status,
		Version:           report.Version,
		Score:             report.Score,
		Category:          report.Category,
		Confidence:        report.Confidence,
		Comment:           report.Comment,
		ImageHash:         report.ImageHash,
		ImageCID:          report.ImageCID,
		ModeratedByUserID: report.ModeratedByUserID,
		ModerationNote:    report.ModerationNote,
		ModeratedAt:       report.ModeratedAt,
		CreatedAt:         report.CreatedAt,
		UpdatedAt:         report.UpdatedAt,
	}
}

func toProductResponse(product domain.Product) dto.ProductResponse {
	return dto.ProductResponse{
		ProductID:         product.ProductID,
		ShopID:            product.ShopID,
		OwnerUserID:       product.OwnerUserID,
		Name:              product.Name,
		Description:       product.Description,
		Category:          product.Category,
		Tags:              product.Tags,
		ImageURLs:         product.ImageURLs,
		FreshnessNote:     product.FreshnessNote,
		FreshnessScore:    product.FreshnessScore,
		Price:             product.Price,
		Currency:          product.Currency,
		Status:            product.Status,
		Version:           product.Version,
		ModeratedByUserID: product.ModeratedByUserID,
		ModerationNote:    product.ModerationNote,
		ModeratedAt:       product.ModeratedAt,
		CreatedAt:         product.CreatedAt,
		UpdatedAt:         product.UpdatedAt,
	}
}

// History returns a product's recorded change history and its price series.
func (h *ProductHandler) History(c *gin.Context) {
	windowDays, err := parseOptionalPositiveIntQuery(c.Query("windowDays"), "windowDays")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	history, err := h.products.History(c.Request.Context(), productsvc.HistoryInput{
		ShopID:     c.Param("shopId"),
		ProductID:  c.Param("productId"),
		WindowDays: windowDays,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}

	entries := make([]dto.ProductHistoryEntryResponse, 0, len(history.Entries))
	for _, entry := range history.Entries {
		changes := make([]dto.FieldChangeResponse, 0, len(entry.Changes))
		for _, change := range entry.Changes {
			changes = append(changes, dto.FieldChangeResponse{
				Field:  change.Field,
				Before: change.Before,
				After:  change.After,
			})
		}
		entries = append(entries, dto.ProductHistoryEntryResponse{
			SHA:              entry.SHA,
			ShortSHA:         entry.ShortSHA,
			PreviousSHA:      entry.PreviousSHA,
			Sequence:         entry.Sequence,
			Action:           entry.Action,
			Status:           entry.Status,
			ActorUserID:      entry.ActorUserID,
			ActorName:        entry.ActorName,
			OccurredAt:       entry.OccurredAt,
			Verified:         entry.Verified,
			ContentHashValid: entry.ContentHashValid,
			SignatureValid:   entry.SignatureValid,
			ChainLinkValid:   entry.ChainLinkValid,
			PriceAfter:       entry.PriceAfter,
			Changes:          changes,
		})
	}

	points := make([]dto.PricePointResponse, 0, len(history.PriceHistory))
	for _, point := range history.PriceHistory {
		points = append(points, dto.PricePointResponse{At: point.At, Price: point.Price})
	}

	c.JSON(http.StatusOK, dto.ProductHistoryResponse{
		ProductID:     history.ProductID,
		Entries:       entries,
		ChainVerified: history.ChainVerified,
		PriceHistory:  points,
		WindowDays:    history.WindowDays,
	})
}
