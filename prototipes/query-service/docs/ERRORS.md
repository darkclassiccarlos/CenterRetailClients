# Errores Comunes - Query Service API

Este documento describe los errores comunes que pueden ocurrir al usar la API del Query Service.

## Códigos de Respuesta HTTP

### 200 OK
Operación exitosa. Retorna los datos solicitados. Los datos pueden venir del cache o del Read Model.

### 400 Bad Request
Request inválido. El cliente debe corregir el request antes de reintentar.

### 404 Not Found
Recurso no encontrado. El ID o SKU proporcionado no existe en el sistema.

### 500 Internal Server Error
Error interno del servidor. El servidor encontró un error inesperado al leer datos.

### 503 Service Unavailable
Servicio no disponible. Generalmente por problemas de conexión con dependencias (cache, base de datos).

---

## Errores de Validación (400 Bad Request)

### ID Inválido (UUID malformado)

**Error:** `invalid item id`

**Causa:** Se proporcionó un ID que no es un UUID válido.

**Solución:** Usar un UUID válido en el formato: `550e8400-e29b-41d4-a716-446655440000`

**Ejemplo de Request Inválido:**
```
GET /api/v1/inventory/items/invalid-id
```

**Ejemplo de Response:**
```json
{
  "error": "invalid item id"
}
```

---

### SKU Vacío

**Error:** `sku is required`

**Causa:** Se intentó obtener un item por SKU pero el SKU está vacío.

**Solución:** Proporcionar un SKU válido en la URL.

**Ejemplo de Request Inválido:**
```
GET /api/v1/inventory/items/sku/
```

**Ejemplo de Response:**
```json
{
  "error": "sku is required"
}
```

---

### Parámetros de Paginación Inválidos

**Error:** `invalid pagination parameters`

**Causa:** Se proporcionaron parámetros de paginación inválidos (aunque el sistema los corrige automáticamente).

**Solución:** 
- `page` debe ser >= 1 (se corrige automáticamente)
- `page_size` debe estar entre 1 y 100 (se corrige automáticamente)

**Ejemplo de Request Inválido:**
```
GET /api/v1/inventory/items?page=-1&page_size=200
```

**Nota:** El sistema corrige automáticamente estos valores, por lo que no se retorna error, pero se usan valores por defecto.

---

## Errores de Recurso No Encontrado (404 Not Found)

### Item No Encontrado por ID

**Error:** `item not found`

**Causa:** Se intentó acceder a un item que no existe en el sistema.

**Solución:** Verificar que el ID del item sea correcto y que el item exista.

**Ejemplo de Request:**
```
GET /api/v1/inventory/items/550e8400-e29b-41d4-a716-446655440000
```

**Ejemplo de Response:**
```json
{
  "error": "item not found"
}
```

---

### Item No Encontrado por SKU

**Error:** `item not found`

**Causa:** Se intentó acceder a un item por SKU que no existe en el sistema.

**Solución:** Verificar que el SKU sea correcto y que el item exista.

**Ejemplo de Request:**
```
GET /api/v1/inventory/items/sku/NONEXISTENT-SKU
```

**Ejemplo de Response:**
```json
{
  "error": "item not found"
}
```

---

## Errores de Conexión al Cache (503 Service Unavailable)

### Cache No Disponible

**Error:** `cache service unavailable`

**Causa:** No se puede establecer conexión con el cache (Redis).

**Solución:** 
1. Verificar que Redis esté corriendo
2. Verificar la configuración de Redis (`REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`, `REDIS_DB`)
3. Verificar conectividad de red

**Ejemplo de Response:**
```json
{
  "error": "cache service unavailable"
}
```

**Nota:** En algunos casos, el servicio puede funcionar sin cache consultando directamente el Read Model, pero con mayor latencia.

---

## Errores de Base de Datos (500 Internal Server Error)

### Error de Lectura

**Error:** `failed to list items` / `failed to get item` / `failed to get stock status`

**Causa:** Error al leer datos del Read Model (base de datos de lectura).

**Solución:**
1. Verificar que la base de datos esté disponible
2. Verificar la configuración de conexión
3. Revisar los logs del servidor para más detalles

**Ejemplo de Response:**
```json
{
  "error": "failed to list items"
}
```

---

### Error de Conexión a Base de Datos

**Error:** `database connection failed`

**Causa:** No se puede establecer conexión con la base de datos de lectura.

**Solución:**
1. Verificar que la base de datos esté corriendo
2. Verificar la configuración de conexión
3. Verificar conectividad de red

---

## Manejo de Errores

### Estructura de Error Response

Todos los errores siguen el siguiente formato:

```json
{
  "error": "mensaje de error descriptivo"
}
```

### Logs del Servidor

Para obtener más detalles sobre los errores, revisar los logs del servidor que incluyen:
- Timestamp del error
- Stack trace (en modo development)
- Contexto adicional (IDs, valores, etc.)
- Cache hit/miss information

### Retry Strategy

Para errores transitorios (503, 500), se recomienda:
1. Implementar retry con backoff exponencial
2. Verificar el estado del servicio antes de reintentar
3. No reintentar para errores de validación (400) sin corregir el request

### Cache Behavior

- **Cache Hit**: Respuesta rápida desde cache, sin acceso a base de datos
- **Cache Miss**: Consulta al Read Model, resultado se cachea para próximas consultas
- **Cache Error**: Si el cache falla, se consulta directamente el Read Model (mayor latencia)

---

## Códigos de Estado por Endpoint

### GET /api/v1/inventory/items
- **200 OK**: Lista de items obtenida exitosamente
- **400 Bad Request**: Parámetros de paginación inválidos (se corrigen automáticamente)
- **500 Internal Server Error**: Error de lectura del Read Model
- **503 Service Unavailable**: Cache no disponible

### GET /api/v1/inventory/items/:id
- **200 OK**: Item obtenido exitosamente
- **400 Bad Request**: ID inválido (UUID malformado)
- **404 Not Found**: Item no encontrado
- **500 Internal Server Error**: Error de lectura del Read Model
- **503 Service Unavailable**: Cache no disponible

### GET /api/v1/inventory/items/sku/:sku
- **200 OK**: Item obtenido exitosamente
- **400 Bad Request**: SKU vacío
- **404 Not Found**: Item no encontrado
- **500 Internal Server Error**: Error de lectura del Read Model
- **503 Service Unavailable**: Cache no disponible

### GET /api/v1/inventory/items/:id/stock
- **200 OK**: Estado de stock obtenido exitosamente
- **400 Bad Request**: ID inválido (UUID malformado)
- **404 Not Found**: Item no encontrado
- **500 Internal Server Error**: Error de lectura del Read Model
- **503 Service Unavailable**: Cache no disponible

### GET /api/v1/health
- **200 OK**: Servicio operativo

---

## Mejores Prácticas

### Uso de Cache

1. **Cache Hit Rate**: Monitorear el cache hit rate para optimizar TTL
2. **Cache Keys**: Usar claves consistentes para mejor rendimiento
3. **TTL Strategy**: Usar TTL más cortos para datos que cambian frecuentemente (stock status)

### Paginación

1. **Page Size**: Usar page_size razonable (10-50) para mejor rendimiento
2. **Page Numbers**: No hacer requests a páginas muy altas si no es necesario
3. **Cache**: Las listas paginadas se cachean, cambios frecuentes pueden requerir invalidación

### Performance

1. **Cache First**: Siempre intentar obtener datos del cache primero
2. **Read Model**: Consultar Read Model solo en cache miss
3. **Escalabilidad**: El servicio es stateless y escalable horizontalmente

