package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"listener-service/internal/config"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

// SingleWriterDB implements Single Writer Principle for SQLite
// Only one writer can access the database at a time
type SingleWriterDB struct {
	db     *sql.DB
	logger *zap.Logger
	mu     sync.Mutex // Mutex to ensure single writer
}

// QueryRow executes a query that returns a single row
func (swdb *SingleWriterDB) QueryRow(query string, args ...interface{}) *sql.Row {
	return swdb.db.QueryRow(query, args...)
}

// Ping checks the database connection
func (swdb *SingleWriterDB) Ping() error {
	return swdb.db.Ping()
}

// NewSingleWriterDB creates a new database connection with single writer principle
func NewSingleWriterDB(cfg *config.Config, logger *zap.Logger) (*SingleWriterDB, error) {
	db, err := sql.Open("sqlite3", cfg.SQLitePath+"?_journal_mode=WAL&_foreign_keys=1")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(1) // Single writer
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	swdb := &SingleWriterDB{
		db:     db,
		logger: logger,
	}

	// Initialize schema
	if err := swdb.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return swdb, nil
}

// initSchema creates the database schema
func (swdb *SingleWriterDB) initSchema() error {
	schema := `
	-- Stores table: Information about physical stores
	CREATE TABLE IF NOT EXISTS stores (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		location TEXT,
		code TEXT UNIQUE NOT NULL,
		active INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		CHECK(active IN (0, 1))
	);

	-- Inventory items table: Centralized inventory (single source of truth)
	CREATE TABLE IF NOT EXISTS inventory_items (
		id TEXT PRIMARY KEY,
		sku TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		quantity INTEGER NOT NULL DEFAULT 0,
		reserved INTEGER NOT NULL DEFAULT 0,
		available INTEGER NOT NULL DEFAULT 0,
		version INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		CHECK(quantity >= 0),
		CHECK(reserved >= 0),
		CHECK(available >= 0),
		CHECK(reserved <= quantity),
		CHECK(available = quantity - reserved)
	);

	-- Store reservations table: Track reservations by store
	-- This allows us to know which store has reserved which items
	CREATE TABLE IF NOT EXISTS store_reservations (
		id TEXT PRIMARY KEY,
		store_id TEXT NOT NULL,
		item_id TEXT NOT NULL,
		quantity INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'active',
		reserved_at TEXT NOT NULL,
		released_at TEXT,
		expires_at TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		FOREIGN KEY (store_id) REFERENCES stores(id) ON DELETE CASCADE,
		FOREIGN KEY (item_id) REFERENCES inventory_items(id) ON DELETE CASCADE,
		CHECK(quantity > 0),
		CHECK(status IN ('active', 'released', 'expired', 'fulfilled'))
	);

	-- Indexes for performance
	CREATE INDEX IF NOT EXISTS idx_inventory_items_sku ON inventory_items(sku);
	CREATE INDEX IF NOT EXISTS idx_inventory_items_version ON inventory_items(version);
	CREATE INDEX IF NOT EXISTS idx_stores_code ON stores(code);
	CREATE INDEX IF NOT EXISTS idx_stores_active ON stores(active);
	CREATE INDEX IF NOT EXISTS idx_store_reservations_store_id ON store_reservations(store_id);
	CREATE INDEX IF NOT EXISTS idx_store_reservations_item_id ON store_reservations(item_id);
	CREATE INDEX IF NOT EXISTS idx_store_reservations_status ON store_reservations(status);
	CREATE INDEX IF NOT EXISTS idx_store_reservations_store_item ON store_reservations(store_id, item_id);
	`

	_, err := swdb.db.Exec(schema)
	return err
}

// Close closes the database connection
func (swdb *SingleWriterDB) Close() error {
	return swdb.db.Close()
}

// Store represents a physical store
type Store struct {
	ID        string
	Name      string
	Location  string
	Code      string
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// InventoryItem represents an inventory item in the database
type InventoryItem struct {
	ID          string
	SKU         string
	Name        string
	Description string
	Quantity    int
	Reserved    int
	Available   int
	Version     int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// StoreReservation represents a reservation of inventory by a store
type StoreReservation struct {
	ID         string
	StoreID    string
	ItemID     string
	Quantity   int
	Status     string // active, released, expired, fulfilled
	ReservedAt time.Time
	ReleasedAt *time.Time
	ExpiresAt  *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// CreateItem creates a new inventory item (Single Writer)
func (swdb *SingleWriterDB) CreateItem(ctx context.Context, item *InventoryItem) error {
	swdb.mu.Lock()
	defer swdb.mu.Unlock()

	query := `
		INSERT INTO inventory_items (id, sku, name, description, quantity, reserved, available, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1, ?, ?)
	`

	now := time.Now().UTC()
	available := item.Quantity - item.Reserved
	_, err := swdb.db.ExecContext(ctx, query,
		item.ID, item.SKU, item.Name, item.Description,
		item.Quantity, item.Reserved, available,
		now.Format(time.RFC3339), now.Format(time.RFC3339),
	)

	if err != nil {
		return fmt.Errorf("failed to create item: %w", err)
	}

	return nil
}

// UpdateItem updates an inventory item with optimistic locking
func (swdb *SingleWriterDB) UpdateItem(ctx context.Context, item *InventoryItem) error {
	swdb.mu.Lock()
	defer swdb.mu.Unlock()

	query := `
		UPDATE inventory_items
		SET name = ?, description = ?, version = version + 1, updated_at = ?
		WHERE id = ? AND version = ?
	`

	result, err := swdb.db.ExecContext(ctx, query,
		item.Name, item.Description,
		time.Now().UTC().Format(time.RFC3339),
		item.ID, item.Version,
	)

	if err != nil {
		return fmt.Errorf("failed to update item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrOptimisticLockFailed
	}

	return nil
}

// AdjustStock adjusts stock with optimistic locking
func (swdb *SingleWriterDB) AdjustStock(ctx context.Context, itemID string, adjustment int, expectedVersion int) error {
	swdb.mu.Lock()
	defer swdb.mu.Unlock()

	query := `
		UPDATE inventory_items
		SET quantity = quantity + ?, version = version + 1, updated_at = ?
		WHERE id = ? AND version = ? AND (quantity + ?) >= 0
	`

	result, err := swdb.db.ExecContext(ctx, query,
		adjustment,
		time.Now().UTC().Format(time.RFC3339),
		itemID, expectedVersion,
		adjustment,
	)

	if err != nil {
		return fmt.Errorf("failed to adjust stock: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrOptimisticLockFailed
	}

	return nil
}

// ReserveStock reserves stock with optimistic locking
func (swdb *SingleWriterDB) ReserveStock(ctx context.Context, itemID string, quantity int, expectedVersion int) error {
	swdb.mu.Lock()
	defer swdb.mu.Unlock()

	query := `
		UPDATE inventory_items
		SET reserved = reserved + ?,
		    available = quantity - (reserved + ?),
		    version = version + 1,
		    updated_at = ?
		WHERE id = ? AND version = ? AND (quantity - reserved - ?) >= 0
	`

	result, err := swdb.db.ExecContext(ctx, query,
		quantity,
		quantity,
		time.Now().UTC().Format(time.RFC3339),
		itemID, expectedVersion,
		quantity,
	)

	if err != nil {
		return fmt.Errorf("failed to reserve stock: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrOptimisticLockFailed
	}

	return nil
}

// ReleaseStock releases reserved stock with optimistic locking
func (swdb *SingleWriterDB) ReleaseStock(ctx context.Context, itemID string, quantity int, expectedVersion int) error {
	swdb.mu.Lock()
	defer swdb.mu.Unlock()

	query := `
		UPDATE inventory_items
		SET reserved = reserved - ?,
		    available = quantity - (reserved - ?),
		    version = version + 1,
		    updated_at = ?
		WHERE id = ? AND version = ? AND reserved >= ?
	`

	result, err := swdb.db.ExecContext(ctx, query,
		quantity,
		quantity,
		time.Now().UTC().Format(time.RFC3339),
		itemID, expectedVersion,
		quantity,
	)

	if err != nil {
		return fmt.Errorf("failed to release stock: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrOptimisticLockFailed
	}

	return nil
}

// DeleteItem deletes an inventory item
func (swdb *SingleWriterDB) DeleteItem(ctx context.Context, itemID string) error {
	swdb.mu.Lock()
	defer swdb.mu.Unlock()

	query := `DELETE FROM inventory_items WHERE id = ?`

	_, err := swdb.db.ExecContext(ctx, query, itemID)
	if err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
	}

	return nil
}

// GetItem retrieves an item by ID (read-only, no lock needed)
func (swdb *SingleWriterDB) GetItem(ctx context.Context, itemID string) (*InventoryItem, error) {
	query := `
		SELECT id, sku, name, description, quantity, reserved, available, version, created_at, updated_at
		FROM inventory_items
		WHERE id = ?
	`

	var item InventoryItem
	var createdAtStr, updatedAtStr string

	err := swdb.db.QueryRowContext(ctx, query, itemID).Scan(
		&item.ID, &item.SKU, &item.Name, &item.Description,
		&item.Quantity, &item.Reserved, &item.Available, &item.Version,
		&createdAtStr, &updatedAtStr,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	item.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	item.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

	return &item, nil
}

// CreateStore creates a new store
func (swdb *SingleWriterDB) CreateStore(ctx context.Context, store *Store) error {
	swdb.mu.Lock()
	defer swdb.mu.Unlock()

	query := `
		INSERT INTO stores (id, name, location, code, active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now().UTC()
	active := 0
	if store.Active {
		active = 1
	}

	_, err := swdb.db.ExecContext(ctx, query,
		store.ID, store.Name, store.Location, store.Code, active,
		now.Format(time.RFC3339), now.Format(time.RFC3339),
	)

	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}

	return nil
}

// GetStore retrieves a store by ID
func (swdb *SingleWriterDB) GetStore(ctx context.Context, storeID string) (*Store, error) {
	query := `
		SELECT id, name, location, code, active, created_at, updated_at
		FROM stores
		WHERE id = ?
	`

	var store Store
	var createdAtStr, updatedAtStr string
	var active int

	err := swdb.db.QueryRowContext(ctx, query, storeID).Scan(
		&store.ID, &store.Name, &store.Location, &store.Code, &active,
		&createdAtStr, &updatedAtStr,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrStoreNotFound
		}
		return nil, fmt.Errorf("failed to get store: %w", err)
	}

	store.Active = active == 1
	store.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	store.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

	return &store, nil
}

// CreateStoreReservation creates a reservation of inventory by a store
func (swdb *SingleWriterDB) CreateStoreReservation(ctx context.Context, reservation *StoreReservation) error {
	swdb.mu.Lock()
	defer swdb.mu.Unlock()

	query := `
		INSERT INTO store_reservations (id, store_id, item_id, quantity, status, reserved_at, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now().UTC()
	var expiresAtStr string
	if reservation.ExpiresAt != nil {
		expiresAtStr = reservation.ExpiresAt.Format(time.RFC3339)
	}

	_, err := swdb.db.ExecContext(ctx, query,
		reservation.ID, reservation.StoreID, reservation.ItemID, reservation.Quantity,
		reservation.Status, reservation.ReservedAt.Format(time.RFC3339),
		expiresAtStr, now.Format(time.RFC3339), now.Format(time.RFC3339),
	)

	if err != nil {
		return fmt.Errorf("failed to create store reservation: %w", err)
	}

	return nil
}

// GetStoreReservations retrieves active reservations for a store
func (swdb *SingleWriterDB) GetStoreReservations(ctx context.Context, storeID string) ([]*StoreReservation, error) {
	query := `
		SELECT id, store_id, item_id, quantity, status, reserved_at, released_at, expires_at, created_at, updated_at
		FROM store_reservations
		WHERE store_id = ? AND status = 'active'
		ORDER BY reserved_at DESC
	`

	rows, err := swdb.db.QueryContext(ctx, query, storeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get store reservations: %w", err)
	}
	defer rows.Close()

	var reservations []*StoreReservation
	for rows.Next() {
		var res StoreReservation
		var reservedAtStr, createdAtStr, updatedAtStr string
		var releasedAtStr, expiresAtStr sql.NullString

		err := rows.Scan(
			&res.ID, &res.StoreID, &res.ItemID, &res.Quantity, &res.Status,
			&reservedAtStr, &releasedAtStr, &expiresAtStr,
			&createdAtStr, &updatedAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reservation: %w", err)
		}

		res.ReservedAt, _ = time.Parse(time.RFC3339, reservedAtStr)
		res.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		res.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

		if releasedAtStr.Valid {
			releasedAt, _ := time.Parse(time.RFC3339, releasedAtStr.String)
			res.ReleasedAt = &releasedAt
		}

		if expiresAtStr.Valid {
			expiresAt, _ := time.Parse(time.RFC3339, expiresAtStr.String)
			res.ExpiresAt = &expiresAt
		}

		reservations = append(reservations, &res)
	}

	return reservations, nil
}

var (
	ErrItemNotFound         = errors.New("item not found")
	ErrStoreNotFound        = errors.New("store not found")
	ErrOptimisticLockFailed = errors.New("optimistic lock failed - version mismatch or constraint violation")
)
