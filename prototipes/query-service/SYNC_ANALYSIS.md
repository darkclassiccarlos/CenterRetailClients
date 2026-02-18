# Análisis de Sincronización del Query Service

## 🔍 Problema Identificado

### Estado Actual del Query Service

El Query Service tiene un **repositorio in-memory** (`InMemoryReadRepository`) que **NO se sincroniza automáticamente** con el Command Service.

### ⏱️ Tiempo de Sincronización

**Respuesta directa: NUNCA se sincroniza automáticamente** porque no hay código que actualice el repositorio in-memory.

## 🔍 Análisis del Código

### 1. Repositorio In-Memory (Vacío al Inicio)

```go
// query-service/internal/repository/read_repository.go
type InMemoryReadRepository struct {
    items map[uuid.UUID]*models.InventoryItem  // Map vacío al inicio
}

func NewReadRepository() ReadRepository {
    return &InMemoryReadRepository{
        items: make(map[uuid.UUID]*models.InventoryItem), // Vacío
    }
}
```

**Problema**: El map `items` se inicializa vacío y **nunca se actualiza**.

### 2. Kafka Consumer (Solo Invalida Cache)

```go
// query-service/internal/kafka/consumer.go:218-237
func (h *cacheInvalidationHandler) invalidateCache(ctx context.Context, eventType string, eventData []byte) error {
    switch eventType {
    case "InventoryItemCreated", "InventoryItemUpdated", "InventoryItemDeleted",
         "StockAdjusted", "StockReserved", "StockReleased":
        // Invalidate all inventory-related cache
        h.logger.Info("Invalidating inventory cache", ...)
        // TODO: Implement specific cache key invalidation based on event data
        return nil
    }
}
```

**Problema**: El consumer solo invalida el cache, **NO actualiza el repositorio in-memory**.

### 3. No Hay Código que Actualice el Repositorio

**Búsqueda realizada**: No existe código que haga:
- `r.items[id] = item` (crear/actualizar item)
- `delete(r.items, id)` (eliminar item)
- Cualquier operación que modifique `r.items`

## 📊 Flujo Actual vs. Esperado

### Flujo Actual (Implementación Placeholder)

```
Command Service (Puerto 8080)
    ↓ Publica evento a Kafka
Kafka Topic (inventory.items, inventory.stock)
    ↓ Query Service consume evento
Query Service Kafka Consumer
    ↓ Solo invalida cache
Cache (invalida claves)
    ↓ Repositorio in-memory
InMemoryReadRepository
    ❌ NUNCA se actualiza (permanece vacío)
```

**Resultado**: El repositorio in-memory **siempre está vacío**, por lo que:
- `GET /api/v1/inventory/items/:id` → 404 (item not found)
- `GET /api/v1/inventory/items/sku/:sku` → 404 (item not found)
- `GET /api/v1/inventory/items/:id/stock` → 404 (item not found)
- `GET /api/v1/inventory/items` → Lista vacía

### Flujo Esperado (Arquitectura CQRS + EDA Real)

```
Command Service (Puerto 8080)
    ↓ Publica evento a Kafka
Kafka Topic (inventory.items, inventory.stock)
    ↓ Listener Service consume evento
Listener Service
    ↓ Procesa evento y actualiza base de datos
SQLite Database (Read Model)
    ↓ Query Service lee desde base de datos
Query Service Repository
    ✅ Lee desde base de datos actualizada
```

**Resultado**: El Query Service lee desde una base de datos que el Listener Service actualiza.

## ⏱️ Tiempos de Sincronización

### En Implementación Actual (Placeholder)

| Paso | Tiempo | Estado |
|------|--------|--------|
| Command Service → Kafka | ~1-10ms | ✅ Funciona |
| Kafka → Query Service Consumer | ~1-50ms | ✅ Funciona |
| Query Service Consumer → Cache Invalidation | ~1-5ms | ✅ Funciona |
| Query Service Consumer → Repository Update | **NUNCA** | ❌ No implementado |

**Total**: **NUNCA se sincroniza** (tiempo infinito)

### En Arquitectura CQRS + EDA Real (Esperado)

| Paso | Tiempo | Estado |
|------|--------|--------|
| Command Service → Kafka | ~1-10ms | ✅ |
| Kafka → Listener Service | ~1-50ms | ✅ |
| Listener Service → SQLite | ~5-20ms | ✅ |
| SQLite → Query Service | ~1-5ms | ✅ |

**Total esperado**: **~8-85ms** (muy rápido)

## 🎯 Conclusión

### ¿Cuánto tiempo toma la sincronización?

**Respuesta**: **NUNCA** (tiempo infinito)

**Razón**: El repositorio in-memory del Query Service **nunca se actualiza** porque:
1. Es un **placeholder** para desarrollo
2. No hay código que actualice `r.items` desde eventos
3. El Kafka consumer solo invalida cache, no actualiza el repositorio

### ¿Por qué los tests fallan?

Los tests 6, 9 y 11 fallan porque:
- El repositorio in-memory está vacío
- No hay código que lo actualice desde eventos
- Los items creados en Command Service nunca llegan al Query Service

### Solución Actual vs. Esperada

**Solución Actual (Placeholder)**:
- ✅ Invalida cache cuando hay eventos
- ❌ NO actualiza el repositorio in-memory
- ❌ NO lee desde una base de datos real

**Solución Esperada (Producción)**:
- El Query Service debería leer desde la misma base de datos SQLite que el Listener Service actualiza
- O usar una base de datos separada (Read Model) que el Listener Service actualiza

## 🔧 Próximos Pasos

Para que funcione correctamente, necesitarías:

1. **Opción 1 (Recomendada)**: Hacer que el Query Service lea desde la misma base de datos SQLite que el Listener Service actualiza
2. **Opción 2**: Implementar un listener en el Query Service que actualice el repositorio in-memory desde eventos
3. **Opción 3**: Usar una base de datos separada (Read Model) que el Listener Service actualiza
