# Ejemplos de Request/Response - Command Service API

Este documento contiene ejemplos válidos e inválidos para todos los endpoints de la API.

## POST /api/v1/inventory/items - Crear Item

### Request Válido
```json
{
  "sku": "SKU-001",
  "name": "Laptop Dell XPS 15",
  "description": "High-performance laptop with 16GB RAM and 512GB SSD",
  "quantity": 100
}
```

### Response Válido (201 Created)
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "sku": "SKU-001",
  "name": "Laptop Dell XPS 15",
  "description": "High-performance laptop with 16GB RAM and 512GB SSD",
  "quantity": 100,
  "created_at": "2024-01-15T10:30:00Z"
}
```

### Request Inválido - Campos Requeridos Faltantes
```json
{
  "name": "Laptop Dell XPS 15",
  "quantity": 100
}
```

### Response Error (400 Bad Request)
```json
{
  "error": "Key: 'CreateItemRequest.SKU' Error:Field validation for 'SKU' failed on the 'required' tag"
}
```

### Request Inválido - Cantidad Negativa
```json
{
  "sku": "SKU-001",
  "name": "Laptop Dell XPS 15",
  "quantity": -10
}
```

### Response Error (400 Bad Request)
```json
{
  "error": "Key: 'CreateItemRequest.Quantity' Error:Field validation for 'Quantity' failed on the 'min' tag"
}
```

### Request Inválido - SKU Vacío
```json
{
  "sku": "",
  "name": "Laptop Dell XPS 15",
  "quantity": 100
}
```

### Response Error (400 Bad Request)
```json
{
  "error": "Key: 'CreateItemRequest.SKU' Error:Field validation for 'SKU' failed on the 'required' tag"
}
```

---

## PUT /api/v1/inventory/items/:id - Actualizar Item

### Request Válido
```json
{
  "name": "Laptop Dell XPS 15 - Updated",
  "description": "High-performance laptop with 32GB RAM and 1TB SSD"
}
```

### Response Válido (200 OK)
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "sku": "SKU-001",
  "name": "Laptop Dell XPS 15 - Updated",
  "description": "High-performance laptop with 32GB RAM and 1TB SSD",
  "quantity": 100,
  "updated_at": "2024-01-15T11:45:00Z"
}
```

### Request Inválido - Nombre Faltante
```json
{
  "description": "Updated description"
}
```

### Response Error (400 Bad Request)
```json
{
  "error": "Key: 'UpdateItemRequest.Name' Error:Field validation for 'Name' failed on the 'required' tag"
}
```

### Request Inválido - ID Inválido (UUID malformado)
```
PUT /api/v1/inventory/items/invalid-id
```

### Response Error (400 Bad Request)
```json
{
  "error": "invalid item id"
}
```

### Request Inválido - Item No Encontrado
```
PUT /api/v1/inventory/items/550e8400-e29b-41d4-a716-446655440000
```

### Response Error (404 Not Found)
```json
{
  "error": "item not found"
}
```

---

## DELETE /api/v1/inventory/items/:id - Eliminar Item

### Request Válido
```
DELETE /api/v1/inventory/items/550e8400-e29b-41d4-a716-446655440000
```

### Response Válido (200 OK)
```json
{
  "message": "item deleted successfully"
}
```

### Request Inválido - ID Inválido
```
DELETE /api/v1/inventory/items/invalid-id
```

### Response Error (400 Bad Request)
```json
{
  "error": "invalid item id"
}
```

### Request Inválido - Item No Encontrado
```
DELETE /api/v1/inventory/items/550e8400-e29b-41d4-a716-446655440000
```

### Response Error (404 Not Found)
```json
{
  "error": "item not found"
}
```

---

## POST /api/v1/inventory/items/:id/adjust - Ajustar Stock

### Request Válido - Aumentar Stock
```json
{
  "quantity": 10
}
```

### Response Válido (200 OK)
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "quantity": 110,
  "available": 90,
  "reserved": 20,
  "updated_at": "2024-01-15T12:00:00Z"
}
```

### Request Válido - Disminuir Stock
```json
{
  "quantity": -5
}
```

### Response Válido (200 OK)
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "quantity": 95,
  "available": 75,
  "reserved": 20,
  "updated_at": "2024-01-15T12:00:00Z"
}
```

### Request Inválido - Stock Insuficiente (resultaría en negativo)
```json
{
  "quantity": -200
}
```

### Response Error (400 Bad Request)
```json
{
  "error": "insufficient stock available"
}
```

### Request Inválido - Cantidad Faltante
```json
{}
```

### Response Error (400 Bad Request)
```json
{
  "error": "Key: 'AdjustStockRequest.Quantity' Error:Field validation for 'Quantity' failed on the 'required' tag"
}
```

---

## POST /api/v1/inventory/items/:id/reserve - Reservar Stock

### Request Válido
```json
{
  "quantity": 5
}
```

### Response Válido (200 OK)
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "quantity": 100,
  "available": 75,
  "reserved": 25,
  "updated_at": "2024-01-15T12:00:00Z"
}
```

### Request Inválido - Stock Insuficiente
```json
{
  "quantity": 200
}
```

### Response Error (400 Bad Request)
```json
{
  "error": "insufficient stock available"
}
```

### Request Inválido - Cantidad Menor a 1
```json
{
  "quantity": 0
}
```

### Response Error (400 Bad Request)
```json
{
  "error": "Key: 'ReserveStockRequest.Quantity' Error:Field validation for 'Quantity' failed on the 'min' tag"
}
```

### Request Inválido - Cantidad Faltante
```json
{}
```

### Response Error (400 Bad Request)
```json
{
  "error": "Key: 'ReserveStockRequest.Quantity' Error:Field validation for 'Quantity' failed on the 'required' tag"
}
```

---

## POST /api/v1/inventory/items/:id/release - Liberar Stock Reservado

### Request Válido
```json
{
  "quantity": 5
}
```

### Response Válido (200 OK)
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "quantity": 100,
  "available": 85,
  "reserved": 15,
  "updated_at": "2024-01-15T12:00:00Z"
}
```

### Request Inválido - Cantidad Excede lo Reservado
```json
{
  "quantity": 200
}
```

### Response Error (400 Bad Request)
```json
{
  "error": "invalid release quantity"
}
```

### Request Inválido - Cantidad Menor a 1
```json
{
  "quantity": 0
}
```

### Response Error (400 Bad Request)
```json
{
  "error": "Key: 'ReleaseStockRequest.Quantity' Error:Field validation for 'Quantity' failed on the 'min' tag"
}
```

### Request Inválido - Cantidad Faltante
```json
{}
```

### Response Error (400 Bad Request)
```json
{
  "error": "Key: 'ReleaseStockRequest.Quantity' Error:Field validation for 'Quantity' failed on the 'required' tag"
}
```

---

## GET /health - Health Check

### Request Válido
```
GET /health
```

### Response Válido (200 OK)
```json
{
  "status": "ok",
  "service": "command-service"
}
```

---

## Notas Importantes

1. **UUIDs**: Todos los IDs de items deben ser UUIDs válidos en formato `550e8400-e29b-41d4-a716-446655440000`
2. **Cantidades**: Las cantidades deben ser números enteros >= 0 para creación, >= 1 para reserva/liberación
3. **Campos Requeridos**: `sku`, `name`, y `quantity` son requeridos para crear items
4. **Stock Disponible**: El stock disponible se calcula como `quantity - reserved`
5. **Validaciones**: Todos los requests son validados antes de procesarse

