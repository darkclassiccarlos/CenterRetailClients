package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"query-service/internal/models"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

// SQLiteReadRepository reads from SQLite database (Read Model)
// This database is updated by the Listener Service
type SQLiteReadRepository struct {
	db *sql.DB
}

// NewSQLiteReadRepository creates a new SQLite read repository
func NewSQLiteReadRepository(dbPath string) (ReadRepository, error) {
	// Open database in read-only mode for Query Service
	// Use WAL mode for better read concurrency
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&mode=ro")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool for read operations
	db.SetMaxOpenConns(10) // Multiple readers allowed
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &SQLiteReadRepository{
		db: db,
	}, nil
}

// Close closes the database connection
func (r *SQLiteReadRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// FindByID finds an item by ID
func (r *SQLiteReadRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.InventoryItem, error) {
	query := `
		SELECT id, sku, name, description, quantity, reserved, available, created_at, updated_at
		FROM inventory_items
		WHERE id = ?
	`

	var item models.InventoryItem
	var createdAtStr, updatedAtStr string

	err := r.db.QueryRowContext(ctx, query, id.String()).Scan(
		&item.ID,
		&item.SKU,
		&item.Name,
		&item.Description,
		&item.Quantity,
		&item.Reserved,
		&item.Available,
		&createdAtStr,
		&updatedAtStr,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to find item by ID: %w", err)
	}

	// Parse timestamps
	if createdAt, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
		item.CreatedAt = createdAt
	}
	if updatedAt, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
		item.UpdatedAt = updatedAt
	}

	return &item, nil
}

// FindBySKU finds an item by SKU
func (r *SQLiteReadRepository) FindBySKU(ctx context.Context, sku string) (*models.InventoryItem, error) {
	query := `
		SELECT id, sku, name, description, quantity, reserved, available, created_at, updated_at
		FROM inventory_items
		WHERE sku = ?
	`

	var item models.InventoryItem
	var createdAtStr, updatedAtStr string

	err := r.db.QueryRowContext(ctx, query, sku).Scan(
		&item.ID,
		&item.SKU,
		&item.Name,
		&item.Description,
		&item.Quantity,
		&item.Reserved,
		&item.Available,
		&createdAtStr,
		&updatedAtStr,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to find item by SKU: %w", err)
	}

	// Parse timestamps
	if createdAt, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
		item.CreatedAt = createdAt
	}
	if updatedAt, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
		item.UpdatedAt = updatedAt
	}

	return &item, nil
}

// ListItems lists items with pagination
func (r *SQLiteReadRepository) ListItems(ctx context.Context, page, pageSize int) ([]models.InventoryItem, int, error) {
	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM inventory_items`
	err := r.db.QueryRowContext(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count items: %w", err)
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Get items with pagination
	query := `
		SELECT id, sku, name, description, quantity, reserved, available, created_at, updated_at
		FROM inventory_items
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.QueryContext(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list items: %w", err)
	}
	defer rows.Close()

	items := make([]models.InventoryItem, 0)
	for rows.Next() {
		var item models.InventoryItem
		var createdAtStr, updatedAtStr string

		err := rows.Scan(
			&item.ID,
			&item.SKU,
			&item.Name,
			&item.Description,
			&item.Quantity,
			&item.Reserved,
			&item.Available,
			&createdAtStr,
			&updatedAtStr,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan item: %w", err)
		}

		// Parse timestamps
		if createdAt, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			item.CreatedAt = createdAt
		}
		if updatedAt, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
			item.UpdatedAt = updatedAt
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating items: %w", err)
	}

	return items, total, nil
}

// GetStockStatus gets stock status for an item
func (r *SQLiteReadRepository) GetStockStatus(ctx context.Context, id uuid.UUID) (*models.StockStatus, error) {
	query := `
		SELECT id, sku, quantity, reserved, available, updated_at
		FROM inventory_items
		WHERE id = ?
	`

	var status models.StockStatus
	var updatedAtStr string

	err := r.db.QueryRowContext(ctx, query, id.String()).Scan(
		&status.ID,
		&status.SKU,
		&status.Quantity,
		&status.Reserved,
		&status.Available,
		&updatedAtStr,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to get stock status: %w", err)
	}

	// Parse timestamp
	if updatedAt, err := time.Parse(time.RFC3339, updatedAtStr); err == nil {
		status.UpdatedAt = updatedAt
	}

	return &status, nil
}
