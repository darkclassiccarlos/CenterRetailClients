# Command Service

Servicio de comandos (API de escritura) para el sistema de inventario basado en arquitectura CQRS + EDA.

## 📋 Descripción

Este servicio implementa el patrón CQRS (Command Query Responsibility Segregation) y Event-Driven Architecture (EDA) para manejar todas las operaciones de **escritura** del sistema de inventario. Es parte de una arquitectura distribuida que separa las responsabilidades de escritura (Command Service) y lectura (Query Service).

### Características Principales

- **Arquitectura CQRS**: Separación clara entre comandos (escritura) y queries (lectura)
- **Event-Driven**: Publicación de eventos de dominio para desacoplamiento
- **Domain-Driven Design**: Modelos de dominio con lógica de negocio encapsulada
- **Clean Architecture**: Separación de capas (handlers, domain, repository)
- **JWT/OAuth2 Authentication**: Autenticación mediante tokens JWT (10 minutos de expiración)
- **X-Request-ID**: Control de duplicidad de requests mediante idempotencia
- **Logging estructurado**: Usando zap para logging estructurado
- **Graceful shutdown**: Manejo adecuado de cierre del servidor
- **Documentación Swagger**: Documentación interactiva de la API con Swagger UI

## 🏗️ Arquitectura

Este servicio implementa el patrón CQRS (Command Query Responsibility Segregation) y Event-Driven Architecture (EDA) para manejar todas las operaciones de escritura del sistema de inventario.

### Estructura del Proyecto

```
command-service/
├── cmd/
│   └── api/                 # Punto de entrada de la aplicación
│       └── main.go
├── internal/
│   ├── handlers/            # HTTP handlers (Gin)
│   │   ├── inventory_handler.go
│   │   ├── inventory_handler_test.go
│   │   └── models.go
│   ├── commands/            # Command objects (CQRS)
│   │   └── inventory_commands.go
│   ├── domain/              # Domain models y lógica de negocio
│   │   ├── inventory.go
│   │   └── inventory_test.go
│   ├── repository/          # Interfaces y implementaciones de persistencia
│   │   └── inventory_repository.go
│   ├── events/              # Eventos de dominio y publisher
│   │   ├── event_publisher.go
│   │   ├── kafka_publisher.go
│   │   └── kafka_publisher_test.go
│   ├── auth/                # Autenticación JWT
│   │   ├── jwt.go
│   │   └── auth_handler.go
│   └── config/              # Configuración de la aplicación
│       └── config.go
├── pkg/
│   ├── logger/              # Utilidades de logging
│   │   └── logger.go
│   ├── middleware/          # Middleware de Gin
│   │   ├── auth_middleware.go
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
│   ├── EVENTS.md            # Documentación de eventos publicados
│   └── REQUEST_ID.md        # Documentación de X-Request-ID
├── scripts/                  # Scripts de pruebas
│   ├── run_tests.sh         # Ejecutar pruebas unitarias
│   ├── test_e2e_flow.sh     # Pruebas end-to-end
│   ├── test_stock_operations.sh  # Pruebas de operaciones de stock
│   ├── test_release_stock.sh     # Pruebas de liberación de stock
│   ├── test_request_id.sh        # Pruebas de X-Request-ID
│   └── README_REQUEST_ID_TESTS.md
├── test-results/             # Resultados de pruebas
│   ├── README.md
│   ├── SUMMARY.md
│   └── [TIMESTAMP]/          # Ejecuciones con timestamp
├── go.mod
├── go.sum
├── README.md                 # Este archivo
├── TESTING.md                # Guía de pruebas unitarias
└── TEST_RESULTS.md           # Resultados consolidados de pruebas
```

## 🚀 Inicio Rápido

### Prerrequisitos

- **Go 1.20 o superior** - [Descargar Go](https://golang.org/dl/)
- **Kafka** (opcional para desarrollo) - Para event broker
- **Terminal/Command Line** - Para ejecutar comandos

### Instalación y Ejecución

#### 1. Navegar al Proyecto

```bash
cd command-service
```

#### 2. Instalar Dependencias

```bash
go mod download
```

#### 3. Configurar Variables de Entorno (Opcional)

Crea un archivo `.env` en la raíz del proyecto:

```env
PORT=8080
ENVIRONMENT=development
JWT_SECRET=your-secret-key-change-in-production-min-32-chars

# Kafka Configuration
KAFKA_BROKERS=localhost:9093
KAFKA_TOPIC_ITEMS=inventory.items
KAFKA_TOPIC_STOCK=inventory.stock
KAFKA_CLIENT_ID=command-service
KAFKA_ACKS=all
KAFKA_RETRIES=3
```

**Nota:** Actualmente el servicio usa implementaciones in-memory para el repositorio y event publisher, por lo que no requiere base de datos ni Kafka para funcionar.

#### 4. Ejecutar el Servicio

```bash
go run cmd/api/main.go
```

El servicio se iniciará en `http://localhost:8080`

#### 5. Verificar que el Servicio Está Corriendo

```bash
# Health check
curl http://localhost:8080/api/v1/health

# Respuesta esperada:
# {"status":"ok","service":"command-service"}
```

#### 6. Acceder a la Documentación Swagger

Abre en tu navegador:
- `http://localhost:8080/swagger/index.html`

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

## 🔄 X-Request-ID e Idempotencia

### Generación Automática

Si no se proporciona `X-Request-ID`, el servidor genera automáticamente un UUID y lo retorna en el header de respuesta.

### Idempotencia

Para operaciones de escritura (POST, PUT, DELETE, PATCH):

1. **Primera Request**: Se procesa normalmente y se almacena la respuesta (TTL: 5 minutos)
2. **Request Duplicada**: Si se envía el mismo `X-Request-ID` dentro del TTL, se retorna la respuesta cacheada sin procesar nuevamente

### Ejemplo

```bash
# Primera request
curl -X POST http://localhost:8080/api/v1/inventory/items \
  -H "Authorization: Bearer <token>" \
  -H "X-Request-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{"sku": "SKU-001", "name": "Test Item", "quantity": 100}'

# Segunda request (duplicada) - retorna respuesta cacheada
curl -X POST http://localhost:8080/api/v1/inventory/items \
  -H "Authorization: Bearer <token>" \
  -H "X-Request-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -H "Content-Type: application/json" \
  -d '{"sku": "SKU-001", "name": "Test Item", "quantity": 100}'
```

Ver `docs/REQUEST_ID.md` para más detalles.

## 📡 Endpoints

### Health Check
- `GET /api/v1/health` - Verifica el estado del servicio (público)

### Swagger Documentation
- `GET /swagger/index.html` - Documentación interactiva de la API (Swagger UI)

### Autenticación
- `POST /api/v1/auth/login` - Obtener token JWT (público)

### Inventory Operations (Requieren JWT)
- `POST /api/v1/inventory/items` - Crear un nuevo item de inventario
- `PUT /api/v1/inventory/items/:id` - Actualizar un item de inventario
- `DELETE /api/v1/inventory/items/:id` - Eliminar un item de inventario
- `POST /api/v1/inventory/items/:id/adjust` - Ajustar stock
- `POST /api/v1/inventory/items/:id/reserve` - Reservar stock
- `POST /api/v1/inventory/items/:id/release` - Liberar stock reservado

Todos los endpoints de inventario soportan `X-Request-ID` para idempotencia.

## ⚙️ Configuración

El servicio se configura mediante variables de entorno:

| Variable | Descripción | Default | Requerido |
|----------|-------------|---------|-----------|
| `PORT` | Puerto del servidor HTTP | `8080` | No |
| `ENVIRONMENT` | Ambiente de ejecución (`development`/`production`) | `development` | No |
| `JWT_SECRET` | Secret para firmar tokens JWT | `your-secret-key-change-in-production-min-32-chars` | No |
| `KAFKA_BROKERS` | Brokers de Kafka (comma-separated) | `localhost:9093` | No* |
| `KAFKA_TOPIC_ITEMS` | Topic para eventos de items | `inventory.items` | No |
| `KAFKA_TOPIC_STOCK` | Topic para eventos de stock | `inventory.stock` | No |
| `KAFKA_CLIENT_ID` | Client ID de Kafka | `command-service` | No |
| `KAFKA_ACKS` | Nivel de acks (`0`, `1`, `all`) | `all` | No |
| `KAFKA_RETRIES` | Número de reintentos | `3` | No |

\* *Actualmente no requerido ya que el servicio usa implementaciones in-memory. Se requiere cuando se implemente Kafka real.*

## 📚 Documentación Adicional

El proyecto incluye documentación detallada en la carpeta `docs/`:

- **`docs/EXAMPLES.md`** - Ejemplos detallados de requests y responses válidos e inválidos para cada endpoint
- **`docs/ERRORS.md`** - Documentación completa de errores comunes, códigos de respuesta HTTP y manejo de errores
- **`docs/EVENTS.md`** - Documentación de eventos publicados: topics, formato, atributos obligatorios
- **`docs/REQUEST_ID.md`** - Documentación de X-Request-ID e idempotencia

### Códigos de Respuesta HTTP

- **200 OK** - Operación exitosa
- **201 Created** - Recurso creado exitosamente
- **202 Accepted** - Comando aceptado para procesamiento asíncrono
- **400 Bad Request** - Request inválido (validación fallida)
- **401 Unauthorized** - No autorizado (token JWT inválido o faltante)
- **404 Not Found** - Recurso no encontrado
- **409 Conflict** - Conflicto (duplicidad, etc.)
- **500 Internal Server Error** - Error interno del servidor
- **503 Service Unavailable** - Servicio no disponible (conexión a dependencias)

### Errores Comunes

- **Validación**: Campos requeridos faltantes, valores inválidos
- **Duplicidad**: SKU duplicado
- **Integridad**: Stock insuficiente, cantidad excede lo reservado
- **Conexión al Broker**: Event broker no disponible

Ver `docs/ERRORS.md` para detalles completos.

### Eventos Publicados

El servicio publica eventos de dominio a través de un Event Broker:

- **Topics**: `inventory.items`, `inventory.stock`
- **Formato**: JSON con estructura estándar (eventType, eventId, aggregateId, occurredAt, version, data)
- **Atributos Obligatorios**: eventType, eventId, aggregateId, occurredAt, version, data

**Tipos de eventos:**
- `InventoryItemCreated`, `InventoryItemUpdated`, `InventoryItemDeleted`
- `StockAdjusted`, `StockReserved`, `StockReleased`

Ver `docs/EVENTS.md` para detalles completos de cada evento.

## 🧪 Pruebas

### Pruebas Unitarias

```bash
# Ejecutar todas las pruebas
./scripts/run_tests.sh

# O manualmente
go test ./internal/... -v
```

**Resultados:**
- **Total de pruebas:** 33
- **Exitosas:** 33 ✅
- **Cobertura total:** ~51% (Domain: 96.6%, Handlers: 53.7%, Events: 28.0%)

Ver `TEST_RESULTS.md` para resultados consolidados de todas las pruebas.

### Pruebas End-to-End

```bash
# Pruebas de flujo completo
./scripts/test_e2e_flow.sh

# Pruebas de operaciones de stock
./scripts/test_stock_operations.sh

# Pruebas de liberación de stock
./scripts/test_release_stock.sh

# Pruebas de X-Request-ID e idempotencia
./scripts/test_request_id.sh
```

### Scripts de Pruebas Disponibles

- `scripts/run_tests.sh` - Ejecutar pruebas unitarias con reportes
- `scripts/test_e2e_flow.sh` - Pruebas end-to-end del flujo completo
- `scripts/test_stock_operations.sh` - Pruebas de operaciones de stock
- `scripts/test_release_stock.sh` - Pruebas de liberación de stock
- `scripts/test_request_id.sh` - Pruebas de X-Request-ID e idempotencia

Ver `TEST_RESULTS.md` para resultados consolidados de todas las pruebas.

## 🔧 Desarrollo

### Regenerar Documentación Swagger

Si modificas las anotaciones Swagger, regenera la documentación:

```bash
swag init -g cmd/api/main.go -o ./docs
```

### Estructura de Código

- **`internal/handlers/`** - HTTP handlers (Gin)
- **`internal/domain/`** - Modelos de dominio y lógica de negocio
- **`internal/commands/`** - Command objects (CQRS)
- **`internal/events/`** - Eventos de dominio y publisher
- **`internal/repository/`** - Interfaces y implementaciones de persistencia
- **`internal/auth/`** - Autenticación JWT
- **`pkg/middleware/`** - Middleware de Gin (auth, error handling, request ID)
- **`pkg/logger/`** - Utilidades de logging
- **`pkg/errors/`** - Manejo de errores estandarizado

## 📝 Notas Importantes

### Estado Actual

- ✅ **Implementado**: API REST completa con documentación Swagger
- ✅ **Implementado**: Arquitectura CQRS + EDA con separación de capas
- ✅ **Implementado**: Eventos de dominio definidos y documentados
- ✅ **Implementado**: Autenticación JWT/OAuth2
- ✅ **Implementado**: X-Request-ID e idempotencia
- ⚠️ **Placeholder**: Repositorio in-memory (pendiente implementación PostgreSQL)
- ⚠️ **Placeholder**: Event Publisher in-memory (pendiente implementación Kafka real)

### Arquitectura Distribuida

Este servicio está diseñado para ser parte de una arquitectura distribuida más grande:

1. **Command Service** (este servicio, puerto 8080) - Maneja operaciones de escritura
2. **Query Service** (puerto 8081) - Maneja operaciones de lectura
3. **Listener Service** (puerto 8082) - Procesa eventos y actualiza base de datos
4. **Event Broker** - Kafka para comunicación entre servicios
5. **Inventory Database** - Base de datos de escritura
6. **Read Model / Cache** - Modelo de lectura optimizado

### Próximos Pasos de Implementación

1. **Persistencia Real**: Reemplazar `InMemoryInventoryRepository` con PostgreSQL
2. **Event Broker**: Reemplazar `InMemoryEventPublisher` con Kafka real
3. **Validaciones**: Agregar validaciones de negocio más robustas
4. **Tests**: Mejorar cobertura de tests (actualmente 51%)
5. **Métricas**: Agregar métricas y observabilidad

## 🐛 Troubleshooting

### El servicio no inicia

- Verificar que el puerto 8080 no esté en uso
- Verificar que Go esté instalado correctamente: `go version`
- Verificar que las dependencias estén instaladas: `go mod download`

### Error 401 en endpoints

- Verificar que se esté enviando el token JWT en el header `Authorization: Bearer <token>`
- Verificar que el token no haya expirado (10 minutos)
- Obtener un nuevo token desde `/api/v1/auth/login`

### Error 404 en endpoints

- Verificar que el servidor esté corriendo
- Verificar que estés usando la ruta correcta: `/api/v1/...`
- Verificar los logs del servidor para más detalles

### Error al generar documentación Swagger

- Verificar que swag esté instalado: `swag --version`
- Instalar swag: `go install github.com/swaggo/swag/cmd/swag@latest`
- Regenerar documentación: `swag init -g cmd/api/main.go -o ./docs`

## 📚 Recursos Adicionales

- **Swagger UI**: `http://localhost:8080/swagger/index.html`
- **Ejemplos**: Ver `docs/EXAMPLES.md`
- **Errores**: Ver `docs/ERRORS.md`
- **Eventos**: Ver `docs/EVENTS.md`
- **X-Request-ID**: Ver `docs/REQUEST_ID.md`
- **Pruebas**: Ver `TEST_RESULTS.md`
- **Query Service**: Ver `../query-service/README.md`
- **Listener Service**: Ver `../listener-service/README.md`
