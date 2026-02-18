# Query Service

Servicio de consultas (API de lectura) para el sistema de inventario basado en arquitectura CQRS + EDA. Optimizado para baja latencia y alta escalabilidad.

## 📋 Descripción

Este servicio implementa el patrón CQRS (Command Query Responsibility Segregation) y Event-Driven Architecture (EDA) para manejar todas las operaciones de **lectura** del sistema de inventario. Es parte de una arquitectura distribuida que separa las responsabilidades de escritura (Command Service) y lectura (Query Service).

### Características Principales

- **Stateless**: Diseñado para escalabilidad horizontal
- **Cache-First**: Optimizado para lectura rápida desde Redis Cache
- **Baja Latencia**: Respuestas ultra-rápidas para consultas frecuentes
- **Alta Escalabilidad**: Puede escalarse horizontalmente sin problemas
- **Read Model**: Lee desde un modelo de lectura optimizado (SQLite/Read Database)
- **JWT/OAuth2 Authentication**: Autenticación mediante tokens JWT (10 minutos de expiración)
- **X-Request-ID**: Trazabilidad mediante X-Request-ID en todos los requests
- **Logging estructurado**: Usando zap para logging estructurado
- **Graceful shutdown**: Manejo adecuado de cierre del servidor
- **Documentación Swagger**: Documentación interactiva de la API con Swagger UI

## 🏗️ Arquitectura

Este servicio está diseñado para ser **stateless** y **altamente escalable**, enfocándose en servir datos rápidamente desde el Redis Cache (Read Model).

### Estructura del Proyecto

```
query-service/
├── cmd/
│   └── api/                 # Punto de entrada de la aplicación
│       └── main.go
├── internal/
│   ├── handlers/            # HTTP handlers (Gin) - solo GET
│   │   ├── inventory_handler.go
│   │   ├── inventory_handler_test.go
│   │   └── models.go
│   ├── models/              # Read models
│   │   └── inventory.go
│   ├── cache/               # Cache layer (Redis)
│   │   └── redis_cache.go
│   ├── repository/          # Read repository (Read Model)
│   │   ├── read_repository.go
│   │   └── sqlite_repository.go
│   ├── kafka/               # Kafka consumer para invalidación de cache
│   │   └── consumer.go
│   ├── auth/                # Autenticación JWT
│   │   ├── jwt.go
│   │   ├── auth_handler.go
│   │   └── auth_handler_test.go
│   └── config/              # Configuración de la aplicación
│       └── config.go
├── pkg/
│   ├── logger/              # Utilidades de logging
│   │   └── logger.go
│   ├── middleware/          # Middleware de Gin
│   │   ├── auth_middleware.go
│   │   ├── auth_middleware_test.go
│   │   ├── error_handler.go
│   │   ├── request_id.go
│   │   └── request_id_test.go
│   └── errors/              # Manejo de errores estandarizado
│       └── errors.go
├── docs/                     # Documentación Swagger generada
│   ├── docs.go
│   ├── swagger.json
│   ├── swagger.yaml
│   ├── EXAMPLES.md          # Ejemplos de requests/responses
│   ├── ERRORS.md            # Documentación de errores
│   └── REQUEST_ID.md        # Documentación de X-Request-ID
├── scripts/                  # Scripts de pruebas
│   ├── test_query_service.sh     # Pruebas de endpoints
│   ├── test_request_id.sh        # Pruebas de X-Request-ID
│   └── README_REQUEST_ID_TESTS.md
├── test-results/             # Resultados de pruebas
│   ├── README.md
│   ├── run_tests.sh
│   ├── test_integration.sh
│   ├── TESTING_SUMMARY.md
│   └── [TIMESTAMP]/          # Ejecuciones con timestamp
├── go.mod
├── go.sum
├── README.md                 # Este archivo
└── TEST_RESULTS.md           # Resultados consolidados de pruebas
```

## 🚀 Inicio Rápido

### Prerrequisitos

- **Go 1.20 o superior** - [Descargar Go](https://golang.org/dl/)
- **Redis** (opcional pero recomendado) - Para cache distribuido
- **SQLite** - Incluido en Go (no requiere instalación adicional)
- **Terminal/Command Line** - Para ejecutar comandos

### Instalación y Ejecución

#### 1. Navegar al Proyecto

```bash
cd query-service
```

#### 2. Instalar Dependencias

```bash
go mod download
```

#### 3. Configurar Variables de Entorno (Opcional)

Crea un archivo `.env` en la raíz del proyecto:

```env
PORT=8081
ENVIRONMENT=development
JWT_SECRET=your-secret-key-change-in-production-min-32-chars

# Redis Configuration (opcional)
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
USE_CACHE=true
CACHE_TTL=300

# SQLite Configuration (Read Model)
SQLITE_PATH=../listener-service/inventory.db

# Kafka Configuration (para invalidación de cache)
USE_KAFKA=true
KAFKA_BROKERS=localhost:9093
KAFKA_TOPIC_ITEMS=inventory.items
KAFKA_TOPIC_STOCK=inventory.stock
KAFKA_GROUP_ID=query-service
```

**Nota:** El servicio funciona con valores por defecto. Redis es opcional pero recomendado para mejor rendimiento. Si Redis no está disponible, el servicio usa cache in-memory como fallback.

#### 4. Ejecutar el Servicio

```bash
go run cmd/api/main.go
```

El servicio se iniciará en `http://localhost:8081`

#### 5. Verificar que el Servicio Está Corriendo

```bash
# Health check
curl http://localhost:8081/api/v1/health

# Respuesta esperada:
# {"status":"ok","service":"query-service"}
```

#### 6. Acceder a la Documentación Swagger

Abre en tu navegador:
- `http://localhost:8081/swagger/index.html`

## 🔐 Autenticación

### Obtener Token JWT

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

### Usar Token en Requests

```bash
GET /api/v1/inventory/items
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### Usuarios Disponibles

- `admin` / `admin123`
- `user` / `user123`
- `operator` / `operator123`

## 🔄 X-Request-ID y Trazabilidad

### Generación Automática

Si no se proporciona `X-Request-ID`, el servidor genera automáticamente un UUID y lo retorna en el header de respuesta.

### Trazabilidad

El `X-Request-ID` se usa principalmente para trazabilidad y correlación de logs:

- Todos los logs incluyen `request_id` para facilitar la trazabilidad
- El mismo `X-Request-ID` puede usarse en múltiples requests para correlacionar logs
- El `X-Request-ID` siempre está presente en los headers de respuesta

### Ejemplo

```bash
# Request con X-Request-ID
curl -X GET http://localhost:8081/api/v1/inventory/items \
  -H "Authorization: Bearer <token>" \
  -H "X-Request-ID: 550e8400-e29b-41d4-a716-446655440000"

# Response incluye X-Request-ID en header
# X-Request-ID: 550e8400-e29b-41d4-a716-446655440000
```

Ver `docs/REQUEST_ID.md` para más detalles.

## 📡 Endpoints

### Health Check
- `GET /api/v1/health` - Verifica el estado del servicio (público)

### Swagger Documentation
- `GET /swagger/index.html` - Documentación interactiva de la API (Swagger UI)

### Autenticación
- `POST /api/v1/auth/login` - Obtener token JWT (público)

### Inventory Query Operations (Requieren JWT)
- `GET /api/v1/inventory/items` - Listar items de inventario (paginado)
- `GET /api/v1/inventory/items/:id` - Obtener item por ID
- `GET /api/v1/inventory/items/sku/:sku` - Obtener item por SKU
- `GET /api/v1/inventory/items/:id/stock` - Obtener estado de stock

Todos los endpoints soportan `X-Request-ID` para trazabilidad.

## ⚙️ Configuración

El servicio se configura mediante variables de entorno:

| Variable | Descripción | Default | Requerido |
|----------|-------------|---------|-----------|
| `PORT` | Puerto del servidor HTTP | `8081` | No |
| `ENVIRONMENT` | Ambiente de ejecución (`development`/`production`) | `development` | No |
| `JWT_SECRET` | Secret para firmar tokens JWT | `your-secret-key-change-in-production-min-32-chars` | No |
| `REDIS_HOST` | Host de Redis | `localhost` | No* |
| `REDIS_PORT` | Puerto de Redis | `6379` | No* |
| `REDIS_PASSWORD` | Contraseña de Redis | `` | No* |
| `REDIS_DB` | Base de datos de Redis | `0` | No* |
| `USE_CACHE` | Habilitar cache (Redis) | `true` | No |
| `CACHE_TTL` | TTL del cache en segundos | `300` (5 minutos) | No |
| `SQLITE_PATH` | Ruta al archivo SQLite (Read Model) | `../listener-service/inventory.db` | No |
| `USE_KAFKA` | Habilitar Kafka consumer para invalidación de cache | `true` | No |
| `KAFKA_BROKERS` | Brokers de Kafka (comma-separated) | `localhost:9093` | No* |
| `KAFKA_TOPIC_ITEMS` | Topic para eventos de items | `inventory.items` | No |
| `KAFKA_TOPIC_STOCK` | Topic para eventos de stock | `inventory.stock` | No |
| `KAFKA_GROUP_ID` | Consumer group ID | `query-service` | No |

\* *Opcional. Si Redis no está disponible, el servicio usa cache in-memory como fallback.*

## 🎯 Optimizaciones de Rendimiento

### Cache Strategy

- **Cache-First**: Todas las consultas intentan primero obtener datos del cache
- **TTL Configurable**: Tiempo de vida del cache configurable (default: 5 minutos)
- **Cache Keys**: Claves optimizadas por tipo de consulta:
  - `item:id:{id}` - Item por ID
  - `item:sku:{sku}` - Item por SKU
  - `stock:{id}` - Estado de stock (TTL más corto)
  - `items:list:{page}:{pageSize}` - Lista paginada

### Sincronización Rápida

Cuando se recibe un evento de Kafka:

1. **Extrae item ID y SKU** del evento
2. **Invalida cache específico**:
   - `item:id:{id}` si ID está disponible
   - `item:sku:{sku}` si SKU está disponible
   - `stock:{id}` si ID está disponible
3. **Invalida cache de listas**: `items:list:*` (todas las páginas)
4. **Si no hay información específica**: Invalida todos los patrones relacionados

Esto asegura que los datos se actualicen rápidamente después de eventos.

### Escalabilidad

- **Stateless**: Sin estado compartido, escalable horizontalmente
- **Cache Distribuido**: Redis permite cache compartido entre instancias
- **Read Model Optimizado**: Modelo de lectura optimizado para consultas rápidas

## 📚 Documentación Adicional

El proyecto incluye documentación detallada en la carpeta `docs/`:

- **`docs/EXAMPLES.md`** - Ejemplos detallados de requests y responses válidos e inválidos para cada endpoint
- **`docs/ERRORS.md`** - Documentación completa de errores comunes, códigos de respuesta HTTP y manejo de errores
- **`docs/REQUEST_ID.md`** - Documentación de X-Request-ID y trazabilidad

### Códigos de Respuesta HTTP

- **200 OK** - Operación exitosa, datos obtenidos (pueden venir del cache o Read Model)
- **400 Bad Request** - Request inválido (parámetros de paginación inválidos, ID/SKU inválido)
- **401 Unauthorized** - No autorizado (token JWT inválido o faltante)
- **404 Not Found** - Recurso no encontrado
- **500 Internal Server Error** - Error interno del servidor (error de lectura o conexión a base de datos)
- **503 Service Unavailable** - Servicio no disponible (error de conexión al cache)

### Errores Comunes

- **Validación**: ID inválido (UUID malformado), SKU vacío, parámetros de paginación inválidos
- **Recurso No Encontrado**: Item no encontrado por ID o SKU
- **Conexión al Cache**: Cache (Redis) no disponible (usa fallback in-memory)
- **Base de Datos**: Error de lectura del Read Model

Ver `docs/ERRORS.md` para detalles completos.

## 🧪 Pruebas

### Pruebas Unitarias

```bash
# Ejecutar todas las pruebas
./test-results/run_tests.sh

# O manualmente
go test ./internal/handlers ./internal/auth ./pkg/middleware -v
```

**Resultados:**
- **Total de pruebas:** 25
- **Exitosas:** 25 ✅
- **Cobertura total:** ~69.4% (Handlers: 68.3%, Auth: 79.6%, Middleware: 59.5%)

Ver `TEST_RESULTS.md` para resultados consolidados de todas las pruebas.

### Pruebas de Integración

```bash
# Pruebas de integración end-to-end
./test-results/test_integration.sh

# Pruebas de endpoints
./scripts/test_query_service.sh

# Pruebas de X-Request-ID
./scripts/test_request_id.sh
```

### Scripts de Pruebas Disponibles

- `test-results/run_tests.sh` - Ejecutar pruebas unitarias con reportes
- `test-results/test_integration.sh` - Pruebas de integración end-to-end
- `scripts/test_query_service.sh` - Pruebas de endpoints
- `scripts/test_request_id.sh` - Pruebas de X-Request-ID y trazabilidad

Ver `TEST_RESULTS.md` para resultados consolidados de todas las pruebas.

## 📝 Notas Importantes

### Estado Actual

- ✅ **Implementado**: API REST completa con documentación Swagger
- ✅ **Implementado**: Arquitectura CQRS + EDA con separación de capas
- ✅ **Implementado**: Cache layer con Redis (con fallback in-memory)
- ✅ **Implementado**: Endpoints optimizados para lectura
- ✅ **Implementado**: Autenticación JWT/OAuth2
- ✅ **Implementado**: X-Request-ID para trazabilidad
- ✅ **Implementado**: Kafka consumer para invalidación de cache
- ✅ **Implementado**: Lectura desde SQLite (Read Model)

### Arquitectura Distribuida

Este servicio está diseñado para trabajar junto con:

1. **Command Service** (puerto 8080) - Maneja operaciones de escritura
2. **Query Service** (este servicio, puerto 8081) - Maneja operaciones de lectura
3. **Listener Service** (puerto 8082) - Procesa eventos y actualiza SQLite
4. **Event Broker** - Kafka para comunicación entre servicios
5. **Redis Cache** - Cache distribuido para alta performance
6. **SQLite Database** - Read Model optimizado para lectura

### Eventual Consistency

En arquitectura CQRS + EDA, los cambios pueden tardar unos segundos en estar disponibles en el Query Service después de ser procesados por el Listener Service. Esto es normal y esperado.

### Próximos Pasos de Implementación

1. **Métricas**: Agregar métricas de performance y cache hit rate
2. **Tests**: Mejorar cobertura de tests (actualmente 69.4%)
3. **Optimización**: Ajustar TTL según patrones de uso
4. **Producción**: Configurar Redis con contraseña y TLS

## 🐛 Troubleshooting

### El servicio no inicia

- Verificar que el puerto 8081 no esté en uso
- Verificar que Go esté instalado correctamente: `go version`
- Verificar que las dependencias estén instaladas: `go mod download`

### Error 401 en endpoints

- Verificar que se esté enviando el token JWT en el header `Authorization: Bearer <token>`
- Verificar que el token no haya expirado (10 minutos)
- Obtener un nuevo token desde `/api/v1/auth/login`

### Cache no funciona

- Verificar que Redis esté corriendo (si se usa Redis real)
- Verificar la configuración de Redis en variables de entorno
- Verificar los logs para errores de conexión
- **Nota:** Si Redis no está disponible, el servicio usa cache in-memory automáticamente

### Error 404 en endpoints

- Verificar que el servidor esté corriendo
- Verificar que estés usando la ruta correcta: `/api/v1/...`
- Verificar los logs del servidor para más detalles
- **Nota:** En arquitectura CQRS + EDA, puede haber eventual consistency (items recién creados pueden tardar unos segundos en estar disponibles)

### Error al generar documentación Swagger

- Verificar que swag esté instalado: `swag --version`
- Instalar swag: `go install github.com/swaggo/swag/cmd/swag@latest`
- Regenerar documentación: `swag init -g cmd/api/main.go -o ./docs`

## 📚 Recursos Adicionales

- **Swagger UI**: `http://localhost:8081/swagger/index.html`
- **Ejemplos**: Ver `docs/EXAMPLES.md`
- **Errores**: Ver `docs/ERRORS.md`
- **X-Request-ID**: Ver `docs/REQUEST_ID.md`
- **Pruebas**: Ver `TEST_RESULTS.md`
- **Command Service**: Ver `../command-service/README.md`
- **Listener Service**: Ver `../listener-service/README.md`
