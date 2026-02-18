# Esquema de Base de Datos SQLite - Listener Service

Este documento describe el esquema de la base de datos SQLite que actúa como **única fuente de verdad** (Single Source of Truth) para el inventario centralizado.

## 📋 Arquitectura

El inventario está centralizado en una base de datos SQLite que es la **única fuente autorizada** para realizar cambios en el stock. Las tiendas ya no escriben localmente y luego sincronizan; ahora deben llamar a la API central (Command Service) para reservar/descontar stock.

## 🗄️ Esquema de Base de Datos

### Tabla: `stores`

Información sobre las tiendas físicas.

```sql
CREATE TABLE stores (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    location TEXT,
    code TEXT UNIQUE NOT NULL,
    active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CHECK(active IN (0, 1))
);
```

**Campos:**
- `id`: Identificador único de la tienda (UUID)
- `name`: Nombre de la tienda
- `location`: Ubicación de la tienda
- `code`: Código único de la tienda (para identificación rápida)
- `active`: Estado activo/inactivo (1 = activo, 0 = inactivo)
- `created_at`: Fecha de creación (ISO 8601)
- `updated_at`: Fecha de última actualización (ISO 8601)

**Índices:**
- `idx_stores_code`: Índice único en `code`
- `idx_stores_active`: Índice en `active` para consultas rápidas

### Tabla: `inventory_items`

Inventario centralizado (Single Source of Truth).

```sql
CREATE TABLE inventory_items (
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
```

**Campos:**
- `id`: Identificador único del item (UUID)
- `sku`: Stock Keeping Unit (único)
- `name`: Nombre del item
- `description`: Descripción del item
- `quantity`: Cantidad total en inventario
- `reserved`: Cantidad reservada por tiendas
- `available`: Cantidad disponible (calculada: quantity - reserved)
- `version`: Versión para optimistic locking
- `created_at`: Fecha de creación (ISO 8601)
- `updated_at`: Fecha de última actualización (ISO 8601)

**Constraints:**
- `quantity >= 0`: La cantidad no puede ser negativa
- `reserved >= 0`: Las reservas no pueden ser negativas
- `available >= 0`: Lo disponible no puede ser negativo
- `reserved <= quantity`: Las reservas no pueden exceder la cantidad total
- `available = quantity - reserved`: Lo disponible debe ser consistente

**Índices:**
- `idx_inventory_items_sku`: Índice único en `sku`
- `idx_inventory_items_version`: Índice en `version` para optimistic locking

### Tabla: `store_reservations`

Seguimiento de reservas por tienda (para auditoría y tracking).

```sql
CREATE TABLE store_reservations (
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
```

**Campos:**
- `id`: Identificador único de la reserva (UUID)
- `store_id`: ID de la tienda (FK a `stores`)
- `item_id`: ID del item (FK a `inventory_items`)
- `quantity`: Cantidad reservada
- `status`: Estado de la reserva (`active`, `released`, `expired`, `fulfilled`)
- `reserved_at`: Fecha/hora de la reserva (ISO 8601)
- `released_at`: Fecha/hora de liberación (opcional)
- `expires_at`: Fecha/hora de expiración (opcional)
- `created_at`: Fecha de creación (ISO 8601)
- `updated_at`: Fecha de última actualización (ISO 8601)

**Foreign Keys:**
- `store_id` → `stores(id)`: ON DELETE CASCADE
- `item_id` → `inventory_items(id)`: ON DELETE CASCADE

**Constraints:**
- `quantity > 0`: La cantidad reservada debe ser positiva
- `status IN (...)`: Solo estados válidos

**Índices:**
- `idx_store_reservations_store_id`: Índice en `store_id`
- `idx_store_reservations_item_id`: Índice en `item_id`
- `idx_store_reservations_status`: Índice en `status`
- `idx_store_reservations_store_item`: Índice compuesto en `(store_id, item_id)`

## 🔄 Flujo de Operaciones

### 1. Reserva de Stock por Tienda

1. Tienda llama a Command Service: `POST /api/v1/inventory/items/:id/reserve`
2. Command Service valida y publica evento `StockReserved` a Kafka
3. Listener Service consume el evento
4. Listener Service actualiza `inventory_items` (incrementa `reserved`, decrementa `available`)
5. Listener Service crea registro en `store_reservations` con status `active`

### 2. Liberación de Stock

1. Tienda llama a Command Service: `POST /api/v1/inventory/items/:id/release`
2. Command Service valida y publica evento `StockReleased` a Kafka
3. Listener Service consume el evento
4. Listener Service actualiza `inventory_items` (decrementa `reserved`, incrementa `available`)
5. Listener Service actualiza `store_reservations` con status `released` y `released_at`

### 3. Ajuste de Stock

1. Administrador llama a Command Service: `POST /api/v1/inventory/items/:id/adjust`
2. Command Service valida y publica evento `StockAdjusted` a Kafka
3. Listener Service consume el evento
4. Listener Service actualiza `inventory_items` (ajusta `quantity`, recalcula `available`)

## 🔒 Optimistic Locking

Todas las operaciones de escritura usan **optimistic locking** con el campo `version`:

```sql
UPDATE inventory_items
SET quantity = quantity + ?,
    available = (quantity + ?) - reserved,
    version = version + 1,
    updated_at = ?
WHERE id = ? AND version = ? AND (quantity + ?) >= 0
```

Si la versión no coincide, la operación falla y se reintenta automáticamente.

## 📊 Consultas Útiles

### Obtener inventario disponible

```sql
SELECT id, sku, name, quantity, reserved, available
FROM inventory_items
WHERE available > 0
ORDER BY available DESC;
```

### Obtener reservas activas de una tienda

```sql
SELECT sr.id, sr.quantity, sr.reserved_at, sr.expires_at,
       i.sku, i.name
FROM store_reservations sr
JOIN inventory_items i ON sr.item_id = i.id
WHERE sr.store_id = ? AND sr.status = 'active'
ORDER BY sr.reserved_at DESC;
```

### Obtener stock total por tienda

```sql
SELECT s.name, s.code,
       SUM(sr.quantity) as total_reserved
FROM stores s
LEFT JOIN store_reservations sr ON s.id = sr.store_id AND sr.status = 'active'
GROUP BY s.id, s.name, s.code;
```

### Obtener items con bajo stock

```sql
SELECT id, sku, name, quantity, reserved, available
FROM inventory_items
WHERE available < 10
ORDER BY available ASC;
```

## 🎯 Ventajas del Diseño

1. **Single Source of Truth**: Solo una base de datos centralizada
2. **Consistencia**: Optimistic locking previene conflictos
3. **Auditoría**: Tabla `store_reservations` permite tracking completo
4. **Escalabilidad**: SQLite es liviano y eficiente para este caso de uso
5. **Integridad**: Foreign keys y constraints garantizan integridad de datos
6. **Performance**: Índices optimizados para consultas frecuentes

## 🔧 Mantenimiento

### Backup

```bash
# Backup de la base de datos
sqlite3 inventory.db ".backup inventory_backup.db"
```

### Vacuum (optimización)

```bash
# Optimizar base de datos
sqlite3 inventory.db "VACUUM;"
```

### Análisis de índices

```bash
# Analizar y optimizar índices
sqlite3 inventory.db "ANALYZE;"
```

## 📝 Notas Importantes

1. **Single Writer Principle**: Solo el Listener Service escribe en esta base de datos
2. **WAL Mode**: La base de datos usa Write-Ahead Logging (WAL) para mejor concurrencia
3. **Foreign Keys**: Habilitadas con `_foreign_keys=1` en la conexión
4. **Timestamps**: Todos los timestamps están en formato ISO 8601 (RFC3339)
5. **Versioning**: El campo `version` se incrementa en cada actualización para optimistic locking

## 🚀 Próximos Pasos

1. **Migraciones**: Implementar sistema de migraciones para cambios de esquema
2. **Backup Automático**: Implementar backups automáticos periódicos
3. **Monitoreo**: Agregar métricas de uso de la base de datos
4. **Optimización**: Monitorear y optimizar queries lentas

