# X-Request-ID Header - Command Service

## Descripción

El Command Service implementa control de duplicidad de requests mediante el header `X-Request-ID`. Este mecanismo permite:

1. **Trazabilidad**: Rastrear requests a través de múltiples servicios
2. **Idempotencia**: Evitar procesamiento duplicado de requests
3. **Correlación**: Correlacionar logs y eventos relacionados

## Funcionamiento

### Generación de Request ID

- Si el cliente **proporciona** `X-Request-ID` en el header, se usa ese ID
- Si el cliente **no proporciona** `X-Request-ID`, el servidor genera un nuevo UUID
- El `X-Request-ID` siempre se retorna en el header de respuesta

### Idempotencia

Para operaciones de escritura (POST, PUT, DELETE, PATCH):

1. **Primera Request**: Se procesa normalmente y se almacena la respuesta (TTL: 5 minutos)
2. **Request Duplicada**: Si se envía el mismo `X-Request-ID` dentro del TTL, se retorna la respuesta cacheada sin procesar nuevamente

### Almacenamiento

- **Tipo**: In-memory (por defecto)
- **TTL**: 5 minutos
- **Limpieza**: Automática cada minuto

## Uso

### Ejemplo: Crear Item con Request ID

```bash
# Primera request
curl -X POST http://localhost:8080/api/v1/inventory/items \
  -H "Authorization: Bearer <token>" \
  -H "X-Request-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{
    "sku": "SKU-001",
    "name": "Test Item",
    "quantity": 100
  }'

# Response incluye X-Request-ID en header
# X-Request-ID: 550e8400-e29b-41d4-a716-446655440000
```

### Ejemplo: Request Duplicada (Idempotencia)

```bash
# Segunda request con el mismo X-Request-ID (dentro de 5 minutos)
curl -X POST http://localhost:8080/api/v1/inventory/items \
  -H "Authorization: Bearer <token>" \
  -H "X-Request-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{
    "sku": "SKU-001",
    "name": "Test Item",
    "quantity": 100
  }'

# Response: Retorna la respuesta cacheada (HTTP 200)
# No se procesa nuevamente, evitando duplicados
```

### Ejemplo: Sin Request ID (Generación Automática)

```bash
# Request sin X-Request-ID
curl -X POST http://localhost:8080/api/v1/inventory/items \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "sku": "SKU-002",
    "name": "Test Item 2",
    "quantity": 50
  }'

# Response incluye X-Request-ID generado automáticamente
# X-Request-ID: <nuevo-uuid-generado>
```

## Endpoints que Soportan Idempotencia

Todos los endpoints de escritura soportan idempotencia:

- `POST /api/v1/inventory/items` - Crear item
- `PUT /api/v1/inventory/items/:id` - Actualizar item
- `DELETE /api/v1/inventory/items/:id` - Eliminar item
- `POST /api/v1/inventory/items/:id/adjust` - Ajustar stock
- `POST /api/v1/inventory/items/:id/reserve` - Reservar stock
- `POST /api/v1/inventory/items/:id/release` - Liberar stock

## Logs

Todos los logs incluyen el `request_id` para facilitar la trazabilidad:

```
INFO HTTP Request
  method=POST
  path=/api/v1/inventory/items
  status=201
  latency=45ms
  client_ip=127.0.0.1
  request_id=550e8400-e29b-41d4-a716-446655440000
```

## Notas Importantes

1. **TTL**: Las respuestas cacheadas expiran después de 5 minutos
2. **Solo Operaciones de Escritura**: GET, HEAD, OPTIONS no se verifican para idempotencia
3. **Fail Open**: Si hay un error al verificar idempotencia, la request se procesa normalmente
4. **Almacenamiento Temporal**: El almacenamiento es in-memory y se pierde al reiniciar el servicio

## Próximos Pasos

- Implementar almacenamiento en Redis para persistencia entre reinicios
- Configurar TTL por endpoint
- Agregar métricas de idempotencia (cache hits/misses)

