# Eventos Publicados - Command Service

Este documento describe los eventos de dominio publicados por el Command Service en la arquitectura CQRS + EDA.

## Event Broker

El servicio publica eventos a través de un Event Broker (Kafka, RabbitMQ, etc.) configurado mediante la variable de entorno `EVENT_BROKER`.

**Configuración:**
- Variable de entorno: `EVENT_BROKER` (default: `localhost:9092`)
- Formato: `host:port`

## Formato de Eventos

Todos los eventos siguen el siguiente formato estándar:

```json
{
  "eventType": "string",
  "eventId": "uuid",
  "aggregateId": "uuid",
  "occurredAt": "ISO8601 timestamp",
  "version": "integer",
  "data": {
    // Datos específicos del evento
  }
}
```

### Atributos Obligatorios

- `eventType`: Tipo del evento (string, requerido)
- `eventId`: Identificador único del evento (UUID, requerido)
- `aggregateId`: ID del agregado afectado (UUID, requerido)
- `occurredAt`: Timestamp ISO 8601 del momento en que ocurrió el evento (string, requerido)
- `version`: Versión del esquema del evento (integer, requerido)
- `data`: Objeto con los datos específicos del evento (object, requerido)

## Eventos de Inventario

### 1. InventoryItemCreatedEvent

**Topic:** `inventory.items.created`

**Descripción:** Evento publicado cuando se crea un nuevo item de inventario.

**Formato:**
```json
{
  "eventType": "InventoryItemCreated",
  "eventId": "550e8400-e29b-41d4-a716-446655440000",
  "aggregateId": "550e8400-e29b-41d4-a716-446655440000",
  "occurredAt": "2024-01-15T10:30:00Z",
  "version": 1,
  "data": {
    "itemId": "550e8400-e29b-41d4-a716-446655440000",
    "sku": "SKU-001",
    "name": "Laptop Dell XPS 15",
    "description": "High-performance laptop with 16GB RAM and 512GB SSD",
    "quantity": 100
  }
}
```

**Atributos Obligatorios en `data`:**
- `itemId` (UUID): ID del item creado
- `sku` (string): SKU del producto
- `name` (string): Nombre del producto
- `quantity` (integer): Cantidad inicial de stock

**Atributos Opcionales en `data`:**
- `description` (string): Descripción del producto

---

### 2. InventoryItemUpdatedEvent

**Topic:** `inventory.items.updated`

**Descripción:** Evento publicado cuando se actualiza un item de inventario.

**Formato:**
```json
{
  "eventType": "InventoryItemUpdated",
  "eventId": "550e8400-e29b-41d4-a716-446655440001",
  "aggregateId": "550e8400-e29b-41d4-a716-446655440000",
  "occurredAt": "2024-01-15T11:45:00Z",
  "version": 1,
  "data": {
    "itemId": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Laptop Dell XPS 15 - Updated",
    "description": "High-performance laptop with 32GB RAM and 1TB SSD"
  }
}
```

**Atributos Obligatorios en `data`:**
- `itemId` (UUID): ID del item actualizado
- `name` (string): Nuevo nombre del producto

**Atributos Opcionales en `data`:**
- `description` (string): Nueva descripción del producto

---

### 3. InventoryItemDeletedEvent

**Topic:** `inventory.items.deleted`

**Descripción:** Evento publicado cuando se elimina un item de inventario.

**Formato:**
```json
{
  "eventType": "InventoryItemDeleted",
  "eventId": "550e8400-e29b-41d4-a716-446655440002",
  "aggregateId": "550e8400-e29b-41d4-a716-446655440000",
  "occurredAt": "2024-01-15T12:00:00Z",
  "version": 1,
  "data": {
    "itemId": "550e8400-e29b-41d4-a716-446655440000",
    "sku": "SKU-001"
  }
}
```

**Atributos Obligatorios en `data`:**
- `itemId` (UUID): ID del item eliminado
- `sku` (string): SKU del producto eliminado

---

### 4. StockAdjustedEvent

**Topic:** `inventory.stock.adjusted`

**Descripción:** Evento publicado cuando se ajusta el stock de un item (aumento o disminución).

**Formato:**
```json
{
  "eventType": "StockAdjusted",
  "eventId": "550e8400-e29b-41d4-a716-446655440003",
  "aggregateId": "550e8400-e29b-41d4-a716-446655440000",
  "occurredAt": "2024-01-15T12:15:00Z",
  "version": 1,
  "data": {
    "itemId": "550e8400-e29b-41d4-a716-446655440000",
    "sku": "SKU-001",
    "adjustment": 10,
    "previousQuantity": 100,
    "newQuantity": 110
  }
}
```

**Atributos Obligatorios en `data`:**
- `itemId` (UUID): ID del item
- `sku` (string): SKU del producto
- `adjustment` (integer): Cantidad ajustada (positivo para aumento, negativo para disminución)
- `previousQuantity` (integer): Cantidad anterior
- `newQuantity` (integer): Nueva cantidad total

---

### 5. StockReservedEvent

**Topic:** `inventory.stock.reserved`

**Descripción:** Evento publicado cuando se reserva stock de un item.

**Formato:**
```json
{
  "eventType": "StockReserved",
  "eventId": "550e8400-e29b-41d4-a716-446655440004",
  "aggregateId": "550e8400-e29b-41d4-a716-446655440000",
  "occurredAt": "2024-01-15T12:30:00Z",
  "version": 1,
  "data": {
    "itemId": "550e8400-e29b-41d4-a716-446655440000",
    "sku": "SKU-001",
    "reservedQuantity": 5,
    "totalQuantity": 100,
    "reservedTotal": 25,
    "availableQuantity": 75
  }
}
```

**Atributos Obligatorios en `data`:**
- `itemId` (UUID): ID del item
- `sku` (string): SKU del producto
- `reservedQuantity` (integer): Cantidad reservada en esta operación
- `totalQuantity` (integer): Cantidad total de stock
- `reservedTotal` (integer): Total de stock reservado
- `availableQuantity` (integer): Cantidad disponible (total - reservado)

---

### 6. StockReleasedEvent

**Topic:** `inventory.stock.released`

**Descripción:** Evento publicado cuando se libera stock previamente reservado.

**Formato:**
```json
{
  "eventType": "StockReleased",
  "eventId": "550e8400-e29b-41d4-a716-446655440005",
  "aggregateId": "550e8400-e29b-41d4-a716-446655440000",
  "occurredAt": "2024-01-15T12:45:00Z",
  "version": 1,
  "data": {
    "itemId": "550e8400-e29b-41d4-a716-446655440000",
    "sku": "SKU-001",
    "releasedQuantity": 5,
    "totalQuantity": 100,
    "reservedTotal": 20,
    "availableQuantity": 80
  }
}
```

**Atributos Obligatorios en `data`:**
- `itemId` (UUID): ID del item
- `sku` (string): SKU del producto
- `releasedQuantity` (integer): Cantidad liberada en esta operación
- `totalQuantity` (integer): Cantidad total de stock
- `reservedTotal` (integer): Total de stock reservado después de la liberación
- `availableQuantity` (integer): Cantidad disponible (total - reservado)

---

## Consumo de Eventos

Los eventos publicados pueden ser consumidos por:

1. **Query Service**: Para actualizar el modelo de lectura (Read Model)
2. **Listener Service**: Para procesar eventos y actualizar otros sistemas
3. **Otros servicios**: Para mantener consistencia eventual entre servicios

## Notas Importantes

- Todos los eventos son **idempotentes** y deben incluir un `eventId` único
- Los eventos se publican **después** de persistir los cambios en la base de datos
- En caso de error al publicar eventos, el sistema registra el error pero no revierte la operación (patrón "at-least-once delivery")
- Los eventos deben ser consumidos en orden para mantener la consistencia eventual

