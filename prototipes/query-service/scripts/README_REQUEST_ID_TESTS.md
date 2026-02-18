# Pruebas de X-Request-ID y Trazabilidad - Query Service

## Descripción

Este script (`test_request_id.sh`) realiza pruebas del sistema de trazabilidad mediante `X-Request-ID` en el Query Service.

## Pruebas Incluidas

### 1. Generación Automática de X-Request-ID
- **Objetivo**: Verificar que el sistema genera automáticamente un UUID cuando no se proporciona `X-Request-ID`
- **Endpoint**: `GET /api/v1/health`
- **Validación**: Verifica que el header `X-Request-ID` está presente en la respuesta

### 2. Uso de X-Request-ID Proporcionado
- **Objetivo**: Verificar que el sistema usa el `X-Request-ID` proporcionado por el cliente
- **Endpoint**: `GET /api/v1/health`
- **Validación**: Compara el `X-Request-ID` enviado con el retornado en la respuesta

### 3. X-Request-ID en Consulta de Items
- **Objetivo**: Verificar que `X-Request-ID` se propaga correctamente en consultas de lista
- **Endpoint**: `GET /api/v1/inventory/items`
- **Validación**: Verifica presencia del header en la respuesta

### 4. X-Request-ID en Consulta por ID
- **Objetivo**: Verificar que `X-Request-ID` se propaga en consultas por ID
- **Endpoint**: `GET /api/v1/inventory/items/:id`
- **Validación**: Verifica presencia del header en la respuesta

### 5. X-Request-ID en Headers de Respuesta
- **Objetivo**: Verificar que `X-Request-ID` siempre está presente en los headers de respuesta
- **Validación**: Verifica presencia y corrección del header

### 6. Múltiples Requests con Mismo X-Request-ID
- **Objetivo**: Verificar trazabilidad con múltiples requests usando el mismo `X-Request-ID`
- **Endpoints**: Múltiples `GET /api/v1/inventory/items`
- **Validación**: Todas las requests retornan el mismo `X-Request-ID` para correlación

## Uso

```bash
# Ejecutar todas las pruebas
./scripts/test_request_id.sh

# Con configuración personalizada
QUERY_SERVICE=http://localhost:8081 \
JWT_USERNAME=admin \
JWT_PASSWORD=admin123 \
./scripts/test_request_id.sh
```

## Variables de Entorno

- `QUERY_SERVICE`: URL del Query Service (default: `http://localhost:8081`)
- `JWT_USERNAME`: Usuario para autenticación (default: `admin`)
- `JWT_PASSWORD`: Contraseña para autenticación (default: `admin123`)

## Requisitos

- `curl` instalado
- `uuidgen` o `python3` para generar UUIDs (opcional, el script tiene fallback)
- Query Service ejecutándose y accesible
- Autenticación JWT configurada
- Al menos un item en la base de datos para algunas pruebas

## Salida Esperada

```
╔════════════════════════════════════════════════════════════╗
║     Pruebas de X-Request-ID y Trazabilidad - Query Service║
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

1. **Trazabilidad**: El `X-Request-ID` se usa principalmente para trazabilidad en Query Service
2. **Idempotencia**: El Query Service es principalmente de lectura, por lo que la idempotencia es menos crítica
3. **Correlación**: El mismo `X-Request-ID` puede usarse en múltiples requests para correlacionar logs
4. **Logs**: Todos los logs incluyen `request_id` para facilitar la trazabilidad

