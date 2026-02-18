# Resultados de Pruebas - Listener Service

Este documento consolida todos los resultados de pruebas del Listener Service.

## 📊 Resumen Ejecutivo

**Última actualización:** 2025-11-09  
**Estado general:** ⚠️ Pruebas pendientes de implementación

### Estado Actual

- **Pruebas Unitarias:** Pendientes de implementación
- **Pruebas de Integración:** Pendientes de implementación
- **Pruebas E2E:** Verificadas manualmente en flujo completo

---

## 🏗️ Arquitectura del Servicio

El Listener Service implementa:
- **Single Writer Principle**: Solo este servicio escribe en SQLite
- **Optimistic Locking**: Version/timestamp para concurrencia
- **Event Processing**: Procesa todos los eventos de inventario
- **Retry Logic**: Reintentos automáticos con backoff exponencial
- **Dead Letter Queue**: Manejo de eventos fallidos (placeholder)

---

## 🔄 Pruebas de Integración Manual

### Flujo Verificado

El servicio ha sido verificado manualmente en el flujo completo:

1. **Command Service** publica eventos a Kafka
2. **Listener Service** consume eventos de Kafka
3. **Listener Service** procesa eventos y actualiza SQLite
4. **Query Service** lee desde SQLite (Read Model)

### Eventos Procesados

El servicio procesa los siguientes eventos:

#### Items Events
- ✅ **InventoryItemCreated**: Crea un nuevo item
- ✅ **InventoryItemUpdated**: Actualiza un item existente
- ✅ **InventoryItemDeleted**: Elimina un item

#### Stock Events
- ✅ **StockAdjusted**: Ajusta la cantidad de stock
- ✅ **StockReserved**: Reserva stock
- ✅ **StockReleased**: Libera stock reservado

### Verificación Manual

**Endpoint de monitoreo:**
```bash
GET http://localhost:8082/api/v1/monitoring/stats
```

**Verificación de base de datos:**
```bash
sqlite3 listener-service/inventory.db "SELECT * FROM inventory_items;"
```

---

## 🐛 Correcciones Implementadas

### Bug en `processStockAdjusted`

**Problema:** El método `processStockAdjusted` estaba calculando incorrectamente el ajuste de stock.

**Código Anterior (Incorrecto):**
```go
adjustment := event.Quantity - currentItem.Quantity
```

**Problema:** El evento `StockAdjustedEvent` que se publica desde Command Service tiene:
- `Quantity`: El ajuste (diferencia) que se quiere aplicar (ej: +25 o -30)
- `NewTotal`: La nueva cantidad total después del ajuste

El código anterior estaba interpretando `event.Quantity` como la cantidad total nueva, cuando en realidad es el ajuste.

**Código Corregido:**
```go
// The event.Quantity field contains the adjustment (difference), not the new total
// For example: if stock was 100 and we adjust by +25, event.Quantity = 25
adjustment := event.Quantity
```

**Resultado:** Los valores de stock ahora se actualizan correctamente en todo el flujo.

---

## 🎯 Próximos Pasos de Implementación

### Pruebas Unitarias Pendientes

1. **Event Processor (`internal/events/event_processor.go`)**
   - Procesamiento de eventos de items
   - Procesamiento de eventos de stock
   - Manejo de errores
   - Retry logic

2. **Database (`internal/database/sqlite.go`)**
   - Operaciones CRUD
   - Optimistic locking
   - Manejo de conflictos
   - Transacciones

3. **Kafka Consumer (`internal/kafka/consumer.go`)**
   - Consumo de mensajes
   - Manejo de offsets
   - Manejo de errores
   - Dead Letter Queue

### Pruebas de Integración Pendientes

1. **Integración con Kafka**
   - Consumo de eventos reales
   - Procesamiento de eventos
   - Manejo de errores de Kafka

2. **Integración con SQLite**
   - Escritura de datos
   - Optimistic locking en concurrencia
   - Manejo de conflictos

3. **Flujo End-to-End**
   - Command Service → Kafka → Listener Service → SQLite
   - Verificación de consistencia
   - Verificación de eventual consistency

---

## 📊 Métricas de Procesamiento

### Endpoint de Monitoreo

El servicio expone un endpoint de monitoreo:

```bash
GET http://localhost:8082/api/v1/monitoring/stats
```

**Respuesta esperada:**
```json
{
  "events_processed": 100,
  "events_failed": 0,
  "events_in_dlq": 0,
  "last_processed_at": "2025-11-09T10:00:00Z"
}
```

---

## 🔒 Single Writer Principle

El servicio implementa el **Single Writer Principle** para garantizar que solo un proceso escriba en la base de datos SQLite:

- **Mutex**: Usa un mutex para serializar todas las escrituras
- **Connection Pool**: Configurado con `MaxOpenConns=1` para un solo escritor
- **Atomic Operations**: Todas las operaciones de escritura son atómicas

---

## 🔄 Optimistic Locking

El servicio usa **Optimistic Locking** con version/timestamp:

- **Version Field**: Cada registro tiene un campo `version` que se incrementa en cada actualización
- **Version Check**: Antes de actualizar, se verifica que la versión coincida
- **Conflict Detection**: Si la versión no coincide, se retorna `ErrOptimisticLockFailed`
- **Retry Logic**: Los conflictos se manejan con reintentos automáticos

### Ejemplo de Optimistic Locking

```sql
UPDATE inventory_items
SET quantity = quantity + ?, version = version + 1, updated_at = ?
WHERE id = ? AND version = ? AND (quantity + ?) >= 0
```

Si la versión no coincide, la actualización falla y se reintenta.

---

## 🔄 Retry Logic

El servicio implementa retry logic con backoff exponencial:

- **Max Retries**: Configurable (default: 3)
- **Retry Delay**: Delay incremental entre reintentos
- **Optimistic Lock Failures**: Se reintentan automáticamente
- **Other Errors**: Se reintentan según configuración

---

## 📨 Dead Letter Queue

El servicio puede enviar eventos fallidos a un Dead Letter Queue:

- **DLQ Enabled**: Configurable (default: true)
- **DLQ Topic**: Configurable (default: `inventory.dlq`)
- **Failed Events**: Eventos que fallan después de todos los reintentos

**Nota:** Actualmente es un placeholder, pendiente de implementación real.

---

## 🎯 Flujo de Procesamiento

1. **Consume Event**: El consumer recibe un evento de Kafka
2. **Extract Event Type**: Extrae el tipo de evento de los headers
3. **Process Event**: Procesa el evento con retry logic
4. **Update Database**: Actualiza SQLite con optimistic locking
5. **Handle Failures**: Envía a DLQ si falla después de reintentos
6. **Commit Offset**: Marca el mensaje como procesado

---

## 📝 Notas Importantes

1. **Single Writer**: Solo este servicio debe escribir en SQLite para garantizar consistencia.

2. **Optimistic Locking**: Los conflictos de versión son normales en alta concurrencia y se manejan automáticamente con retry logic.

3. **Eventual Consistency**: En arquitectura CQRS + EDA, los cambios pueden tardar unos segundos en estar disponibles en el Query Service.

4. **Base de Datos**: La base de datos SQLite es la única fuente de verdad para el inventario.

5. **Pruebas**: Las pruebas unitarias y de integración están pendientes de implementación.

---

## 📚 Referencias

- **Esquema de Base de Datos**: Ver `SCHEMA.md` para detalles del esquema SQLite
- **Command Service**: Ver `../command-service/README.md`
- **Query Service**: Ver `../query-service/README.md`
- **Arquitectura CQRS**: Ver documentación de arquitectura distribuida

