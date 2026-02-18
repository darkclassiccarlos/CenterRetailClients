package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"query-service/internal/cache"
	"query-service/internal/config"
	"query-service/internal/models"
	"query-service/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type InventoryHandler struct {
	logger     *zap.Logger
	repository repository.ReadRepository
	cache      cache.Cache
	cacheTTL   int
}

// GetRepository returns the repository instance (for Kafka consumer)
func (h *InventoryHandler) GetRepository() repository.ReadRepository {
	return h.repository
}

func NewInventoryHandler(logger *zap.Logger, cfg *config.Config) (*InventoryHandler, error) {
	// Create repository (SQLite or InMemory)
	var repo repository.ReadRepository
	var err error

	if cfg.SQLitePath != "" {
		// Use SQLite repository (reads from same database as Listener Service)
		logger.Info("Initializing SQLite repository", zap.String("path", cfg.SQLitePath))
		repo, err = repository.NewSQLiteReadRepository(cfg.SQLitePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create SQLite repository: %w", err)
		}
		logger.Info("SQLite repository initialized successfully")
	} else {
		// Fallback to in-memory repository (for testing)
		logger.Warn("Using in-memory repository (SQLite not configured)")
		repo = repository.NewReadRepository()
	}

	// Create cache client (optional)
	var cacheClient cache.Cache
	if cfg.UseCache {
		logger.Info("Initializing cache (Redis)", zap.String("host", cfg.RedisHost), zap.String("port", cfg.RedisPort))
		cacheClient = cache.NewCache(cfg, logger)
		logger.Info("Cache initialized successfully")
	} else {
		logger.Info("Cache disabled (USE_CACHE=false)")
		cacheClient = nil // Will be checked before use
	}

	return &InventoryHandler{
		logger:     logger,
		repository: repo,
		cache:      cacheClient,
		cacheTTL:   cfg.CacheTTL,
	}, nil
}

// ListItems handles GET /api/v1/inventory/items
// @Summary      List inventory items
// @Description  Obtiene una lista paginada de items de inventario. Los resultados están optimizados para lectura rápida desde cache.
//
// **Características:**
// - Paginación automática (page, page_size)
// - Cache de resultados para mejor rendimiento
// - Respuestas rápidas y escalables
// - Cache-first strategy para baja latencia
//
// **Ejemplos válidos:**
// - Lista con paginación por defecto: `GET /api/v1/inventory/items`
// - Lista con paginación personalizada: `GET /api/v1/inventory/items?page=1&page_size=20`
// - Primera página: `GET /api/v1/inventory/items?page=1&page_size=10`
//
// **Ejemplos inválidos:**
// - Página negativa: `GET /api/v1/inventory/items?page=-1`
// - Page size mayor a 100: `GET /api/v1/inventory/items?page_size=200`
// - Page size negativo: `GET /api/v1/inventory/items?page_size=-10`
//
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-Request-ID  header    string  false  "Request ID for request tracking (UUID). If not provided, a new one will be generated."
// @Param        page          query     int     false  "Page number (default: 1, min: 1)" example(1)
// @Param        page_size     query     int     false  "Items per page (default: 10, min: 1, max: 100)" example(10)
// @Success      200           {object}  ListItemsResponse  "Lista de items obtenida exitosamente"
// @Failure      400           {object}  ErrorResponse      "Request inválido - parámetros de paginación inválidos"
// @Failure      401           {object}  ErrorResponse      "No autorizado - token JWT inválido o faltante"
// @Failure      500           {object}  ErrorResponse      "Error interno del servidor - error de lectura o conexión a base de datos"
// @Failure      503           {object}  ErrorResponse      "Servicio no disponible - error de conexión al cache"
// @Router       /inventory/items [get]
func (h *InventoryHandler) ListItems(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Try cache first (if enabled)
	if h.cache != nil {
		cacheKey := cacheKeyListItems(page, pageSize)
		var cachedResponse ListItemsResponse
		if err := cache.GetJSON(c.Request.Context(), h.cache, cacheKey, &cachedResponse); err == nil {
			h.logger.Debug("Cache hit", zap.String("key", cacheKey))
			c.JSON(http.StatusOK, cachedResponse)
			return
		}
	}

	// Cache miss - fetch from repository
	items, total, err := h.repository.ListItems(c.Request.Context(), page, pageSize)
	if err != nil {
		h.logger.Error("Failed to list items", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list items"})
		return
	}

	// Convert to response models
	responseItems := make([]InventoryItemResponse, len(items))
	for i, item := range items {
		responseItems[i] = InventoryItemResponse{
			ID:          item.ID,
			SKU:         item.SKU,
			Name:        item.Name,
			Description: item.Description,
			Quantity:    item.Quantity,
			Reserved:    item.Reserved,
			Available:   item.Available,
			CreatedAt:   item.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   item.UpdatedAt.Format(time.RFC3339),
		}
	}

	totalPages := (total + pageSize - 1) / pageSize
	response := ListItemsResponse{
		Items:      responseItems,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}

	// Cache the response (if enabled)
	if h.cache != nil {
		cacheKey := cacheKeyListItems(page, pageSize)
		cache.SetJSON(c.Request.Context(), h.cache, cacheKey, response, cache.TTL(h.cacheTTL))
	}

	c.JSON(http.StatusOK, response)
}

// GetItemByID handles GET /api/v1/inventory/items/:id
// @Summary      Get inventory item by ID
// @Description  Obtiene un item de inventario por su ID. Optimizado para lectura rápida desde cache.
//
// **Características:**
// - Cache de resultados individuales
// - Respuestas ultra-rápidas (cache hit)
// - Escalable horizontalmente
// - Cache-first strategy para baja latencia
//
// **Ejemplos válidos:**
// - Obtener item por ID válido: `GET /api/v1/inventory/items/550e8400-e29b-41d4-a716-446655440000`
//
// **Ejemplos inválidos:**
// - ID inválido (UUID malformado): `GET /api/v1/inventory/items/invalid-id`
// - ID no encontrado: `GET /api/v1/inventory/items/550e8400-e29b-41d4-a716-446655440000` (si no existe)
//
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-Request-ID  header    string  false  "Request ID for request tracking (UUID). If not provided, a new one will be generated."
// @Param        id            path      string  true   "Item ID (UUID)" example(550e8400-e29b-41d4-a716-446655440000)
// @Success      200           {object}  InventoryItemResponse  "Item obtenido exitosamente"
// @Failure      400           {object}  ErrorResponse          "ID inválido - UUID malformado"
// @Failure      401           {object}  ErrorResponse          "No autorizado - token JWT inválido o faltante"
// @Failure      404           {object}  ErrorResponse          "Item no encontrado"
// @Failure      500           {object}  ErrorResponse          "Error interno del servidor - error de lectura o conexión a base de datos"
// @Failure      503           {object}  ErrorResponse          "Servicio no disponible - error de conexión al cache"
// @Router       /inventory/items/{id} [get]
func (h *InventoryHandler) GetItemByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	// Try cache first (if enabled)
	if h.cache != nil {
		cacheKey := cacheKeyItemByID(id.String())
		var cachedItem models.InventoryItem
		if err := cache.GetJSON(c.Request.Context(), h.cache, cacheKey, &cachedItem); err == nil {
			h.logger.Debug("Cache hit", zap.String("key", cacheKey))
			response := InventoryItemResponse{
				ID:          cachedItem.ID,
				SKU:         cachedItem.SKU,
				Name:        cachedItem.Name,
				Description: cachedItem.Description,
				Quantity:    cachedItem.Quantity,
				Reserved:    cachedItem.Reserved,
				Available:   cachedItem.Available,
				CreatedAt:   cachedItem.CreatedAt.Format(time.RFC3339),
				UpdatedAt:   cachedItem.UpdatedAt.Format(time.RFC3339),
			}
			c.JSON(http.StatusOK, response)
			return
		}
	}

	// Cache miss - fetch from repository
	item, err := h.repository.FindByID(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrItemNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
			return
		}
		h.logger.Error("Failed to find item", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get item"})
		return
	}

	response := InventoryItemResponse{
		ID:          item.ID,
		SKU:         item.SKU,
		Name:        item.Name,
		Description: item.Description,
		Quantity:    item.Quantity,
		Reserved:    item.Reserved,
		Available:   item.Available,
		CreatedAt:   item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   item.UpdatedAt.Format(time.RFC3339),
	}

	// Cache the response (if enabled)
	if h.cache != nil {
		cacheKey := cacheKeyItemByID(id.String())
		cache.SetJSON(c.Request.Context(), h.cache, cacheKey, item, cache.TTL(h.cacheTTL))
	}

	c.JSON(http.StatusOK, response)
}

// GetItemBySKU handles GET /api/v1/inventory/items/sku/:sku
// @Summary      Get inventory item by SKU
// @Description  Obtiene un item de inventario por su SKU. Optimizado para lectura rápida desde cache.
//
// **Características:**
// - Cache de resultados por SKU
// - Respuestas ultra-rápidas (cache hit)
// - Búsqueda optimizada
// - Cache-first strategy para baja latencia
//
// **Ejemplos válidos:**
// - Obtener item por SKU válido: `GET /api/v1/inventory/items/sku/SKU-001`
// - Obtener item por SKU con formato diferente: `GET /api/v1/inventory/items/sku/PROD-2024-ABC`
//
// **Ejemplos inválidos:**
// - SKU vacío: `GET /api/v1/inventory/items/sku/`
// - SKU no encontrado: `GET /api/v1/inventory/items/sku/NONEXISTENT-SKU`
//
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-Request-ID  header    string  false  "Request ID for request tracking (UUID). If not provided, a new one will be generated."
// @Param        sku           path      string  true   "SKU (Stock Keeping Unit)" example(SKU-001)
// @Success      200           {object}  InventoryItemResponse  "Item obtenido exitosamente"
// @Failure      400           {object}  ErrorResponse          "SKU inválido - SKU vacío"
// @Failure      401           {object}  ErrorResponse          "No autorizado - token JWT inválido o faltante"
// @Failure      404           {object}  ErrorResponse          "Item no encontrado"
// @Failure      500           {object}  ErrorResponse          "Error interno del servidor - error de lectura o conexión a base de datos"
// @Failure      503           {object}  ErrorResponse          "Servicio no disponible - error de conexión al cache"
// @Router       /inventory/items/sku/{sku} [get]
func (h *InventoryHandler) GetItemBySKU(c *gin.Context) {
	sku := c.Param("sku")
	if sku == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sku is required"})
		return
	}

	// Try cache first (if enabled)
	if h.cache != nil {
		cacheKey := cacheKeyItemBySKU(sku)
		var cachedItem models.InventoryItem
		if err := cache.GetJSON(c.Request.Context(), h.cache, cacheKey, &cachedItem); err == nil {
			h.logger.Debug("Cache hit", zap.String("key", cacheKey))
			response := InventoryItemResponse{
				ID:          cachedItem.ID,
				SKU:         cachedItem.SKU,
				Name:        cachedItem.Name,
				Description: cachedItem.Description,
				Quantity:    cachedItem.Quantity,
				Reserved:    cachedItem.Reserved,
				Available:   cachedItem.Available,
				CreatedAt:   cachedItem.CreatedAt.Format(time.RFC3339),
				UpdatedAt:   cachedItem.UpdatedAt.Format(time.RFC3339),
			}
			c.JSON(http.StatusOK, response)
			return
		}
	}

	// Cache miss - fetch from repository
	item, err := h.repository.FindBySKU(c.Request.Context(), sku)
	if err != nil {
		if err == repository.ErrItemNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
			return
		}
		h.logger.Error("Failed to find item by SKU", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get item"})
		return
	}

	response := InventoryItemResponse{
		ID:          item.ID,
		SKU:         item.SKU,
		Name:        item.Name,
		Description: item.Description,
		Quantity:    item.Quantity,
		Reserved:    item.Reserved,
		Available:   item.Available,
		CreatedAt:   item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   item.UpdatedAt.Format(time.RFC3339),
	}

	// Cache the response (if enabled)
	if h.cache != nil {
		cacheKey := cacheKeyItemBySKU(sku)
		cache.SetJSON(c.Request.Context(), h.cache, cacheKey, item, cache.TTL(h.cacheTTL))
	}

	c.JSON(http.StatusOK, response)
}

// GetStockStatus handles GET /api/v1/inventory/items/:id/stock
// @Summary      Get stock status
// @Description  Obtiene el estado de stock de un item (cantidad total, reservada y disponible). Optimizado para lectura rápida desde cache.
//
// **Características:**
// - Cache de estado de stock (TTL más corto para datos frecuentemente actualizados)
// - Respuestas ultra-rápidas (cache hit)
// - Ideal para consultas frecuentes de disponibilidad
// - Cache-first strategy para baja latencia
//
// **Ejemplos válidos:**
// - Obtener estado de stock por ID válido: `GET /api/v1/inventory/items/550e8400-e29b-41d4-a716-446655440000/stock`
//
// **Ejemplos inválidos:**
// - ID inválido (UUID malformado): `GET /api/v1/inventory/items/invalid-id/stock`
// - ID no encontrado: `GET /api/v1/inventory/items/550e8400-e29b-41d4-a716-446655440000/stock` (si no existe)
//
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-Request-ID  header    string  false  "Request ID for request tracking (UUID). If not provided, a new one will be generated."
// @Param        id            path      string  true   "Item ID (UUID)" example(550e8400-e29b-41d4-a716-446655440000)
// @Success      200           {object}  StockStatusResponse  "Estado de stock obtenido exitosamente"
// @Failure      400           {object}  ErrorResponse         "ID inválido - UUID malformado"
// @Failure      401           {object}  ErrorResponse         "No autorizado - token JWT inválido o faltante"
// @Failure      404           {object}  ErrorResponse         "Item no encontrado"
// @Failure      500           {object}  ErrorResponse         "Error interno del servidor - error de lectura o conexión a base de datos"
// @Failure      503           {object}  ErrorResponse         "Servicio no disponible - error de conexión al cache"
// @Router       /inventory/items/{id}/stock [get]
func (h *InventoryHandler) GetStockStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	// Try cache first (if enabled)
	if h.cache != nil {
		cacheKey := cacheKeyStockStatus(id.String())
		var cachedStatus models.StockStatus
		if err := cache.GetJSON(c.Request.Context(), h.cache, cacheKey, &cachedStatus); err == nil {
			h.logger.Debug("Cache hit", zap.String("key", cacheKey))
			response := StockStatusResponse{
				ID:        cachedStatus.ID,
				SKU:       cachedStatus.SKU,
				Quantity:  cachedStatus.Quantity,
				Reserved:  cachedStatus.Reserved,
				Available: cachedStatus.Available,
				UpdatedAt: cachedStatus.UpdatedAt.Format(time.RFC3339),
			}
			c.JSON(http.StatusOK, response)
			return
		}
	}

	// Cache miss - fetch from repository
	status, err := h.repository.GetStockStatus(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrItemNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
			return
		}
		h.logger.Error("Failed to get stock status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get stock status"})
		return
	}

	response := StockStatusResponse{
		ID:        status.ID,
		SKU:       status.SKU,
		Quantity:  status.Quantity,
		Reserved:  status.Reserved,
		Available: status.Available,
		UpdatedAt: status.UpdatedAt.Format(time.RFC3339),
	}

	// Cache the response (if enabled, shorter TTL for stock status as it changes frequently)
	if h.cache != nil {
		cacheKey := cacheKeyStockStatus(id.String())
		cache.SetJSON(c.Request.Context(), h.cache, cacheKey, status, cache.TTL(h.cacheTTL/2))
	}

	c.JSON(http.StatusOK, response)
}

// Cache key helpers
func cacheKeyItemByID(id string) string {
	return "item:id:" + id
}

func cacheKeyItemBySKU(sku string) string {
	return "item:sku:" + sku
}

func cacheKeyStockStatus(id string) string {
	return "stock:" + id
}

func cacheKeyListItems(page, pageSize int) string {
	return "items:list:" + strconv.Itoa(page) + ":" + strconv.Itoa(pageSize)
}
