# Pruebas de X-Request-ID e Idempotencia - Command Service

## Descripción

Este script (`test_request_id.sh`) realiza pruebas exhaustivas del sistema de control de duplicidad de requests mediante `X-Request-ID` en el Command Service.

## Pruebas Incluidas

### 1. Generación Automática de X-Request-ID
- **Objetivo**: Verificar que el sistema genera automáticamente un UUID cuando no se proporciona `X-Request-ID`
- **Endpoint**: `GET /api/v1/health`
- **Validación**: Verifica que el header `X-Request-ID` está presente en la respuesta

### 2. Uso de X-Request-ID Proporcionado
- **Objetivo**: Verificar que el sistema usa el `X-Request-ID` proporcionado por el cliente
- **Endpoint**: `GET /api/v1/health`
- **Validación**: Compara el `X-Request-ID` enviado con el retornado en la respuesta

### 3. Idempotencia - Crear Item
- **Objetivo**: Verificar que requests duplicados con el mismo `X-Request-ID` retornan la respuesta cacheada
- **Endpoint**: `POST /api/v1/inventory/items`
- **Validación**: 
  - Primera request procesa normalmente (HTTP 201)
  - Segunda request con mismo `X-Request-ID` retorna respuesta cacheada (HTTP 200)
  - Las respuestas deben ser idénticas

### 4. Idempotencia - Ajustar Stock
- **Objetivo**: Verificar idempotencia en operaciones de ajuste de stock
- **Endpoint**: `POST /api/v1/inventory/items/:id/adjust`
- **Validación**: Requests duplicados retornan la misma respuesta cacheada

### 5. X-Request-ID en Headers de Respuesta
- **Objetivo**: Verificar que `X-Request-ID` siempre está presente en los headers de respuesta
- **Validación**: Verifica presencia y corrección del header

### 6. Requests Diferentes con Mismo X-Request-ID
- **Objetivo**: Verificar que el sistema detecta duplicados basándose en `X-Request-ID`, no en el contenido
- **Endpoint**: `POST /api/v1/inventory/items`
- **Validación**: 
  - Dos requests con mismo `X-Request-ID` pero contenido diferente
  - Debe retornar la respuesta cacheada de la primera request

## Uso

```bash
# Ejecutar todas las pruebas
./scripts/test_request_id.sh

# Con configuración personalizada
COMMAND_SERVICE=http://localhost:8080 \
JWT_USERNAME=admin \
JWT_PASSWORD=admin123 \
./scripts/test_request_id.sh
```

## Variables de Entorno

- `COMMAND_SERVICE`: URL del Command Service (default: `http://localhost:8080`)
- `JWT_USERNAME`: Usuario para autenticación (default: `admin`)
- `JWT_PASSWORD`: Contraseña para autenticación (default: `admin123`)

## Requisitos

- `curl` instalado
- `uuidgen` o `python3` para generar UUIDs (opcional, el script tiene fallback)
- Command Service ejecutándose y accesible
- Autenticación JWT configurada

## Salida Esperada

```
╔════════════════════════════════════════════════════════════╗
║  Pruebas de X-Request-ID e Idempotencia - Command Service ║
╚════════════════════════════════════════════════════════════╝

🔐 Obteniendo token JWT...
✅ Token JWT obtenido exitosamente

=== Test 1: Generación automática de X-Request-ID ===
✓ PASS: Generación automática de X-Request-ID

=== Test 2: Uso de X-Request-ID proporcionado ===
✓ PASS: Uso de X-Request-ID proporcionado

...

╔════════════════════════════════════════════════════════════╗
║                      RESUMEN DE PRUEBAS                     ║
╚════════════════════════════════════════════════════════════╝
Total de pruebas: 6
Pruebas exitosas: 6
Pruebas fallidas: 0

✅ Todas las pruebas pasaron exitosamente
```

## Notas Importantes

1. **TTL de Idempotencia**: Las respuestas cacheadas expiran después de 5 minutos
2. **Operaciones de Escritura**: Solo las operaciones POST, PUT, DELETE, PATCH verifican idempotencia
3. **GET Requests**: No se verifican para idempotencia, pero sí incluyen `X-Request-ID` para trazabilidad
4. **Almacenamiento**: El almacenamiento es in-memory y se pierde al reiniciar el servicio

