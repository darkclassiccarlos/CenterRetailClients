# Errores Comunes - Command Service API

Este documento describe los errores comunes que pueden ocurrir al usar la API del Command Service.

## Códigos de Respuesta HTTP

### 200 OK
Operación exitosa. Retorna los datos solicitados.

### 201 Created
Recurso creado exitosamente. Retorna el recurso creado con su ID.

### 202 Accepted
Comando aceptado para procesamiento asíncrono. El comando será procesado en segundo plano.

### 400 Bad Request
Request inválido. El cliente debe corregir el request antes de reintentar.

### 404 Not Found
Recurso no encontrado. El ID proporcionado no existe en el sistema.

### 409 Conflict
Conflicto de estado. Generalmente por duplicidad o violación de reglas de negocio.

### 500 Internal Server Error
Error interno del servidor. El servidor encontró un error inesperado.

### 503 Service Unavailable
Servicio no disponible. Generalmente por problemas de conexión con dependencias (base de datos, event broker).

---

## Errores de Validación (400 Bad Request)

### Campos Requeridos Faltantes

**Error:** `Key: 'CreateItemRequest.SKU' Error:Field validation for 'SKU' failed on the 'required' tag`

**Causa:** Se intentó crear un item sin proporcionar el campo SKU (requerido).

**Solución:** Incluir todos los campos requeridos en el request.

**Ejemplo de Request Inválido:**
```json
{
  "name": "Product Name",
  "quantity": 100
}
```

**Ejemplo de Request Válido:**
```json
{
  "sku": "SKU-001",
  "name": "Product Name",
  "quantity": 100
}
```

---

### Valores Inválidos

**Error:** `Key: 'CreateItemRequest.Quantity' Error:Field validation for 'Quantity' failed on the 'min' tag`

**Causa:** Se proporcionó un valor que no cumple con las validaciones (ej: cantidad negativa).

**Solución:** Asegurarse de que los valores cumplan con las reglas de validación:
- `quantity` debe ser >= 0 para creación
- `quantity` debe ser >= 1 para reserva/liberación

**Ejemplo de Request Inválido:**
```json
{
  "sku": "SKU-001",
  "name": "Product Name",
  "quantity": -10
}
```

**Ejemplo de Request Válido:**
```json
{
  "sku": "SKU-001",
  "name": "Product Name",
  "quantity": 100
}
```

---

### ID Inválido (UUID malformado)

**Error:** `invalid item id`

**Causa:** Se proporcionó un ID que no es un UUID válido.

**Solución:** Usar un UUID válido en el formato: `550e8400-e29b-41d4-a716-446655440000`

**Ejemplo de Request Inválido:**
```
PUT /api/v1/inventory/items/invalid-id
```

**Ejemplo de Request Válido:**
```
PUT /api/v1/inventory/items/550e8400-e29b-41d4-a716-446655440000
```

---

## Errores de Duplicidad (409 Conflict)

### SKU Duplicado

**Error:** `SKU already exists`

**Causa:** Se intentó crear un item con un SKU que ya existe en el sistema.

**Solución:** Usar un SKU único o actualizar el item existente.

**Ejemplo de Response:**
```json
{
  "error": "SKU already exists"
}
```

---

## Errores de Integridad (400 Bad Request)

### Stock Insuficiente

**Error:** `insufficient stock available`

**Causa:** Se intentó reservar o ajustar stock más allá de lo disponible.

**Solución:** Verificar el stock disponible antes de realizar la operación.

**Ejemplo de Request Inválido:**
```json
{
  "quantity": 200
}
```

Cuando el stock disponible es solo 100.

**Ejemplo de Response:**
```json
{
  "error": "insufficient stock available"
}
```

---

### Cantidad a Liberar Excede lo Reservado

**Error:** `invalid release quantity`

**Causa:** Se intentó liberar más stock del que está reservado.

**Solución:** Verificar la cantidad reservada antes de liberar.

**Ejemplo de Request Inválido:**
```json
{
  "quantity": 200
}
```

Cuando solo hay 20 unidades reservadas.

**Ejemplo de Response:**
```json
{
  "error": "invalid release quantity"
}
```

---

## Errores de Recurso No Encontrado (404 Not Found)

### Item No Encontrado

**Error:** `item not found`

**Causa:** Se intentó acceder a un item que no existe en el sistema.

**Solución:** Verificar que el ID del item sea correcto y que el item exista.

**Ejemplo de Response:**
```json
{
  "error": "item not found"
}
```

---

## Errores de Conexión al Broker (503 Service Unavailable)

### Event Broker No Disponible

**Error:** `failed to publish event: connection to event broker failed`

**Causa:** No se puede establecer conexión con el event broker (Kafka, RabbitMQ, etc.).

**Solución:** 
1. Verificar que el event broker esté corriendo
2. Verificar la configuración de `EVENT_BROKER`
3. Verificar conectividad de red

**Ejemplo de Response:**
```json
{
  "error": "failed to publish event: connection to event broker failed"
}
```

**Nota:** En algunos casos, el comando puede haberse ejecutado exitosamente pero el evento no se publicó. El sistema registra el error para procesamiento posterior.

---

## Errores de Base de Datos (500 Internal Server Error)

### Error de Persistencia

**Error:** `failed to save item`

**Causa:** Error al guardar en la base de datos (conexión perdida, constraint violation, etc.).

**Solución:**
1. Verificar que la base de datos esté disponible
2. Verificar la configuración de conexión
3. Revisar los logs del servidor para más detalles

**Ejemplo de Response:**
```json
{
  "error": "failed to save item"
}
```

---

### Error de Conexión a Base de Datos

**Error:** `database connection failed`

**Causa:** No se puede establecer conexión con la base de datos.

**Solución:**
1. Verificar que la base de datos esté corriendo
2. Verificar la configuración de conexión (DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME)
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

### Retry Strategy

Para errores transitorios (503, 500), se recomienda:
1. Implementar retry con backoff exponencial
2. Verificar el estado del servicio antes de reintentar
3. No reintentar para errores de validación (400) sin corregir el request

---

## Códigos de Estado por Endpoint

### POST /api/v1/inventory/items
- **201 Created**: Item creado exitosamente
- **400 Bad Request**: Validación fallida, campos faltantes, valores inválidos
- **409 Conflict**: SKU duplicado
- **500 Internal Server Error**: Error de persistencia o conexión a base de datos
- **503 Service Unavailable**: Event broker no disponible

### PUT /api/v1/inventory/items/:id
- **200 OK**: Item actualizado exitosamente
- **400 Bad Request**: ID inválido, validación fallida
- **404 Not Found**: Item no encontrado
- **500 Internal Server Error**: Error de persistencia
- **503 Service Unavailable**: Event broker no disponible

### DELETE /api/v1/inventory/items/:id
- **200 OK**: Item eliminado exitosamente
- **400 Bad Request**: ID inválido
- **404 Not Found**: Item no encontrado
- **500 Internal Server Error**: Error de persistencia
- **503 Service Unavailable**: Event broker no disponible

### POST /api/v1/inventory/items/:id/adjust
- **200 OK**: Stock ajustado exitosamente
- **400 Bad Request**: ID inválido, cantidad faltante, stock insuficiente
- **404 Not Found**: Item no encontrado
- **500 Internal Server Error**: Error de persistencia
- **503 Service Unavailable**: Event broker no disponible

### POST /api/v1/inventory/items/:id/reserve
- **200 OK**: Stock reservado exitosamente
- **400 Bad Request**: ID inválido, cantidad inválida, stock insuficiente
- **404 Not Found**: Item no encontrado
- **500 Internal Server Error**: Error de persistencia
- **503 Service Unavailable**: Event broker no disponible

### POST /api/v1/inventory/items/:id/release
- **200 OK**: Stock liberado exitosamente
- **400 Bad Request**: ID inválido, cantidad inválida, cantidad excede lo reservado
- **404 Not Found**: Item no encontrado
- **500 Internal Server Error**: Error de persistencia
- **503 Service Unavailable**: Event broker no disponible

### GET /api/v1/health
- **200 OK**: Servicio operativo

