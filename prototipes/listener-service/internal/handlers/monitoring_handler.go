package handlers

import (
	"net/http"

	"listener-service/internal/database"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type MonitoringHandler struct {
	db     *database.SingleWriterDB
	logger *zap.Logger
}

func NewMonitoringHandler(db *database.SingleWriterDB, logger *zap.Logger) *MonitoringHandler {
	return &MonitoringHandler{
		db:     db,
		logger: logger,
	}
}

// GetStats godoc
// @Summary      Get service statistics
// @Description  Obtiene estadísticas del servicio incluyendo conteo de items, tiendas y reservas
// @Tags         monitoring
// @Accept       json
// @Produce      json
// @Success      200  {object}  StatsResponse  "Estadísticas del servicio"
// @Failure      500  {object}  ErrorResponse  "Error interno del servidor"
// @Router       /monitoring/stats [get]
func (h *MonitoringHandler) GetStats(c *gin.Context) {
	stats := make(map[string]interface{})

	// Get inventory items count
	var itemsCount int
	err := h.db.QueryRow("SELECT COUNT(*) FROM inventory_items").Scan(&itemsCount)
	if err != nil {
		h.logger.Error("Failed to get items count", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get statistics"})
		return
	}
	stats["inventory_items"] = itemsCount

	// Get stores count
	var storesCount int
	err = h.db.QueryRow("SELECT COUNT(*) FROM stores").Scan(&storesCount)
	if err != nil {
		h.logger.Error("Failed to get stores count", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get statistics"})
		return
	}
	stats["stores"] = storesCount

	// Get active reservations count
	var reservationsCount int
	err = h.db.QueryRow("SELECT COUNT(*) FROM store_reservations WHERE status = 'active'").Scan(&reservationsCount)
	if err != nil {
		h.logger.Error("Failed to get reservations count", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get statistics"})
		return
	}
	stats["active_reservations"] = reservationsCount

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"stats":  stats,
	})
}

// GetDatabaseStatus godoc
// @Summary      Get database status
// @Description  Obtiene el estado de la base de datos SQLite
// @Tags         monitoring
// @Accept       json
// @Produce      json
// @Success      200  {object}  DatabaseStatusResponse  "Estado de la base de datos"
// @Failure      500  {object}  ErrorResponse           "Error interno del servidor"
// @Router       /monitoring/database/status [get]
func (h *MonitoringHandler) GetDatabaseStatus(c *gin.Context) {
	// Test database connection
	err := h.db.Ping()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  "database connection failed",
		})
		return
	}

	response := DatabaseStatusResponse{
		Status: "ok",
	}
	response.Database.Connected = true
	response.Database.Type = "sqlite"

	c.JSON(http.StatusOK, response)
}

