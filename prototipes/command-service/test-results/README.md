# Resultados de Pruebas Unitarias

Este directorio contiene los resultados de las pruebas unitarias ejecutadas para el Command Service.

## Estructura

```
test-results/
├── README.md                    # Este archivo
├── YYYYMMDD_HHMMSS/             # Directorio con timestamp de ejecución
│   ├── summary.txt              # Resumen de la ejecución
│   ├── coverage/                # Reportes de cobertura
│   │   ├── total.html           # Reporte HTML de cobertura total
│   │   ├── total.out            # Datos de cobertura total
│   │   ├── domain.out           # Cobertura del paquete domain
│   │   ├── domain.html          # Reporte HTML de domain
│   │   ├── events.out           # Cobertura del paquete events
│   │   ├── events.html          # Reporte HTML de events
│   │   ├── handlers.out         # Cobertura del paquete handlers
│   │   ├── handlers.html        # Reporte HTML de handlers
│   │   ├── repository.out       # Cobertura del paquete repository
│   │   └── repository.html      # Reporte HTML de repository
│   ├── domain/                  # Resultados de pruebas de domain
│   │   └── test_output.txt      # Output de las pruebas
│   ├── events/                  # Resultados de pruebas de events
│   │   └── test_output.txt      # Output de las pruebas
│   ├── handlers/                # Resultados de pruebas de handlers
│   │   └── test_output.txt      # Output de las pruebas
│   └── repository/              # Resultados de pruebas de repository
│       └── test_output.txt      # Output de las pruebas
```

## Ejecutar Pruebas

Para ejecutar todas las pruebas y generar los reportes:

```bash
./scripts/run_tests.sh
```

O desde el directorio raíz del proyecto:

```bash
cd command-service
./scripts/run_tests.sh
```

## Casos de Prueba Cubiertos

### Domain (inventory.go)

- ✅ Creación de items de inventario
- ✅ Cálculo de cantidad disponible
- ✅ Ajuste de stock (aumentar/disminuir)
- ✅ Reserva de stock
- ✅ Liberación de stock
- ✅ Validaciones de negocio (stock insuficiente, cantidad inválida)

### Events (kafka_publisher.go)

- ✅ Mapeo de tipos de eventos
- ✅ Selección de topics según tipo de evento
- ✅ Generación de partition keys
- ✅ Publicación de eventos (InventoryItemCreated, InventoryItemUpdated, InventoryItemDeleted)
- ✅ Publicación de eventos de stock (StockAdjusted, StockReserved, StockReleased)

### Handlers (inventory_handler.go)

#### CreateItem
- ✅ Caso exitoso: Crear item con todos los campos
- ✅ Error: Campos requeridos faltantes
- ✅ Error: Cantidad negativa
- ✅ Error: Error de repositorio

#### UpdateItem
- ✅ Caso exitoso: Actualizar nombre y descripción
- ✅ Error: Item no encontrado
- ✅ Error: ID inválido

#### DeleteItem
- ✅ Caso exitoso: Eliminar item existente
- ✅ Error: Item no encontrado
- ✅ Error: ID inválido

#### AdjustStock
- ✅ Caso exitoso: Aumentar stock
- ✅ Caso exitoso: Disminuir stock
- ✅ Error: Stock insuficiente (resultado negativo)
- ✅ Error: Item no encontrado

#### ReserveStock
- ✅ Caso exitoso: Reservar stock disponible
- ✅ Error: Stock insuficiente
- ✅ Error: Item no encontrado

#### ReleaseStock
- ✅ Caso exitoso: Liberar stock reservado
- ✅ Error: Cantidad a liberar excede lo reservado
- ✅ Error: Item no encontrado

### Repository (inventory_repository.go)

- ✅ Guardar items
- ✅ Buscar por ID
- ✅ Buscar por SKU
- ✅ Eliminar items
- ✅ Manejo de errores (item no encontrado)

## Cobertura de Código

El objetivo es mantener una cobertura de código superior al 80% para todos los paquetes.

Para ver el reporte de cobertura HTML:

```bash
open test-results/YYYYMMDD_HHMMSS/coverage/total.html
```

## Notas

- Las pruebas utilizan mocks para aislar las dependencias (Kafka, Repository)
- Los tests están diseñados para ser independientes y ejecutables en cualquier orden
- Se utiliza `testify` para assertions y mocks
- Los reportes se generan automáticamente con cada ejecución

