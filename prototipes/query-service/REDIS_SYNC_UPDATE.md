# Actualización de Redis y Autenticación JWT

## Resumen de Cambios

### 1. Autenticación JWT Implementada ✅

- **Módulo de autenticación**: `internal/auth/jwt.go` y `internal/auth/auth_handler.go`
- **Middleware de autenticación**: `pkg/middleware/auth_middleware.go`
- **Endpoint de login**: `POST /api/v1/auth/login`
- **Endpoints protegidos**: Todos los endpoints de inventory requieren JWT
- **Swagger actualizado**: Incluye autenticación BearerAuth con ejemplos

### 2. Implementación Real de Redis ✅

- **Reemplazado**: Cache in-memory placeholder por implementación Redis real
- **Cliente Redis**: `github.com/go-redis/redis/v8` (compatible con Go 1.20)
- **Configuración**: Conexión a Redis en puerto 6379 (Docker)
- **Fallback**: Si Redis no está disponible, usa cache in-memory automáticamente

### 3. Estrategia de Sincronización Rápida ✅

- **Invalidación selectiva**: Invalida cache específico por item ID/SKU cuando está disponible
- **Invalidación por patrón**: Usa `DeleteByPattern` para invalidar múltiples claves relacionadas
- **Sincronización rápida**: 
  - Invalida cache de item específico (por ID y SKU)
  - Invalida cache de stock status
  - Invalida cache de listas (todas las páginas)
  - Esto asegura que los datos se actualicen rápidamente después de eventos

## Configuración de Redis

### Variables de Entorno

```env
# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=  # Opcional, si Redis requiere contraseña
REDIS_DB=0
USE_CACHE=true  # Habilitar cache (Redis)
CACHE_TTL=300   # TTL del cache en segundos (default: 5 minutos)

# JWT Configuration
JWT_SECRET=your-secret-key-change-in-production-min-32-chars
```

### Docker Compose (Redis)

El servicio está configurado para conectarse a Redis en:
- **Host**: `localhost`
- **Puerto**: `6379`
- **Password**: Opcional (configurado en `REDIS_PASSWORD`)

Ejemplo de configuración Docker Compose:
```yaml
redis:
  image: redis:7.2-alpine
  container_name: redis
  hostname: redis
  ports:
    - "6379:6379"
  command: >
    sh -c "
    if [ -n \"$$REDIS_PASSWORD\" ]; then
      redis-server --appendonly yes --requirepass \"$$REDIS_PASSWORD\"
    else
      redis-server --appendonly yes
    fi
    "
  environment:
    - REDIS_PASSWORD=${REDIS_PASSWORD:-}
  volumes:
    - redis-data:/data
  networks:
    - redis-network
  healthcheck:
    test: ["CMD", "redis-cli", "ping"]
    interval: 10s
    timeout: 5s
    retries: 5
  restart: unless-stopped
```

## Estrategia de Cache

### Claves de Cache

- `item:id:{id}` - Item por ID
- `item:sku:{sku}` - Item por SKU
- `stock:{id}` - Estado de stock (TTL más corto)
- `items:list:{page}:{pageSize}` - Lista paginada

### Invalidación de Cache

Cuando se recibe un evento de Kafka:

1. **Extrae item ID y SKU** del evento
2. **Invalida cache específico**:
   - `item:id:{id}` si ID está disponible
   - `item:sku:{sku}` si SKU está disponible
   - `stock:{id}` si ID está disponible
3. **Invalida cache de listas**: `items:list:*` (todas las páginas)
4. **Si no hay información específica**: Invalida todos los patrones relacionados

### Configuración de Pool de Conexiones

```go
PoolSize:     10,  // Número de conexiones en el pool
MinIdleConns: 5,   // Mínimo de conexiones idle
DialTimeout:  5 * time.Second,
ReadTimeout:  3 * time.Second,
WriteTimeout: 3 * time.Second,
MaxRetries:   3,
```

## Autenticación JWT

### Endpoint de Login

```bash
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "admin123"
}
```

**Respuesta:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "type": "Bearer",
  "expires_in": 600,
  "expires_at": "2024-01-15T12:00:00Z"
}
```

### Usuarios Disponibles

- `admin` / `admin123`
- `user` / `user123`
- `operator` / `operator123`

### Uso del Token

```bash
GET /api/v1/inventory/items
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### Endpoints Protegidos

Todos los endpoints de inventory requieren autenticación:
- `GET /api/v1/inventory/items`
- `GET /api/v1/inventory/items/:id`
- `GET /api/v1/inventory/items/sku/:sku`
- `GET /api/v1/inventory/items/:id/stock`

### Endpoints Públicos

- `GET /api/v1/health`
- `POST /api/v1/auth/login`

## Swagger UI

### Acceso

```
http://localhost:8081/swagger/index.html
```

### Autenticación en Swagger

1. Hacer login en `/api/v1/auth/login` con credenciales:
   - Usuario: `admin`
   - Password: `admin123`
2. Copiar el token de la respuesta
3. Hacer clic en el botón "Authorize" en Swagger UI
4. Pegar el token en el formato: `Bearer {token}`
5. Ahora puedes probar todos los endpoints protegidos directamente desde Swagger

## Mejoras de Rendimiento

### Cache-First Strategy

1. **Consulta cache primero**: Todas las consultas intentan obtener datos del cache
2. **Cache miss**: Si no hay cache, consulta la base de datos
3. **Actualización de cache**: Los datos obtenidos se almacenan en cache para próximas consultas
4. **TTL configurable**: Tiempo de vida del cache configurable (default: 5 minutos)

### Sincronización Rápida

- **Invalidación inmediata**: Cuando se recibe un evento, el cache se invalida inmediatamente
- **Invalidación selectiva**: Solo se invalida el cache relacionado con el item modificado
- **Invalidación de listas**: Las listas se invalidan para asegurar datos frescos

## Próximos Pasos

1. **Métricas**: Agregar métricas de cache hit rate
2. **Monitoreo**: Monitorear uso de Redis y conexiones
3. **Optimización**: Ajustar TTL según patrones de uso
4. **Producción**: Configurar Redis con contraseña y TLS

