# ENTREGABLE - Sistema de Gestión de Inventario

## 📋 Resumen Ejecutivo

Este documento presenta la propuesta de migración de un sistema monolítico de gestión de inventario a una arquitectura distribuida basada en **CQRS (Command Query Responsibility Segregation)** y **Event-Driven Architecture (EDA)**, diseñada para resolver problemas críticos de consistencia, latencia y escalabilidad.

---

## 🏗️ Arquitectura Actual

### Descripción del Sistema Actual

La empresa mantiene un sistema de gestión de inventario para una cadena de tiendas minoristas con las siguientes características:

#### Componentes Principales

- **Backend Monolítico**: Aplicación única que maneja todas las operaciones (lectura y escritura)
- **Frontend Legacy**: Aplicación web antigua que consume el backend monolítico
- **Base de Datos Local por Tienda**: Cada tienda mantiene su propia base de datos local
- **Base de Datos Central**: Base de datos centralizada que almacena el inventario consolidado
- **Sincronización Periódica**: Proceso de sincronización que se ejecuta cada 15 minutos

#### Flujo de Operación Actual

```
┌─────────────┐         ┌─────────────┐         ┌─────────────┐
│   Tienda 1  │         │   Tienda 2  │         │   Tienda N  │
│  (DB Local) │         │  (DB Local) │         │  (DB Local) │
└──────┬──────┘         └──────┬──────┘         └──────┬──────┘
       │                       │                       │
       │  Sincronización       │  Sincronización       │
       │  cada 15 minutos      │  cada 15 minutos      │
       └───────────┬───────────┴───────────┬───────────┘
                   │                       │
                   ▼                       ▼
         ┌─────────────────────────┐
         │  Backend Monolítico     │
         │  (Lectura + Escritura)  │
         └───────────┬─────────────┘
                     │
                     ▼
         ┌─────────────────────────┐
         │  Base de Datos Central  │
         └─────────────────────────┘
```

### Problemas Identificados

#### 1. **Inconsistencias de Stock**
- **Problema**: Las actualizaciones de stock se sincronizan cada 15 minutos, generando ventanas de inconsistencia
- **Impacto**: Los clientes pueden ver stock disponible que ya fue vendido, o viceversa
- **Consecuencia**: Pérdida de ventas y frustración del cliente

#### 2. **Alta Latencia en Actualizaciones**
- **Problema**: Los cambios de stock no se reflejan en tiempo real
- **Impacto**: Experiencia de usuario degradada, especialmente en operaciones de alta frecuencia
- **Consecuencia**: Pérdida de competitividad frente a sistemas más ágiles

#### 3. **Escalabilidad Limitada**
- **Problema**: El backend monolítico no puede escalar independientemente para lectura y escritura
- **Impacto**: Cuellos de botella durante picos de carga
- **Consecuencia**: Necesidad de sobre-provisionar recursos

#### 4. **Acoplamiento Fuerte**
- **Problema**: Todas las operaciones están acopladas en un solo servicio
- **Impacto**: Cambios en una funcionalidad afectan a todo el sistema
- **Consecuencia**: Dificultad para mantener y evolucionar el sistema

#### 5. **Puntos Únicos de Falla**
- **Problema**: Si el backend monolítico falla, todo el sistema se detiene
- **Impacto**: Disponibilidad limitada
- **Consecuencia**: Pérdida de servicio durante mantenimientos o fallos

---

## 🚀 Arquitectura Propuesta

### Visión General

La nueva arquitectura implementa **CQRS + Event-Driven Architecture** para separar las responsabilidades de lectura y escritura, garantizar consistencia eventual y mejorar la escalabilidad.

#### Principios Arquitectónicos

1. **Separación de Responsabilidades**: Comandos (escritura) y Queries (lectura) en servicios independientes
2. **Event-Driven**: Comunicación asíncrona mediante eventos
3. **Single Writer Principle**: Solo un servicio escribe en la fuente de verdad
4. **Optimistic Locking**: Control de concurrencia sin bloqueos pesimistas
5. **Cache-First**: Optimización de lecturas mediante cache distribuido

### Componentes de la Arquitectura

```
┌─────────────────────────────────────────────────────────────────┐
│                        FRONTEND                                  │
│  (Consume Query Service para lectura y Command Service para     │
│   escritura - Reservas/Compras)                                 │
└───────────────┬───────────────────────────┬─────────────────────┘
                │                           │
                │ GET /stock                │ POST /reserve
                │                           │
        ┌───────▼────────┐         ┌────────▼────────┐
        │ Query Service  │         │ Command Service │
        │ (API Lectura)  │         │ (API Escritura) │
        │ Puerto: 8081   │         │ Puerto: 8080    │
        └───────┬────────┘         └────────┬────────┘
                │                           │
                │ Lee desde                 │ Publica eventos
                │                           │
        ┌───────▼────────┐         ┌────────▼────────┐
        │  Redis Cache   │         │  Event Broker   │
        │  (Read Model)  │         │     (Kafka)     │
        │   (Opcional)   │         │                 │
        └────────────────┘         └────────┬────────┘
                                             │
                                             │ Consume eventos
                                             │
                                    ┌────────▼────────┐
                                    │ Listener Service │
                                    │ (Event Processor)│
                                    └────────┬─────────┘
                                             │
                                             │ Escribe (Single Writer)
                                             │
                                    ┌────────▼────────┐
                                    │ Inventory DB    │
                                    │ (SQLite)        │
                                    │ Fuente de Verdad│
                                    └─────────────────┘
```

### 1. Command Service (API Escritura)

#### Responsabilidades

- **Recibe peticiones de escritura/comando** (POST /reserve, POST /adjust, etc.)
- **Valida la petición** según reglas de negocio
- **Publica eventos en el Event Broker** (Kafka)
- **Retorna 202 Accepted inmediatamente** (procesamiento asíncrono)

#### Características Técnicas

- **Puerto**: 8080
- **Protocolo**: HTTP REST
- **Respuesta**: 202 Accepted (asíncrono)
- **Eventos Publicados**:
  - `inventory.items.created`
  - `inventory.items.updated`
  - `inventory.items.deleted`
  - `inventory.stock.adjusted`
  - `inventory.stock.reserved`
  - `inventory.stock.released`

#### Flujo de Operación

```
1. Cliente → POST /api/v1/inventory/items/:id/reserve
2. Command Service valida la petición
3. Command Service publica evento StockReserved en Kafka
4. Command Service retorna 202 Accepted
5. Cliente recibe confirmación inmediata
```

#### Beneficios

- **Baja Latencia de Respuesta**: Respuesta inmediata sin esperar procesamiento
- **Alta Disponibilidad**: No bloquea operaciones de lectura
- **Desacoplamiento**: No depende de la base de datos para responder

---

### 2. Query Service (API Lectura)

#### Responsabilidades

- **Recibe peticiones de consulta** (GET /stock, GET /items, etc.)
- **Lee directamente de Redis Cache** para latencia mínima
- **Fallback a Read Model** si el cache no está disponible
- **Optimizado para alta frecuencia de lectura**

#### Características Técnicas

- **Puerto**: 8081
- **Protocolo**: HTTP REST
- **Estrategia**: Cache-First
- **Cache**: Redis (opcional, con fallback a in-memory)
- **TTL**: Configurable (default: 5 minutos)

#### Flujo de Operación

```
1. Cliente → GET /api/v1/inventory/items/:id/stock
2. Query Service consulta Redis Cache
3. Si existe en cache → Retorna inmediatamente
4. Si no existe → Consulta Read Model y actualiza cache
5. Retorna respuesta al cliente
```

#### Beneficios

- **Ultra Baja Latencia**: Respuestas en milisegundos desde cache
- **Alta Escalabilidad**: Stateless, escalable horizontalmente
- **Alta Disponibilidad**: Múltiples instancias pueden servir requests

---

### 3. Event Broker (Kafka)

#### Responsabilidades

- **Centraliza la comunicación** entre servicios
- **Garantiza fiabilidad** mediante persistencia de eventos
- **Tolerancia a fallos**: Los eventos persisten hasta ser procesados
- **Ordenamiento**: Mantiene el orden de eventos por partición

#### Características Técnicas

- **Tecnología**: Apache Kafka
- **Puertos**: 
  - 9092 (conexiones externas)
  - 9093 (conexiones externas alternativas)
  - 29093 (comunicación interna entre brokers)
- **Topics**:
  - `inventory.items` (3 particiones)
  - `inventory.stock` (3 particiones)
  - `inventory.dlq` (1 partición - Dead Letter Queue)
- **Visualización**: Kafdrop en puerto 9000

#### Configuración

```yaml
# docker-components/kafka/docker-compose.yml
- Zookeeper (puerto 2181)
- Kafka (puertos 9092, 9093, 29093)
- Kafdrop (puerto 9000) - Visualización web
- Kafka-init - Crea topics automáticamente
```

#### Beneficios

- **Fiabilidad**: Eventos no se pierden
- **Desacoplamiento**: Servicios no se conocen directamente
- **Escalabilidad**: Procesamiento paralelo mediante particiones
- **Auditoría**: Historial completo de eventos

---

### 4. Listener Service (Event Processor)

#### Responsabilidades

- **Consume eventos del Event Broker** (Kafka)
- **Es el único que escribe en la BD** (Single Writer Principle)
- **Aplica lógica de Bloqueo Optimista** para manejar concurrencia
- **Garantiza Consistencia Estricta (CP)** en la fuente de verdad

#### Características Técnicas

- **Single Writer Principle**: Solo este servicio escribe en SQLite
- **Optimistic Locking**: Usa campo `version` para control de concurrencia
- **Retry Logic**: Reintentos automáticos con backoff exponencial
- **Dead Letter Queue**: Manejo de eventos fallidos

#### Flujo de Procesamiento

```
1. Listener Service consume evento de Kafka
2. Extrae tipo de evento y datos
3. Aplica lógica de negocio (validaciones)
4. Intenta actualizar SQLite con optimistic locking
5. Si hay conflicto de versión → Reintenta
6. Si falla después de reintentos → Envía a DLQ
7. Commit del offset en Kafka
```

#### Optimistic Locking

```sql
UPDATE inventory_items
SET quantity = quantity + ?, 
    version = version + 1,
    updated_at = ?
WHERE id = ? 
  AND version = ? 
  AND (quantity + ?) >= 0
```

Si la versión no coincide, la actualización falla y se reintenta automáticamente.

#### Beneficios

- **Consistencia Garantizada**: Solo un escritor evita conflictos
- **Alta Concurrencia**: Optimistic locking permite procesamiento paralelo
- **Tolerancia a Fallos**: Reintentos automáticos y DLQ

---

### 5. Inventory Database (SQLite)

#### Responsabilidades

- **Fuente de Verdad (Single Source of Truth)**
- **Almacena el stock real** con control de versión
- **Solo escritura desde Listener Service** (Single Writer Principle)
- **Consistencia Estricta (CP)**

#### Características Técnicas

- **Tecnología**: SQLite
- **Esquema**: Tabla `inventory_items` con campo `version`
- **Control de Concurrencia**: Optimistic locking mediante `version`
- **Escritor Único**: Solo Listener Service puede escribir

#### Esquema de Base de Datos

```sql
CREATE TABLE inventory_items (
    id TEXT PRIMARY KEY,
    sku TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    quantity INTEGER NOT NULL DEFAULT 0,
    reserved_quantity INTEGER NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
```

#### Beneficios

- **Consistencia**: Garantiza que el stock siempre sea correcto
- **Simplicidad**: SQLite es ligero y fácil de mantener
- **Auditoría**: Campo `version` permite rastrear cambios

---

### 6. Read Model / Cache (Redis - Opcional)

#### Responsabilidades

- **Base de datos de solo lectura** optimizada para consultas rápidas
- **Se actualiza en tiempo real** impulsada por eventos desde el Listener
- **Cache distribuido** para alta disponibilidad
- **Fallback a in-memory** si Redis no está disponible

#### Características Técnicas

- **Tecnología**: Redis (opcional)
- **Estrategia**: Cache-First
- **TTL**: Configurable por tipo de dato
- **Actualización**: Event-driven desde Listener Service

#### Nota Importante

**La actualización de Redis es opcional**, por lo que los prototipos están diseñados para funcionar sin Redis, usando implementaciones in-memory como fallback. Esto permite:

- **Desarrollo sin dependencias externas**: Los servicios funcionan sin Redis
- **Flexibilidad**: Puede habilitarse Redis en producción para mejor performance
- **Resiliencia**: El sistema funciona aunque Redis falle

#### Beneficios

- **Ultra Baja Latencia**: Respuestas en milisegundos
- **Alta Escalabilidad**: Cache distribuido
- **Resiliencia**: Fallback automático si Redis no está disponible

---

## 📊 Comparación: Arquitectura Actual vs Propuesta

| Aspecto | Arquitectura Actual | Arquitectura Propuesta |
|---------|---------------------|------------------------|
| **Consistencia** | Eventual (sincronización cada 15 min) | Eventual con garantías (tiempo real) |
| **Latencia de Lectura** | Alta (consulta a BD central) | Ultra baja (cache Redis) |
| **Latencia de Escritura** | Alta (escritura síncrona) | Baja (202 Accepted inmediato) |
| **Escalabilidad** | Limitada (monolítico) | Alta (servicios independientes) |
| **Disponibilidad** | Punto único de falla | Alta (servicios desacoplados) |
| **Acoplamiento** | Fuerte | Bajo (event-driven) |
| **Mantenibilidad** | Difícil | Fácil (separación de responsabilidades) |
| **Tolerancia a Fallos** | Baja | Alta (eventos persistentes) |
| **Auditoría** | Limitada | Completa (historial de eventos) |

---

## 🎯 Justificación de la Migración

### Problemas Resueltos

#### 1. **Inconsistencias de Stock**
- **Solución**: Event-driven con procesamiento en tiempo real
- **Resultado**: Stock siempre consistente, sin ventanas de 15 minutos

#### 2. **Alta Latencia**
- **Solución**: Cache-First en Query Service + procesamiento asíncrono
- **Resultado**: Lecturas en milisegundos, escrituras con 202 Accepted

#### 3. **Escalabilidad Limitada**
- **Solución**: Servicios independientes escalables horizontalmente
- **Resultado**: Escalar lectura y escritura independientemente

#### 4. **Acoplamiento Fuerte**
- **Solución**: Event-driven architecture
- **Resultado**: Servicios desacoplados, cambios aislados

#### 5. **Puntos Únicos de Falla**
- **Solución**: Servicios independientes con fallbacks
- **Resultado**: Alta disponibilidad, degradación gradual

### Beneficios Adicionales

- **Mejor Experiencia de Usuario**: Respuestas instantáneas
- **Mayor Confiabilidad**: Eventos persistentes, no se pierden datos
- **Facilidad de Mantenimiento**: Código organizado por responsabilidades
- **Preparado para el Futuro**: Arquitectura moderna y escalable

---

## 📦 Estado del Proyecto y Prototipos

### Prototipos Implementados

#### 1. Command Service ✅

**Ubicación**: `prototipes/command-service/`

**Estado**: ✅ Implementado y funcional

**Características**:
- API REST completa con documentación Swagger
- Publicación de eventos a Kafka
- Validaciones de negocio
- Manejo de errores robusto
- Tests unitarios e integración

**Endpoints**:
- `POST /api/v1/inventory/items` - Crear item
- `POST /api/v1/inventory/items/:id/reserve` - Reservar stock
- `POST /api/v1/inventory/items/:id/release` - Liberar stock
- `POST /api/v1/inventory/items/:id/adjust` - Ajustar stock

**Puerto**: 8080

**Documentación**: `prototipes/command-service/docs/`

---

#### 2. Query Service ✅

**Ubicación**: `prototipes/query-service/`

**Estado**: ✅ Implementado y funcional

**Características**:
- API REST optimizada para lectura
- Cache-First con Redis (opcional, fallback a in-memory)
- Paginación
- Documentación Swagger
- Stateless y escalable

**Endpoints**:
- `GET /api/v1/inventory/items` - Listar items (paginado)
- `GET /api/v1/inventory/items/:id` - Obtener item por ID
- `GET /api/v1/inventory/items/:id/stock` - Obtener estado de stock

**Puerto**: 8081

**Documentación**: `prototipes/query-service/docs/`

---

#### 3. Listener Service ✅

**Ubicación**: `prototipes/listener-service/`

**Estado**: ✅ Implementado y funcional

**Características**:
- Consumo de eventos de Kafka
- Single Writer Principle
- Optimistic Locking
- Retry logic con backoff
- Dead Letter Queue (DLQ)
- Base de datos SQLite

**Funcionalidades**:
- Procesa eventos de items (created, updated, deleted)
- Procesa eventos de stock (adjusted, reserved, released)
- Actualiza SQLite con control de versión
- Manejo de errores y DLQ

**Documentación**: `prototipes/listener-service/SCHEMA.md`

---

#### 4. Event Broker (Kafka) ✅

**Ubicación**: `docker-components/kafka/`

**Estado**: ✅ Configurado y funcional

**Componentes**:
- Zookeeper (puerto 2181)
- Kafka (puertos 9092, 9093, 29093)
- Kafdrop (puerto 9000) - Visualización web
- Kafka-init - Inicialización automática de topics

**Topics Configurados**:
- `inventory.items` (3 particiones)
- `inventory.stock` (3 particiones)
- `inventory.dlq` (1 partición)

**Configuración**: `docker-components/kafka/docker-compose.yml`

---

### Componentes Opcionales

#### Redis Cache (Opcional)

**Estado**: ⚠️ Opcional - Los servicios funcionan sin Redis

**Implementación**:
- Query Service tiene interfaz para Redis
- Fallback automático a in-memory si Redis no está disponible
- Permite desarrollo sin dependencias externas

**Configuración**:
- Variables de entorno: `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`
- Si no se configura, usa implementación in-memory

---

## 🚀 Guía de Inicio Rápido para Evaluadores

### Prerrequisitos

- **Go 1.20 o superior**
- **Docker y Docker Compose**
- **Git**

### Paso 1: Levantar Kafka

```bash
cd docker-components/kafka
docker compose up -d
```

Verificar que Kafka esté corriendo:
- Kafdrop: http://localhost:9000
- Kafka: localhost:9092

### Paso 2: Ejecutar Listener Service

```bash
cd prototipes/listener-service
go run cmd/listener/main.go
```

Este servicio consumirá eventos de Kafka y actualizará SQLite.

### Paso 3: Ejecutar Command Service

```bash
cd prototipes/command-service
go run cmd/api/main.go
```

El servicio estará disponible en: http://localhost:8080
- Swagger: http://localhost:8080/swagger/index.html

### Paso 4: Ejecutar Query Service

```bash
cd prototipes/query-service
go run cmd/api/main.go
```

El servicio estará disponible en: http://localhost:8081
- Swagger: http://localhost:8081/swagger/index.html

### Paso 5: Probar el Flujo Completo

#### 1. Crear un item (Command Service)

```bash
curl -X POST http://localhost:8080/api/v1/inventory/items \
  -H "Content-Type: application/json" \
  -d '{
    "sku": "PROD-001",
    "name": "Producto de Prueba",
    "description": "Descripción del producto",
    "quantity": 100
  }'
```

#### 2. Consultar stock (Query Service)

```bash
curl http://localhost:8081/api/v1/inventory/items/sku/PROD-001/stock
```

#### 3. Reservar stock (Command Service)

```bash
curl -X POST http://localhost:8080/api/v1/inventory/items/{id}/reserve \
  -H "Content-Type: application/json" \
  -d '{
    "quantity": 10
  }'
```

#### 4. Verificar actualización (Query Service)

```bash
curl http://localhost:8081/api/v1/inventory/items/{id}/stock
```

### Visualización

- **Kafdrop**: http://localhost:9000 - Ver eventos en Kafka
- **Command Service Swagger**: http://localhost:8080/swagger/index.html
- **Query Service Swagger**: http://localhost:8081/swagger/index.html

---

## 📚 Documentación Adicional

### Documentación por Servicio

- **Command Service**: `prototipes/command-service/README.md`
- **Query Service**: `prototipes/query-service/README.md`
- **Listener Service**: `prototipes/listener-service/README.md`

### Documentación Técnica

- **Eventos**: `prototipes/command-service/docs/EVENTS.md`
- **Errores**: `prototipes/command-service/docs/ERRORS.md`
- **Ejemplos**: `prototipes/command-service/docs/EXAMPLES.md`
- **Esquema BD**: `prototipes/listener-service/SCHEMA.md`
- **Configuración Kafka**: `prototipes/CONFIGURACION_KAFKA.md`

### Diagramas

- **Arquitectura Distribuida**: `docs/arqDistribuida.png`

---

## 🎓 Conceptos Clave Implementados

### CQRS (Command Query Responsibility Segregation)

- **Separación clara** entre comandos (escritura) y queries (lectura)
- **Command Service**: Solo comandos (POST, PUT, DELETE)
- **Query Service**: Solo queries (GET)

### Event-Driven Architecture (EDA)

- **Comunicación asíncrona** mediante eventos
- **Desacoplamiento** entre servicios
- **Eventos persistentes** en Kafka

### Single Writer Principle

- **Solo Listener Service** escribe en SQLite
- **Evita conflictos** de escritura
- **Garantiza consistencia**

### Optimistic Locking

- **Control de concurrencia** mediante campo `version`
- **Sin bloqueos pesimistas**
- **Alta concurrencia** permitida

### Cache-First Strategy

- **Redis como cache** (opcional)
- **Fallback automático** si Redis no está disponible
- **Ultra baja latencia** en lecturas

---

## 🔮 Próximos Pasos y Mejoras Futuras

### Corto Plazo

1. **Implementar Redis real** (actualmente opcional)
2. **Métricas y observabilidad** (Prometheus, Grafana)
3. **Tests end-to-end** automatizados
4. **Documentación de despliegue** en producción

### Mediano Plazo

1. **Migración de base de datos** (PostgreSQL en lugar de SQLite)
2. **Replicación de Read Model** para alta disponibilidad
3. **Circuit breakers** para resiliencia
4. **Rate limiting** en APIs

### Largo Plazo

1. **Microservicios adicionales** (notificaciones, reportes)
2. **Event sourcing** completo
3. **Saga pattern** para transacciones distribuidas
4. **Multi-región** para alta disponibilidad global

---

## 📝 Conclusiones

La arquitectura propuesta resuelve los problemas críticos del sistema actual mediante:

1. **Separación de responsabilidades** (CQRS)
2. **Comunicación asíncrona** (Event-Driven)
3. **Optimización de lecturas** (Cache-First)
4. **Consistencia garantizada** (Single Writer + Optimistic Locking)
5. **Alta escalabilidad** (servicios independientes)

Los prototipos implementados demuestran la viabilidad de la solución y proporcionan una base sólida para la migración completa del sistema.

---

## 👥 Contacto y Soporte

Para preguntas o aclaraciones sobre este entregable, por favor contactar al equipo de desarrollo.

**Fecha de Entrega**: [Fecha]
**Versión**: 1.0
**Estado**: Prototipos Funcionales ✅

