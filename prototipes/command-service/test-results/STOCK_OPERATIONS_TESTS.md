# Pruebas de Operaciones de Stock - Análisis y Correcciones

## 🐛 Problema Identificado

### Bug en Listener Service - `processStockAdjusted`

**Problema:** El método `processStockAdjusted` en el Listener Service estaba calculando incorrectamente el ajuste de stock.

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

## ✅ Correcciones Implementadas

### 1. Corrección en Listener Service

**Archivo:** `listener-service/internal/events/event_processor.go`

**Cambio:**
- Corregido el cálculo del ajuste en `processStockAdjusted`
- Agregados comentarios explicativos
- El ajuste ahora se toma directamente de `event.Quantity`

### 2. Pruebas de Integración Creadas

**Archivo:** `command-service/internal/handlers/inventory_handler_integration_test.go`

**Pruebas Agregadas:**
- `TestAdjustStock_Integration_VerifiesEventData`: Verifica que AdjustStock publique el evento correcto
- `TestAdjustStock_Negative_VerifiesEventData`: Verifica ajuste negativo
- `TestReserveStock_Integration_VerifiesEventData`: Verifica que ReserveStock publique el evento correcto
- `TestReleaseStock_Integration_VerifiesEventData`: Verifica que ReleaseStock publique el evento correcto
- `TestStockOperations_Sequence_VerifiesMultipleEvents`: Verifica una secuencia completa de operaciones
- `TestAdjustStock_EventPublishingFailure_StillSaves`: Verifica que el item se guarde incluso si falla la publicación del evento

### 3. Script de Pruebas End-to-End

**Archivo:** `command-service/scripts/test_stock_operations.sh`

**Funcionalidades:**
- Verifica servicios disponibles
- Crea item de prueba
- Prueba ajustes de stock (aumentar y disminuir)
- Prueba reservas de stock
- Prueba liberaciones de stock
- Verifica valores finales y consistencia

## 📊 Cobertura de Pruebas Mejorada

### Antes
- **Handlers**: 53.7% ⚠️
- **Events**: 28.0% ⚠️
- **AdjustStock**: 48.1% ⚠️
- **ReserveStock**: 48.1% ⚠️
- **ReleaseStock**: 48.1% ⚠️

### Después (Esperado)
- **Handlers**: ~70%+ ✅
- **Events**: ~50%+ ✅
- **AdjustStock**: ~80%+ ✅
- **ReserveStock**: ~80%+ ✅
- **ReleaseStock**: ~80%+ ✅

## 🧪 Casos de Prueba Agregados

### AdjustStock

#### ✅ TestAdjustStock_Integration_VerifiesEventData
- **Objetivo**: Verificar que AdjustStock publique el evento correcto con los datos correctos
- **Verifica**:
  - Evento `StockAdjustedEvent` publicado
  - `Quantity` = ajuste (diferencia)
  - `NewTotal` = nueva cantidad total
  - `ItemID` y `SKU` correctos

#### ✅ TestAdjustStock_Negative_VerifiesEventData
- **Objetivo**: Verificar ajuste negativo (disminuir stock)
- **Verifica**:
  - Ajuste negativo correcto
  - Nueva cantidad total correcta
  - Evento con valores negativos

### ReserveStock

#### ✅ TestReserveStock_Integration_VerifiesEventData
- **Objetivo**: Verificar que ReserveStock publique el evento correcto
- **Verifica**:
  - Evento `StockReservedEvent` publicado
  - `Quantity` = cantidad reservada en esta operación
  - `Reserved` = total reservado después
  - `Available` = disponible después

### ReleaseStock

#### ✅ TestReleaseStock_Integration_VerifiesEventData
- **Objetivo**: Verificar que ReleaseStock publique el evento correcto
- **Verifica**:
  - Evento `StockReleasedEvent` publicado
  - `Quantity` = cantidad liberada en esta operación
  - `Reserved` = total reservado después
  - `Available` = disponible después

### Secuencia Completa

#### ✅ TestStockOperations_Sequence_VerifiesMultipleEvents
- **Objetivo**: Verificar una secuencia completa de operaciones
- **Flujo**:
  1. Ajustar stock (aumentar +50)
  2. Reservar stock (30 unidades)
  3. Liberar stock (10 unidades)
- **Verifica**:
  - Todos los eventos se publican correctamente
  - Los valores son consistentes en cada paso
  - El estado final es correcto

## 🚀 Ejecutar Pruebas

### Pruebas de Integración

```bash
cd command-service
go test ./internal/handlers -run TestAdjustStock_Integration -v
go test ./internal/handlers -run TestReserveStock_Integration -v
go test ./internal/handlers -run TestReleaseStock_Integration -v
go test ./internal/handlers -run TestStockOperations_Sequence -v
```

### Pruebas End-to-End

```bash
cd command-service
./scripts/test_stock_operations.sh
```

## 📝 Verificación del Flujo Completo

### Flujo Esperado

1. **Command Service** recibe request de ajuste/reserva/liberación
2. **Command Service** actualiza el item en memoria
3. **Command Service** publica evento a Kafka con:
   - `Quantity`: Ajuste (diferencia) o cantidad a reservar/liberar
   - `NewTotal`: Nueva cantidad total (solo para AdjustStock)
   - `Reserved`: Total reservado (solo para Reserve/Release)
   - `Available`: Disponible después (solo para Reserve/Release)
4. **Listener Service** consume el evento
5. **Listener Service** procesa el evento y actualiza SQLite
6. **Query Service** invalida cache y actualiza Read Model

### Verificación

Para verificar que el flujo completo funciona:

1. **Ejecutar servicios:**
   ```bash
   # Terminal 1: Command Service
   cd command-service && go run cmd/api/main.go
   
   # Terminal 2: Listener Service
   cd listener-service && go run cmd/listener/main.go
   
   # Terminal 3: Query Service
   cd query-service && go run cmd/api/main.go
   ```

2. **Ejecutar script de pruebas:**
   ```bash
   cd command-service
   ./scripts/test_stock_operations.sh
   ```

3. **Verificar en Listener Service:**
   ```bash
   # Verificar estadísticas
   curl http://localhost:8082/api/v1/monitoring/stats
   
   # Verificar base de datos
   sqlite3 listener-service/inventory.db "SELECT * FROM inventory_items;"
   ```

## ✅ Resultados Esperados

Después de las correcciones:

- ✅ **AdjustStock**: Los valores se actualizan correctamente
- ✅ **ReserveStock**: Las reservas se registran correctamente
- ✅ **ReleaseStock**: Las liberaciones se procesan correctamente
- ✅ **Eventos**: Se publican con los datos correctos
- ✅ **Listener Service**: Procesa los eventos correctamente
- ✅ **Consistencia**: Los valores son consistentes en todo el flujo

## 🔍 Debugging

Si los valores no se actualizan:

1. **Verificar logs del Command Service:**
   - Buscar "Event published to Kafka"
   - Verificar que el evento tenga los valores correctos

2. **Verificar logs del Listener Service:**
   - Buscar "Stock adjusted", "Stock reserved", "Stock released"
   - Verificar que el ajuste sea correcto

3. **Verificar Kafka:**
   - Usar Kafdrop (http://localhost:9000)
   - Verificar que los mensajes lleguen a los topics

4. **Verificar base de datos:**
   ```bash
   sqlite3 listener-service/inventory.db "SELECT id, sku, quantity, reserved, available, version FROM inventory_items;"
   ```

