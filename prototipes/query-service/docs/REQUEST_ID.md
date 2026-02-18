# X-Request-ID Header - Query Service

## Descripción

El Query Service implementa control de trazabilidad mediante el header `X-Request-ID`. Este mecanismo permite:

1. **Trazabilidad**: Rastrear requests a través de múltiples servicios
2. **Correlación**: Correlacionar logs y eventos relacionados
3. **Idempotencia**: Para operaciones de escritura (si las hay)

## Funcionamiento

### Generación de Request ID

- Si el cliente **proporciona** `X-Request-ID` en el header, se usa ese ID
- Si el cliente **no proporciona** `X-Request-ID`, el servidor genera un nuevo UUID
- El `X-Request-ID` siempre se retorna en el header de respuesta

### Idempotencia

Para operaciones de escritura (POST, PUT, DELETE, PATCH) - principalmente para consistencia:

1. **Primera Request**: Se procesa normalmente y se almacena la respuesta (TTL: 5 minutos)
2. **Request Duplicada**: Si se envía el mismo `X-Request-ID` dentro del TTL, se retorna la respuesta cacheada

**Nota**: El Query Service es principalmente de lectura (GET), por lo que la idempotencia es menos crítica pero se implementa para consistencia.

## Uso

### Ejemplo: Consultar Items con Request ID

```bash
# Request con X-Request-ID
curl -X GET http://localhost:8081/api/v1/inventory/items \
  -H "Authorization: Bearer <token>" \
  -H "X-Request-ID: 550e8400-e29b-41d4-a716-446655440000"

# Response incluye X-Request-ID en header
# X-Request-ID: 550e8400-e29b-41d4-a716-446655440000
```

### Ejemplo: Sin Request ID (Generación Automática)

```bash
# Request sin X-Request-ID
curl -X GET http://localhost:8081/api/v1/inventory/items \
  -H "Authorization: Bearer <token>"

# Response incluye X-Request-ID generado automáticamente
# X-Request-ID: <nuevo-uuid-generado>
```

## Endpoints que Soportan Request ID

Todos los endpoints soportan `X-Request-ID` para trazabilidad:

- `GET /api/v1/inventory/items` - Listar items
- `GET /api/v1/inventory/items/:id` - Obtener item por ID
- `GET /api/v1/inventory/items/sku/:sku` - Obtener item por SKU
- `GET /api/v1/inventory/items/:id/stock` - Obtener estado de stock

## Logs

Todos los logs incluyen el `request_id` para facilitar la trazabilidad:

```
INFO HTTP Request
  method=GET
  path=/api/v1/inventory/items
  status=200
  latency=12ms
  client_ip=127.0.0.1
  request_id=550e8400-e29b-41d4-a716-446655440000
```

## Notas Importantes

1. **Trazabilidad**: El `X-Request-ID` se usa principalmente para trazabilidad en Query Service
2. **Idempotencia**: Solo aplica a operaciones de escritura (si las hay)
3. **Almacenamiento Temporal**: El almacenamiento es in-memory y se pierde al reiniciar el servicio

