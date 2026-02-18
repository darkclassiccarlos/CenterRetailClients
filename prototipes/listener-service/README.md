# Listener Service

Servicio de procesamiento de eventos (Event Processor) para el sistema de inventario basado en arquitectura CQRS + EDA. Procesa eventos de Kafka y actualiza la base de datos SQLite (Single Writer Principle) con bloqueo optimista.

## 📋 Descripción

Este servicio implementa el patrón **Single Writer Principle** y **Optimistic Locking** para procesar eventos de Kafka y actualizar la base de datos SQLite (Inventory Database - Fuente de Verdad).

**Arquitectura Centralizada:** El inventario está centralizado en una base de datos SQLite que es la **única fuente autorizada** para realizar cambios en el stock. Las tiendas ya no escriben localmente y luego sincronizan; ahora deben llamar a la API central (Command Service) para reservar/descontar stock.

### Características Principales

- **Single Writer Principle**: Solo el Listener Service puede escribir en la base de datos transaccional
- **Optimistic Locking**: Usa version/timestamp para manejar concurrencia
- **Event Processing**: Consume eventos de Kafka y actualiza el Read Model
- **Retry Logic**: Reintentos automáticos con backoff exponencial
- **Dead Letter Queue**: Manejo de eventos fallidos (placeholder)
- **Graceful Shutdown**: Cierre ordenado del servicio
- **REST API para Monitoreo**: Endpoints de monitoreo y estadísticas (puerto 8082)

## 🏗️ Arquitectura

Este servicio es el **único escritor** de la base de datos SQLite (Inventory Database), garantizando consistencia y evitando conflictos de escritura.

### Estructura del Proyecto

```
listener-service/
├── cmd/
│   ├── listener/          # Punto de entrada del event processor
│   │   └── main.go
│   └── api/                # Punto de entrada del REST API (monitoreo)
│       └── main.go
├── internal/
│   ├── config/             # Configuración de la aplicación
│   │   └── config.go
│   ├── database/           # SQLite database (Single Writer)
│   │   └── sqlite.go
│   ├── events/             # Event processor
│   │   └── event_processor.go
│   ├── kafka/              # Kafka consumer
│   │   └── consumer.go
│   └── handlers/           # HTTP handlers para monitoreo
│       ├── monitoring_handler.go
│       └── models.go
├── pkg/
│   ├── logger/             # Utilidades de logging
│   │   └── logger.go
│   ├── middleware/         # Middleware de Gin
│   │   └── error_handler.go
│   └── errors/             # Manejo de errores estandarizado
│       └── errors.go
├── docs/                    # Documentación Swagger generada
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
├── go.mod
├── go.sum
├── README.md                # Este archivo
├── SCHEMA.md                # Esquema de base de datos SQLite
└── TEST_RESULTS.md          # Resultados consolidados de pruebas
```

## 🚀 Inicio Rápido

### Prerrequisitos

- **Go 1.20 o superior** - [Descargar Go](https://golang.org/dl/)
- **Kafka** - Para consumir eventos (opcional para desarrollo)
- **SQLite** - Incluido en Go (no requiere instalación adicional)

### Instalación y Ejecución

#### 1. Navegar al Proyecto

```bash
cd listener-service
```

#### 2. Instalar Dependencias

```bash
go mod download
```

#### 3. Configurar Variables de Entorno (Opcional)

Crea un archivo `.env` en la raíz del proyecto:

```env
ENVIRONMENT=development

# Kafka Configuration
KAFKA_BROKERS=localhost:9093
KAFKA_TOPIC_ITEMS=inventory.items
KAFKA_TOPIC_STOCK=inventory.stock
KAFKA_GROUP_ID=listener-service
KAFKA_AUTO_COMMIT=false

# SQLite Configuration
SQLITE_PATH=./inventory.db

# Retry Configuration
MAX_RETRIES=3
RETRY_DELAY_MS=1000

# Dead Letter Queue
DEAD_LETTER_QUEUE=true
DLQ_TOPIC=inventory.dlq

# REST API (Monitoreo)
API_PORT=8082
```

#### 4. Ejecutar el Event Processor

```bash
go run cmd/listener/main.go
```

El servicio se iniciará y comenzará a consumir eventos de Kafka.

#### 5. Ejecutar el REST API (Monitoreo) - Opcional

En otra terminal:

```bash
go run cmd/api/main.go
```

El API de monitoreo se iniciará en `http://localhost:8082`

#### 6. Verificar que el Servicio Está Corriendo

```bash
# Health check (REST API)
curl http://localhost:8082/api/v1/health

# Estadísticas
curl http://localhost:8082/api/v1/monitoring/stats
```

## 📡 Endpoints de Monitoreo

### Health Check
- `GET /api/v1/health` - Verifica el estado del servicio

### Monitoreo
- `GET /api/v1/monitoring/stats` - Estadísticas de procesamiento de eventos
- `GET /api/v1/monitoring/health` - Health check detallado

### Swagger Documentation
- `GET /swagger/index.html` - Documentación interactiva de la API (Swagger UI)

## ⚙️ Configuración

El servicio se configura mediante variables de entorno:

| Variable | Descripción | Default | Requerido |
|----------|-------------|---------|-----------|
| `ENVIRONMENT` | Ambiente de ejecución (`development`/`production`) | `development` | No |
| `KAFKA_BROKERS` | Brokers de Kafka (comma-separated) | `localhost:9093` | No* |
| `KAFKA_TOPIC_ITEMS` | Topic para eventos de items | `inventory.items` | No |
| `KAFKA_TOPIC_STOCK` | Topic para eventos de stock | `inventory.stock` | No |
| `KAFKA_GROUP_ID` | Consumer group ID | `listener-service` | No |
| `KAFKA_AUTO_COMMIT` | Auto commit de offsets | `false` | No |
| `SQLITE_PATH` | Ruta al archivo SQLite | `./inventory.db` | No |
| `MAX_RETRIES` | Máximo número de reintentos | `3` | No |
| `RETRY_DELAY_MS` | Delay entre reintentos (ms) | `1000` | No |
| `DEAD_LETTER_QUEUE` | Habilitar DLQ | `true` | No |
| `DLQ_TOPIC` | Topic para DLQ | `inventory.dlq` | No |
| `API_PORT` | Puerto del REST API (monitoreo) | `8082` | No |

\* *Requerido cuando se use Kafka real*

## 🔒 Single Writer Principle

Este servicio implementa el **Single Writer Principle** para garantizar que solo un proceso escriba en la base de datos SQLite:

- **Mutex**: Usa un mutex para serializar todas las escrituras
- **Connection Pool**: Configurado con `MaxOpenConns=1` para un solo escritor
- **Atomic Operations**: Todas las operaciones de escritura son atómicas

## 🔄 Optimistic Locking

El servicio usa **Optimistic Locking** con version/timestamp:

- **Version Field**: Cada registro tiene un campo `version` que se incrementa en cada actualización
- **Version Check**: Antes de actualizar, se verifica que la versión coincida
- **Conflict Detection**: Si la versión no coincide, se retorna `ErrOptimisticLockFailed`
- **Retry Logic**: Los conflictos se manejan con reintentos automáticos

### Ejemplo de Optimistic Locking

```sql
UPDATE inventory_items
SET quantity = quantity + ?, version = version + 1, updated_at = ?
WHERE id = ? AND version = ? AND (quantity + ?) >= 0
```

Si la versión no coincide, la actualización falla y se reintenta.

## 🔄 Retry Logic

El servicio implementa retry logic con backoff exponencial:

- **Max Retries**: Configurable (default: 3)
- **Retry Delay**: Delay incremental entre reintentos
- **Optimistic Lock Failures**: Se reintentan automáticamente
- **Other Errors**: Se reintentan según configuración

## 📨 Dead Letter Queue

El servicio puede enviar eventos fallidos a un Dead Letter Queue:

- **DLQ Enabled**: Configurable (default: true)
- **DLQ Topic**: Configurable (default: `inventory.dlq`)
- **Failed Events**: Eventos que fallan después de todos los reintentos

**Nota:** Actualmente es un placeholder, pendiente de implementación real.

## 📊 Eventos Procesados

El servicio procesa los siguientes eventos:

### Items Events
- **InventoryItemCreated**: Crea un nuevo item
- **InventoryItemUpdated**: Actualiza un item existente
- **InventoryItemDeleted**: Elimina un item

### Stock Events
- **StockAdjusted**: Ajusta la cantidad de stock
- **StockReserved**: Reserva stock
- **StockReleased**: Libera stock reservado

## 🎯 Flujo de Procesamiento

1. **Consume Event**: El consumer recibe un evento de Kafka
2. **Extract Event Type**: Extrae el tipo de evento de los headers
3. **Process Event**: Procesa el evento con retry logic
4. **Update Database**: Actualiza SQLite con optimistic locking
5. **Handle Failures**: Envía a DLQ si falla después de reintentos
6. **Commit Offset**: Marca el mensaje como procesado

## 🐛 Correcciones Implementadas

### Bug en `processStockAdjusted`

**Problema:** El método `processStockAdjusted` estaba calculando incorrectamente el ajuste de stock.

**Código Anterior (Incorrecto):**
```go
adjustment := event.Quantity - currentItem.Quantity
```

**Problema:** El evento `StockAdjustedEvent` que se publica desde Command Service tiene:
- `Quantity`: El ajuste (diferencia) que se quiere aplicar (ej: +25 o -30)
- `NewTotal`: La nueva cantidad total después del ajuste

El código anterior estaba interpretando `event.Quantity` como la cantidad total nueva, cuando en realidad es el ajuste.

**Código Corregido:**
```go
// The event.Quantity field contains the adjustment (difference), not the new total
// For example: if stock was 100 and we adjust by +25, event.Quantity = 25
adjustment := event.Quantity
```

**Resultado:** Los valores de stock ahora se actualizan correctamente en todo el flujo.

## 🗄️ Esquema de Base de Datos

El servicio usa SQLite como base de datos única fuente de verdad. Ver `SCHEMA.md` para detalles completos del esquema.

### Tablas Principales

- **`stores`**: Información sobre las tiendas físicas
- **`inventory_items`**: Inventario centralizado (Single Source of Truth)
- **`store_reservations`**: Reservas de stock por tienda

## 🧪 Pruebas

### Estado Actual

- ⚠️ **Pruebas Unitarias**: Pendientes de implementación
- ⚠️ **Pruebas de Integración**: Pendientes de implementación
- ✅ **Pruebas E2E**: Verificadas manualmente en flujo completo

Ver `TEST_RESULTS.md` para más detalles sobre el estado de las pruebas.

## 📝 Notas Importantes

### Estado Actual

- ✅ **Implementado**: Single Writer Principle con mutex
- ✅ **Implementado**: Optimistic Locking con version
- ✅ **Implementado**: Event processing para todos los eventos
- ✅ **Implementado**: Retry logic con backoff
- ✅ **Implementado**: REST API para monitoreo
- ✅ **Implementado**: Swagger documentation
- ⚠️ **Placeholder**: DLQ producer (pendiente implementación real)
- ⚠️ **Pendiente**: Pruebas unitarias y de integración

### Arquitectura Distribuida

Este servicio es parte de la arquitectura CQRS + EDA:

1. **Command Service** (puerto 8080) - Publica eventos a Kafka
2. **Query Service** (puerto 8081) - Lee desde Read Model/Cache
3. **Listener Service** (este servicio) - Procesa eventos y actualiza SQLite
4. **Kafka** - Event Broker
5. **SQLite** - Inventory Database (Fuente de Verdad)

### Próximos Pasos de Implementación

1. **DLQ Producer**: Implementar producer real para Dead Letter Queue
2. **Metrics**: Agregar métricas de procesamiento
3. **Monitoring**: Agregar monitoreo y alertas
4. **Tests**: Unit tests e integration tests
5. **Documentación**: Mejorar documentación de eventos procesados

## 🐛 Troubleshooting

### El servicio no inicia

- Verificar que Kafka esté corriendo
- Verificar que los topics existan
- Verificar la configuración de Kafka

### Errores de optimistic locking

- Normal en alta concurrencia
- El servicio reintenta automáticamente
- Verificar logs para más detalles

### Base de datos bloqueada

- Verificar que solo una instancia esté corriendo
- Verificar permisos de escritura en SQLite
- Verificar que no haya otros procesos escribiendo

### Eventos no se procesan

- Verificar que Kafka esté corriendo
- Verificar que los topics existan
- Verificar el consumer group ID
- Verificar los logs para errores

## 📚 Recursos Adicionales

- **Esquema de Base de Datos**: Ver `SCHEMA.md` para detalles del esquema SQLite
- **Swagger UI**: `http://localhost:8082/swagger/index.html`
- **Pruebas**: Ver `TEST_RESULTS.md`
- **Command Service**: Ver `../command-service/README.md`
- **Query Service**: Ver `../query-service/README.md`
- **Arquitectura CQRS**: Ver documentación de arquitectura distribuida
