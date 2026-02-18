package handlers

import (
	"net/http"

	"command-service/internal/commands"
	"command-service/internal/config"
	"command-service/internal/domain"
	"command-service/internal/events"
	"command-service/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type InventoryHandler struct {
	logger     *zap.Logger
	repository repository.InventoryRepository
	eventBus   events.EventPublisher
}

func NewInventoryHandler(logger *zap.Logger, cfg *config.Config) *InventoryHandler {
	// TODO: Initialize repository with actual implementations
	repo := repository.NewInventoryRepository() // Placeholder

	// Initialize Kafka event publisher
	eventBus, err := events.NewKafkaEventPublisher(cfg, logger)
	if err != nil {
		logger.Warn("Failed to initialize Kafka publisher, using in-memory fallback", zap.Error(err))
		eventBus = events.NewEventPublisher() // Fallback to in-memory
	}

	return &InventoryHandler{
		logger:     logger,
		repository: repo,
		eventBus:   eventBus,
	}
}

// CreateItem handles POST /api/v1/inventory/items
// @Summary      Create a new inventory item
// @Description  Crea un nuevo item en el inventario. El SKU debe ser único y la cantidad inicial debe ser >= 0.
// @Description  **Idempotencia**: Incluye X-Request-ID en el header para evitar duplicados. Si se envía el mismo X-Request-ID, se retornará la respuesta cacheada (válida por 5 minutos).
//
// **Ejemplos válidos:**
// - Request completo con todos los campos
// - Request con descripción opcional vacía
// - Request con cantidad inicial 0
//
// **Ejemplos inválidos:**
// - Campos requeridos faltantes (sku, name, quantity)
// - Cantidad negativa
// - SKU vacío
//
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-Request-ID  header    string  false  "Request ID for idempotency (UUID). If not provided, a new one will be generated."
// @Param        request      body      CreateItemRequest  true  "Item creation request"
// @Success      201          {object}  CreateItemResponse  "Item creado exitosamente"
// @Success      200          {object}  CreateItemResponse  "Request duplicado - respuesta cacheada (idempotencia)"
// @Failure      400          {object}  ErrorResponse       "Request inválido - campos requeridos faltantes o valores inválidos"
// @Failure      401          {object}  ErrorResponse       "No autorizado - token JWT inválido o faltante"
// @Failure      409          {object}  ErrorResponse       "Conflicto - SKU duplicado"
// @Failure      500          {object}  ErrorResponse       "Error interno del servidor - error de persistencia o conexión a base de datos"
// @Failure      503          {object}  ErrorResponse       "Servicio no disponible - error de conexión al event broker"
// @Router       /inventory/items [post]
func (h *InventoryHandler) CreateItem(c *gin.Context) {
	var req struct {
		SKU         string `json:"sku" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Quantity    int    `json:"quantity" binding:"required,min=0"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create command
	cmd := commands.CreateItemCommand{
		SKU:         req.SKU,
		Name:        req.Name,
		Description: req.Description,
		Quantity:    req.Quantity,
	}

	// Execute command
	item := domain.NewInventoryItem(cmd.SKU, cmd.Name, cmd.Description, cmd.Quantity)

	// Save to repository
	if err := h.repository.Save(c.Request.Context(), item); err != nil {
		h.logger.Error("Failed to save item", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create item"})
		return
	}

	// Publish event
	event := events.InventoryItemCreatedEvent{
		ItemID:      item.ID,
		SKU:         item.SKU,
		Name:        item.Name,
		Description: item.Description,
		Quantity:    item.Quantity,
		OccurredAt:  item.CreatedAt,
	}
	if err := h.eventBus.Publish(c.Request.Context(), event); err != nil {
		h.logger.Error("Failed to publish event", zap.Error(err))
		// Note: In production, you might want to handle this differently
	}

	h.logger.Info("Item created", zap.String("item_id", item.ID.String()))
	c.JSON(http.StatusCreated, gin.H{
		"id":          item.ID,
		"sku":         item.SKU,
		"name":        item.Name,
		"description": item.Description,
		"quantity":    item.Quantity,
		"created_at":  item.CreatedAt,
	})
}

// UpdateItem handles PUT /api/v1/inventory/items/:id
// @Summary      Update an inventory item
// @Description  Actualiza un item existente en el inventario. Solo se pueden actualizar el nombre y la descripción.
//
// **Ejemplos válidos:**
// - Actualizar nombre y descripción
// - Actualizar solo el nombre (descripción opcional)
//
// **Ejemplos inválidos:**
// - Nombre faltante (campo requerido)
// - ID inválido (UUID malformado)
// - Item no encontrado (ID válido pero no existe)
//
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-Request-ID  header    string  false  "Request ID for idempotency (UUID). If not provided, a new one will be generated."
// @Param        id            path      string  true   "Item ID (UUID)" example(550e8400-e29b-41d4-a716-446655440000)
// @Param        request       body      UpdateItemRequest  true  "Item update request"
// @Success      200           {object}  UpdateItemResponse  "Item actualizado exitosamente"
// @Success      200           {object}  UpdateItemResponse  "Request duplicado - respuesta cacheada (idempotencia)"
// @Failure      400           {object}  ErrorResponse      "Request inválido - ID inválido o campos requeridos faltantes"
// @Failure      401           {object}  ErrorResponse      "No autorizado - token JWT inválido o faltante"
// @Failure      404           {object}  ErrorResponse      "Item no encontrado"
// @Failure      500           {object}  ErrorResponse      "Error interno del servidor - error de persistencia o conexión a base de datos"
// @Failure      503           {object}  ErrorResponse      "Servicio no disponible - error de conexión al event broker"
// @Router       /inventory/items/{id} [put]
func (h *InventoryHandler) UpdateItem(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get item from repository
	item, err := h.repository.FindByID(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrItemNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
			return
		}
		h.logger.Error("Failed to find item", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update item"})
		return
	}

	// Update item
	item.Name = req.Name
	item.Description = req.Description

	// Save changes
	if err := h.repository.Save(c.Request.Context(), item); err != nil {
		h.logger.Error("Failed to save item", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update item"})
		return
	}

	// Publish event
	event := events.InventoryItemUpdatedEvent{
		ItemID:      item.ID,
		Name:        item.Name,
		Description: item.Description,
		OccurredAt:  item.UpdatedAt,
	}
	if err := h.eventBus.Publish(c.Request.Context(), event); err != nil {
		h.logger.Error("Failed to publish event", zap.Error(err))
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          item.ID,
		"sku":         item.SKU,
		"name":        item.Name,
		"description": item.Description,
		"quantity":    item.Quantity,
		"updated_at":  item.UpdatedAt,
	})
}

// DeleteItem handles DELETE /api/v1/inventory/items/:id
// @Summary      Delete an inventory item
// @Description  Elimina un item del inventario. Esta operación no se puede deshacer.
//
// **Ejemplos válidos:**
// - DELETE con ID válido existente
//
// **Ejemplos inválidos:**
// - ID inválido (UUID malformado)
// - Item no encontrado (ID válido pero no existe)
//
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        X-Request-ID  header    string  false  "Request ID for idempotency (UUID). If not provided, a new one will be generated."
// @Param        id           path      string  true   "Item ID (UUID)" example(550e8400-e29b-41d4-a716-446655440000)
// @Success      200          {object}  SuccessResponse  "Item eliminado exitosamente"
// @Success      200          {object}  SuccessResponse  "Request duplicado - respuesta cacheada (idempotencia)"
// @Failure      400          {object}  ErrorResponse    "ID inválido"
// @Failure      401          {object}  ErrorResponse    "No autorizado - token JWT inválido o faltante"
// @Failure      404          {object}  ErrorResponse    "Item no encontrado"
// @Failure      500          {object}  ErrorResponse    "Error interno del servidor - error de persistencia o conexión a base de datos"
// @Failure      503          {object}  ErrorResponse    "Servicio no disponible - error de conexión al event broker"
// @Router       /inventory/items/{id} [delete]
func (h *InventoryHandler) DeleteItem(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	// Get item to verify it exists
	item, err := h.repository.FindByID(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrItemNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
			return
		}
		h.logger.Error("Failed to find item", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete item"})
		return
	}

	// Delete from repository
	if err := h.repository.Delete(c.Request.Context(), id); err != nil {
		h.logger.Error("Failed to delete item", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete item"})
		return
	}

	// Publish event
	event := events.InventoryItemDeletedEvent{
		ItemID:     item.ID,
		SKU:        item.SKU,
		OccurredAt: item.UpdatedAt,
	}
	if err := h.eventBus.Publish(c.Request.Context(), event); err != nil {
		h.logger.Error("Failed to publish event", zap.Error(err))
	}

	c.JSON(http.StatusOK, gin.H{"message": "item deleted successfully"})
}

// AdjustStock handles POST /api/v1/inventory/items/:id/adjust
// @Summary      Adjust stock quantity
// @Description  Ajusta la cantidad de stock de un item. Valores positivos aumentan el stock, valores negativos lo disminuyen. No se puede ajustar a un valor negativo total.
//
// **Ejemplos válidos:**
// - Aumentar stock: `{"quantity": 10}`
// - Disminuir stock: `{"quantity": -5}` (siempre que el resultado sea >= 0)
//
// **Ejemplos inválidos:**
// - Cantidad faltante
// - Ajuste que resultaría en stock negativo
// - ID inválido o item no encontrado
//
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string              true  "Item ID (UUID)" example(550e8400-e29b-41d4-a716-446655440000)
// @Param        request  body      AdjustStockRequest  true  "Stock adjustment request"
// @Success      200      {object}  StockResponse       "Stock ajustado exitosamente"
// @Failure      400      {object}  ErrorResponse       "Request inválido - ID inválido, cantidad faltante o stock insuficiente"
// @Failure      401      {object}  ErrorResponse       "No autorizado - token JWT inválido o faltante"
// @Failure      404      {object}  ErrorResponse       "Item no encontrado"
// @Failure      500      {object}  ErrorResponse       "Error interno del servidor - error de persistencia o conexión a base de datos"
// @Failure      503      {object}  ErrorResponse       "Servicio no disponible - error de conexión al event broker"
// @Router       /inventory/items/{id}/adjust [post]
func (h *InventoryHandler) AdjustStock(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	var req struct {
		Quantity int `json:"quantity" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get item from repository
	item, err := h.repository.FindByID(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrItemNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
			return
		}
		h.logger.Error("Failed to find item", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to adjust stock"})
		return
	}

	// Adjust stock
	if err := item.AdjustStock(req.Quantity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Save changes
	if err := h.repository.Save(c.Request.Context(), item); err != nil {
		h.logger.Error("Failed to save item", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to adjust stock"})
		return
	}

	// Publish event
	event := events.StockAdjustedEvent{
		ItemID:     item.ID,
		SKU:        item.SKU,
		Quantity:   req.Quantity,
		NewTotal:   item.Quantity,
		OccurredAt: item.UpdatedAt,
	}
	if err := h.eventBus.Publish(c.Request.Context(), event); err != nil {
		h.logger.Error("Failed to publish event", zap.Error(err))
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         item.ID,
		"quantity":   item.Quantity,
		"available":  item.AvailableQuantity(),
		"reserved":   item.Reserved,
		"updated_at": item.UpdatedAt,
	})
}

// ReserveStock handles POST /api/v1/inventory/items/:id/reserve
// @Summary      Reserve stock
// @Description  Reserva una cantidad de stock de un item. La cantidad reservada no puede exceder el stock disponible.
//
// **Ejemplos válidos:**
// - Reservar cantidad disponible: `{"quantity": 5}`
//
// **Ejemplos inválidos:**
// - Cantidad faltante
// - Cantidad menor a 1
// - Stock insuficiente (cantidad > disponible)
// - ID inválido o item no encontrado
//
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string               true  "Item ID (UUID)" example(550e8400-e29b-41d4-a716-446655440000)
// @Param        request  body      ReserveStockRequest  true  "Stock reservation request"
// @Success      200      {object}  StockResponse       "Stock reservado exitosamente"
// @Failure      400      {object}  ErrorResponse       "Request inválido - ID inválido, cantidad inválida o stock insuficiente"
// @Failure      401      {object}  ErrorResponse       "No autorizado - token JWT inválido o faltante"
// @Failure      404      {object}  ErrorResponse       "Item no encontrado"
// @Failure      500      {object}  ErrorResponse       "Error interno del servidor - error de persistencia o conexión a base de datos"
// @Failure      503      {object}  ErrorResponse       "Servicio no disponible - error de conexión al event broker"
// @Router       /inventory/items/{id}/reserve [post]
func (h *InventoryHandler) ReserveStock(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	var req struct {
		Quantity int `json:"quantity" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get item from repository
	item, err := h.repository.FindByID(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrItemNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
			return
		}
		h.logger.Error("Failed to find item", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reserve stock"})
		return
	}

	// Reserve stock
	if err := item.ReserveStock(req.Quantity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Save changes
	if err := h.repository.Save(c.Request.Context(), item); err != nil {
		h.logger.Error("Failed to save item", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reserve stock"})
		return
	}

	// Publish event
	event := events.StockReservedEvent{
		ItemID:     item.ID,
		SKU:        item.SKU,
		Quantity:   req.Quantity,
		Reserved:   item.Reserved,
		Available:  item.AvailableQuantity(),
		OccurredAt: item.UpdatedAt,
	}
	if err := h.eventBus.Publish(c.Request.Context(), event); err != nil {
		h.logger.Error("Failed to publish event", zap.Error(err))
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         item.ID,
		"quantity":   item.Quantity,
		"available":  item.AvailableQuantity(),
		"reserved":   item.Reserved,
		"updated_at": item.UpdatedAt,
	})
}

// ReleaseStock handles POST /api/v1/inventory/items/:id/release
// @Summary      Release reserved stock
// @Description  Libera stock previamente reservado de un item. La cantidad a liberar no puede exceder la cantidad reservada.
//
// **Ejemplos válidos:**
// - Liberar cantidad reservada: `{"quantity": 5}`
//
// **Ejemplos inválidos:**
// - Cantidad faltante
// - Cantidad menor a 1
// - Cantidad excede lo reservado
// - ID inválido o item no encontrado
//
// @Tags         inventory
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      string               true  "Item ID (UUID)" example(550e8400-e29b-41d4-a716-446655440000)
// @Param        request  body      ReleaseStockRequest  true  "Stock release request"
// @Success      200      {object}  StockResponse       "Stock liberado exitosamente"
// @Failure      400      {object}  ErrorResponse       "Request inválido - ID inválido, cantidad inválida o cantidad a liberar excede lo reservado"
// @Failure      401      {object}  ErrorResponse       "No autorizado - token JWT inválido o faltante"
// @Failure      404      {object}  ErrorResponse       "Item no encontrado"
// @Failure      500      {object}  ErrorResponse       "Error interno del servidor - error de persistencia o conexión a base de datos"
// @Failure      503      {object}  ErrorResponse       "Servicio no disponible - error de conexión al event broker"
// @Router       /inventory/items/{id}/release [post]
func (h *InventoryHandler) ReleaseStock(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	var req struct {
		Quantity int `json:"quantity" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get item from repository
	item, err := h.repository.FindByID(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrItemNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
			return
		}
		h.logger.Error("Failed to find item", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to release stock"})
		return
	}

	// Release stock
	if err := item.ReleaseStock(req.Quantity); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Save changes
	if err := h.repository.Save(c.Request.Context(), item); err != nil {
		h.logger.Error("Failed to save item", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to release stock"})
		return
	}

	// Publish event
	event := events.StockReleasedEvent{
		ItemID:     item.ID,
		SKU:        item.SKU,
		Quantity:   req.Quantity,
		Reserved:   item.Reserved,
		Available:  item.AvailableQuantity(),
		OccurredAt: item.UpdatedAt,
	}
	if err := h.eventBus.Publish(c.Request.Context(), event); err != nil {
		h.logger.Error("Failed to publish event", zap.Error(err))
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         item.ID,
		"quantity":   item.Quantity,
		"available":  item.AvailableQuantity(),
		"reserved":   item.Reserved,
		"updated_at": item.UpdatedAt,
	})
}
