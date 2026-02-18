# Guía de Pruebas Unitarias - Command Service

## 📋 Descripción

Este documento describe las pruebas unitarias implementadas para el Command Service, incluyendo casos de prueba, cobertura y cómo ejecutarlas.

## 🏗️ Estructura de Pruebas

```
command-service/
├── internal/
│   ├── domain/
│   │   ├── inventory.go
│   │   └── inventory_test.go          # Pruebas de lógica de dominio
│   ├── events/
│   │   ├── kafka_publisher.go
│   │   └── kafka_publisher_test.go     # Pruebas de publicación de eventos
│   ├── handlers/
│   │   ├── inventory_handler.go
│   │   └── inventory_handler_test.go   # Pruebas de handlers HTTP
│   └── repository/
│       └── inventory_repository.go     # Pruebas de repositorio (pendiente)
├── scripts/
│   └── run_tests.sh                    # Script para ejecutar todas las pruebas
└── test-results/
    └── README.md                       # Documentación de resultados
```

## 🧪 Casos de Prueba Implementados

### Domain (inventory.go)

#### ✅ Casos Exitosos
- **TestNewInventoryItem**: Crear un nuevo item de inventario
- **TestAvailableQuantity**: Calcular cantidad disponible
- **TestAdjustStock_Success_Increase**: Aumentar stock exitosamente
- **TestAdjustStock_Success_Decrease**: Disminuir stock exitosamente
- **TestReserveStock_Success**: Reservar stock exitosamente
- **TestReleaseStock_Success**: Liberar stock reservado exitosamente
- **TestFulfillReservation_Success**: Cumplir una reserva exitosamente

#### ❌ Casos de Error
- **TestAdjustStock_Error_NegativeResult**: Error al ajustar stock que resultaría en negativo
- **TestReserveStock_Error_InsufficientStock**: Error al reservar más stock del disponible
- **TestReleaseStock_Error_InvalidQuantity**: Error al liberar más stock del reservado
- **TestFulfillReservation_Error_InvalidQuantity**: Error al cumplir más stock del reservado

### Events (kafka_publisher.go)

#### ✅ Casos Exitosos
- **TestKafkaEventPublisher_Publish_InventoryItemCreatedEvent**: Publicar evento de creación
- **TestKafkaEventPublisher_Publish_StockAdjustedEvent**: Publicar evento de ajuste de stock
- **TestKafkaEventPublisher_GetEventType_AllTypes**: Mapeo de todos los tipos de eventos
- **TestKafkaEventPublisher_GetTopicForEvent_AllTypes**: Selección de topics según tipo de evento
- **TestKafkaEventPublisher_GetPartitionKey_UUID**: Generación de partition key con UUID
- **TestKafkaEventPublisher_GetPartitionKey_String**: Generación de partition key con string
- **TestInMemoryEventPublisher_Publish**: Publicación en memoria (fallback)

### Handlers (inventory_handler.go)

#### CreateItem
- ✅ **TestCreateItem_Success**: Crear item exitosamente
- ❌ **TestCreateItem_InvalidRequest_MissingFields**: Error por campos faltantes
- ❌ **TestCreateItem_InvalidRequest_NegativeQuantity**: Error por cantidad negativa
- ❌ **TestCreateItem_RepositoryError**: Error del repositorio

#### UpdateItem
- ✅ **TestUpdateItem_Success**: Actualizar item exitosamente
- ❌ **TestUpdateItem_NotFound**: Error cuando el item no existe

#### DeleteItem
- ✅ **TestDeleteItem_Success**: Eliminar item exitosamente
- ❌ **TestDeleteItem_NotFound**: Error cuando el item no existe

#### AdjustStock
- ✅ **TestAdjustStock_Success**: Ajustar stock exitosamente
- ❌ **TestAdjustStock_InsufficientStock**: Error por stock insuficiente

#### ReserveStock
- ✅ **TestReserveStock_Success**: Reservar stock exitosamente
- ❌ **TestReserveStock_InsufficientStock**: Error por stock insuficiente

#### ReleaseStock
- ✅ **TestReleaseStock_Success**: Liberar stock exitosamente
- ❌ **TestReleaseStock_InvalidQuantity**: Error por cantidad inválida

## 🚀 Ejecutar Pruebas

### Ejecutar todas las pruebas

```bash
cd command-service
go test ./internal/... -v
```

### Ejecutar pruebas de un paquete específico

```bash
# Pruebas de domain
go test ./internal/domain -v

# Pruebas de events
go test ./internal/events -v

# Pruebas de handlers
go test ./internal/handlers -v
```

### Ejecutar pruebas con cobertura

```bash
# Cobertura de un paquete
go test ./internal/domain -coverprofile=coverage.out
go tool cover -html=coverage.out

# Cobertura total
go test ./internal/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Ejecutar script de pruebas completo

```bash
cd command-service
./scripts/run_tests.sh
```

Este script:
- Ejecuta todas las pruebas por paquete
- Genera reportes de cobertura en HTML
- Guarda los resultados en `test-results/YYYYMMDD_HHMMSS/`
- Genera un resumen con estadísticas

## 📊 Cobertura de Código

El objetivo es mantener una cobertura de código superior al 80% para todos los paquetes.

Para ver el reporte de cobertura HTML:

```bash
open test-results/YYYYMMDD_HHMMSS/coverage/total.html
```

## 🔧 Mocks Utilizados

### MockInventoryRepository
Mock del repositorio de inventario para aislar las pruebas de handlers.

### MockEventPublisher
Mock del publicador de eventos para aislar las pruebas de handlers.

## 📝 Notas Importantes

1. **Independencia**: Las pruebas están diseñadas para ser independientes y ejecutables en cualquier orden.

2. **Mocks**: Se utilizan mocks para aislar las dependencias (Kafka, Repository) y hacer las pruebas más rápidas y confiables.

3. **Assertions**: Se utiliza `testify` para assertions y mocks, proporcionando mensajes de error claros.

4. **Cobertura**: Los reportes de cobertura se generan automáticamente con cada ejecución del script.

5. **Resultados**: Los resultados se guardan en `test-results/` con un timestamp para mantener un historial.

## 🎯 Próximos Pasos

- [ ] Agregar pruebas de integración
- [ ] Agregar pruebas de rendimiento
- [ ] Agregar pruebas de carga
- [ ] Mejorar cobertura de casos edge
- [ ] Agregar pruebas de repository (cuando se implemente PostgreSQL)

## 📚 Referencias

- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Go Coverage Tool](https://go.dev/blog/cover)

