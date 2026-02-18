# Ejemplos de Request/Response - Query Service API

Este documento contiene ejemplos válidos e inválidos para todos los endpoints de la API de lectura.

## GET /api/v1/inventory/items - Listar Items

### Request Válido - Paginación por Defecto
```
GET /api/v1/inventory/items
```

### Response Válido (200 OK)
```json
{
  "items": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "sku": "SKU-001",
      "name": "Laptop Dell XPS 15",
      "description": "High-performance laptop with 16GB RAM and 512GB SSD",
      "quantity": 100,
      "reserved": 20,
      "available": 80,
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T11:45:00Z"
    }
  ],
  "total": 100,
  "page": 1,
  "page_size": 10,
  "total_pages": 10
}
```

### Request Válido - Paginación Personalizada
```
GET /api/v1/inventory/items?page=2&page_size=20
```

### Response Válido (200 OK)
```json
{
  "items": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "sku": "SKU-001",
      "name": "Laptop Dell XPS 15",
      "description": "High-performance laptop with 16GB RAM and 512GB SSD",
      "quantity": 100,
      "reserved": 20,
      "available": 80,
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T11:45:00Z"
    }
  ],
  "total": 100,
  "page": 2,
  "page_size": 20,
  "total_pages": 5
}
```

### Request Inválido - Página Negativa
```
GET /api/v1/inventory/items?page=-1
```

### Response (200 OK - Se corrige automáticamente)
```json
{
  "items": [],
  "total": 100,
  "page": 1,
  "page_size": 10,
  "total_pages": 10
}
```

### Request Inválido - Page Size Mayor a 100
```
GET /api/v1/inventory/items?page_size=200
```

### Response (200 OK - Se limita a 100)
```json
{
  "items": [],
  "total": 100,
  "page": 1,
  "page_size": 100,
  "total_pages": 1
}
```

### Request Inválido - Página Fuera de Rango
```
GET /api/v1/inventory/items?page=999
```

### Response Válido (200 OK - Lista vacía)
```json
{
  "items": [],
  "total": 100,
  "page": 999,
  "page_size": 10,
  "total_pages": 10
}
```

---

## GET /api/v1/inventory/items/:id - Obtener Item por ID

### Request Válido
```
GET /api/v1/inventory/items/550e8400-e29b-41d4-a716-446655440000
```

### Response Válido (200 OK)
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "sku": "SKU-001",
  "name": "Laptop Dell XPS 15",
  "description": "High-performance laptop with 16GB RAM and 512GB SSD",
  "quantity": 100,
  "reserved": 20,
  "available": 80,
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T11:45:00Z"
}
```

### Request Inválido - ID Inválido (UUID malformado)
```
GET /api/v1/inventory/items/invalid-id
```

### Response Error (400 Bad Request)
```json
{
  "error": "invalid item id"
}
```

### Request Inválido - Item No Encontrado
```
GET /api/v1/inventory/items/550e8400-e29b-41d4-a716-446655440000
```

### Response Error (404 Not Found)
```json
{
  "error": "item not found"
}
```

---

## GET /api/v1/inventory/items/sku/:sku - Obtener Item por SKU

### Request Válido
```
GET /api/v1/inventory/items/sku/SKU-001
```

### Response Válido (200 OK)
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "sku": "SKU-001",
  "name": "Laptop Dell XPS 15",
  "description": "High-performance laptop with 16GB RAM and 512GB SSD",
  "quantity": 100,
  "reserved": 20,
  "available": 80,
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T11:45:00Z"
}
```

### Request Válido - SKU con Formato Diferente
```
GET /api/v1/inventory/items/sku/PROD-2024-ABC
```

### Response Válido (200 OK)
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440001",
  "sku": "PROD-2024-ABC",
  "name": "Product ABC",
  "description": "Product description",
  "quantity": 50,
  "reserved": 5,
  "available": 45,
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T11:45:00Z"
}
```

### Request Inválido - SKU Vacío
```
GET /api/v1/inventory/items/sku/
```

### Response Error (400 Bad Request)
```json
{
  "error": "sku is required"
}
```

### Request Inválido - SKU No Encontrado
```
GET /api/v1/inventory/items/sku/NONEXISTENT-SKU
```

### Response Error (404 Not Found)
```json
{
  "error": "item not found"
}
```

---

## GET /api/v1/inventory/items/:id/stock - Obtener Estado de Stock

### Request Válido
```
GET /api/v1/inventory/items/550e8400-e29b-41d4-a716-446655440000/stock
```

### Response Válido (200 OK)
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "sku": "SKU-001",
  "quantity": 100,
  "reserved": 20,
  "available": 80,
  "updated_at": "2024-01-15T12:00:00Z"
}
```

### Request Inválido - ID Inválido (UUID malformado)
```
GET /api/v1/inventory/items/invalid-id/stock
```

### Response Error (400 Bad Request)
```json
{
  "error": "invalid item id"
}
```

### Request Inválido - Item No Encontrado
```
GET /api/v1/inventory/items/550e8400-e29b-41d4-a716-446655440000/stock
```

### Response Error (404 Not Found)
```json
{
  "error": "item not found"
}
```

---

## GET /api/v1/health - Health Check

### Request Válido
```
GET /api/v1/health
```

### Response Válido (200 OK)
```json
{
  "status": "ok",
  "service": "query-service"
}
```

---

## Notas Importantes

1. **Cache**: Todas las respuestas exitosas se cachean para mejorar el rendimiento
2. **Paginación**: 
   - `page` debe ser >= 1 (se corrige automáticamente si es menor)
   - `page_size` debe estar entre 1 y 100 (se corrige automáticamente)
   - Si la página está fuera de rango, se retorna una lista vacía
3. **UUIDs**: Todos los IDs de items deben ser UUIDs válidos en formato `550e8400-e29b-41d4-a716-446655440000`
4. **SKU**: El SKU puede tener cualquier formato de string válido
5. **Cache Hit**: Las respuestas desde cache son más rápidas y no requieren acceso a la base de datos
6. **Cache Miss**: Si el cache no tiene los datos, se consulta el Read Model y se cachea el resultado

## Optimizaciones de Cache

- **Cache Keys**:
  - `item:id:{id}` - Item por ID
  - `item:sku:{sku}` - Item por SKU
  - `stock:{id}` - Estado de stock (TTL más corto)
  - `items:list:{page}:{pageSize}` - Lista paginada

- **TTL (Time To Live)**:
  - Items individuales: 5 minutos (configurable)
  - Estado de stock: 2.5 minutos (la mitad del TTL normal)
  - Listas paginadas: 5 minutos (configurable)

