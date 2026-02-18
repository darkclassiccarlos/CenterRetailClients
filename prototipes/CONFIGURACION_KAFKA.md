# Configuración de Kafka - Arquitectura CQRS + EDA

Este documento describe la configuración de Kafka para todos los servicios de la arquitectura distribuida.

## 📋 Configuración de Servicios

### Command Service

**Archivo:** `command-service/.env`

```env
# Kafka Configuration
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC_ITEMS=inventory.items
KAFKA_TOPIC_STOCK=inventory.stock
KAFKA_CLIENT_ID=command-service
KAFKA_ACKS=all
KAFKA_RETRIES=3
```

**Función:** Publica eventos de inventario a Kafka cuando se realizan operaciones de escritura.

### Query Service

**Archivo:** `query-service/.env`

```env
# Kafka Configuration (for cache invalidation)
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC_ITEMS=inventory.items
KAFKA_TOPIC_STOCK=inventory.stock
KAFKA_GROUP_ID=query-service
KAFKA_AUTO_COMMIT=true
```

**Función:** Consume eventos de Kafka para invalidar cache cuando hay cambios en el inventario.

### Listener Service

**Archivo:** `listener-service/.env`

```env
# Kafka Configuration
KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC_ITEMS=inventory.items
KAFKA_TOPIC_STOCK=inventory.stock
KAFKA_GROUP_ID=listener-service
KAFKA_AUTO_COMMIT=false

# SQLite Configuration
SQLITE_PATH=./inventory.db

# Retry Configuration
MAX_RETRIES=3
RETRY_DELAY_MS=1000

# Dead Letter Queue Configuration
DEAD_LETTER_QUEUE=true
DLQ_TOPIC=inventory.dlq
```

**Función:** Consume eventos de Kafka y actualiza la base de datos SQLite (Single Writer Principle).

## 🔧 Configuración de Kafka

### Variables de Entorno

#### KAFKA_BROKERS
- **Descripción:** Lista de brokers de Kafka (separados por coma)
- **Formato:** `host:port` o `host1:port1,host2:port2`
- **Ejemplos:**
  - Desarrollo local: `localhost:9092`
  - Docker: `kafka:9092` (si está en la misma red Docker)
  - Producción: `kafka1:9092,kafka2:9092,kafka3:9092`

#### KAFKA_TOPIC_ITEMS
- **Descripción:** Topic para eventos de items de inventario
- **Valor por defecto:** `inventory.items`
- **Eventos:** `InventoryItemCreated`, `InventoryItemUpdated`, `InventoryItemDeleted`

#### KAFKA_TOPIC_STOCK
- **Descripción:** Topic para eventos de stock
- **Valor por defecto:** `inventory.stock`
- **Eventos:** `StockAdjusted`, `StockReserved`, `StockReleased`

#### KAFKA_CLIENT_ID (Command Service)
- **Descripción:** ID del cliente Kafka para el producer
- **Valor por defecto:** `command-service`
- **Uso:** Identifica el producer en los logs de Kafka

#### KAFKA_GROUP_ID (Query Service, Listener Service)
- **Descripción:** Consumer group ID
- **Valores:**
  - Query Service: `query-service`
  - Listener Service: `listener-service`
- **Uso:** Permite que múltiples instancias consuman del mismo topic

#### KAFKA_ACKS (Command Service)
- **Descripción:** Nivel de confirmación requerido del broker
- **Valores:** `0`, `1`, `all`
- **Recomendado:** `all` (mayor garantía de entrega)

#### KAFKA_RETRIES (Command Service)
- **Descripción:** Número de reintentos en caso de error
- **Valor por defecto:** `3`
- **Recomendado:** `3` para balance entre confiabilidad y latencia

#### KAFKA_AUTO_COMMIT (Query Service, Listener Service)
- **Descripción:** Auto-commit de offsets
- **Valores:** `true`, `false`
- **Query Service:** `true` (para invalidación de cache, no crítico)
- **Listener Service:** `false` (para garantizar procesamiento, crítico)

## 🐳 Configuración con Docker

Si Kafka está corriendo en un contenedor Docker:

### Opción 1: Kafka en la misma red Docker
```env
KAFKA_BROKERS=kafka:9092
```

### Opción 2: Kafka con puerto mapeado en localhost
```env
KAFKA_BROKERS=localhost:9092
```

### Opción 3: Kafka con múltiples brokers
```env
KAFKA_BROKERS=kafka1:9092,kafka2:9092,kafka3:9092
```

## 📊 Topics de Kafka

### inventory.items
- **Descripción:** Eventos relacionados con items de inventario
- **Eventos:**
  - `InventoryItemCreated`
  - `InventoryItemUpdated`
  - `InventoryItemDeleted`
- **Consumidores:**
  - Query Service (cache invalidation)
  - Listener Service (actualización de SQLite)

### inventory.stock
- **Descripción:** Eventos relacionados con stock
- **Eventos:**
  - `StockAdjusted`
  - `StockReserved`
  - `StockReleased`
- **Consumidores:**
  - Query Service (cache invalidation)
  - Listener Service (actualización de SQLite)

### inventory.dlq (Dead Letter Queue)
- **Descripción:** Eventos que fallaron después de todos los reintentos
- **Uso:** Para análisis y reprocesamiento manual
- **Configuración:** `DEAD_LETTER_QUEUE=true` en Listener Service

## 🔍 Verificación de Configuración

### Verificar que los archivos .env existen

```bash
# Command Service
ls -la command-service/.env

# Query Service
ls -la query-service/.env

# Listener Service
ls -la listener-service/.env
```

### Verificar que la configuración se carga correctamente

```bash
# Command Service
cd command-service
go run -c 'package main; import ("fmt"; "command-service/internal/config"); func main() { cfg := config.Load(); fmt.Printf("Kafka Brokers: %v\n", cfg.KafkaBrokers) }'

# Query Service
cd query-service
go run -c 'package main; import ("fmt"; "query-service/internal/config"); func main() { cfg := config.Load(); fmt.Printf("Kafka Brokers: %v\n", cfg.KafkaBrokers) }'

# Listener Service
cd listener-service
go run -c 'package main; import ("fmt"; "listener-service/internal/config"); func main() { cfg := config.Load(); fmt.Printf("Kafka Brokers: %v\n", cfg.KafkaBrokers) }'
```

## 🚀 Inicio de Servicios

### 1. Iniciar Kafka (si está en Docker)
```bash
docker-compose up -d kafka
```

### 2. Verificar que Kafka está corriendo
```bash
# Verificar logs
docker logs kafka

# Verificar conectividad
nc -zv localhost 9092
```

### 3. Iniciar servicios en orden

```bash
# 1. Listener Service (debe iniciar primero para procesar eventos)
cd listener-service
go run cmd/listener/main.go

# 2. Command Service (publica eventos)
cd command-service
go run cmd/api/main.go

# 3. Query Service (consume eventos para cache)
cd query-service
go run cmd/api/main.go
```

## 📝 Notas Importantes

1. **Orden de Inicio:** El Listener Service debe iniciar antes que el Command Service para procesar eventos inmediatamente.

2. **KAFKA_BROKERS:** 
   - Si Kafka está en un contenedor Docker, usa `kafka:9092` si los servicios están en la misma red Docker
   - Si Kafka está en localhost, usa `localhost:9092`
   - Para producción, usa múltiples brokers: `kafka1:9092,kafka2:9092,kafka3:9092`

3. **KAFKA_AUTO_COMMIT:**
   - `true` para Query Service (cache invalidation no es crítico)
   - `false` para Listener Service (procesamiento crítico, requiere commit manual)

4. **Topics:** Los topics se crean automáticamente si `auto.create.topics.enable=true` en Kafka, o deben crearse manualmente antes de iniciar los servicios.

5. **Sin Autenticación:** Los servicios están configurados sin autenticación. Para producción, agregar configuración de SASL/SSL.

## 🔒 Seguridad (Producción)

Para producción, agregar configuración de seguridad:

```env
# SASL Configuration
KAFKA_SASL_MECHANISM=PLAIN
KAFKA_SASL_USERNAME=username
KAFKA_SASL_PASSWORD=password

# SSL Configuration
KAFKA_SECURITY_PROTOCOL=SASL_SSL
KAFKA_SSL_CA_LOCATION=/path/to/ca-cert
```

## 📚 Recursos Adicionales

- **Command Service README:** `command-service/README.md`
- **Query Service README:** `query-service/README.md`
- **Listener Service README:** `listener-service/README.md`
- **Kafka Documentation:** https://kafka.apache.org/documentation/

