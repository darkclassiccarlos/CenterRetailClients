# Pruebas de Actualización de Redis - Query Service

## 📋 Descripción

Este script prueba que Redis se actualiza correctamente en el Query Service después de que el Listener Service actualiza la base de datos.

## 🎯 Objetivo

Verificar que:
1. Los eventos de confirmación del Listener Service se publiquen correctamente
2. El Query Service consuma estos eventos y actualice Redis con datos nuevos
3. El Query Service siempre responda desde Redis (no bloquea actualizaciones)
4. Las actualizaciones sean asíncronas y no afecten las respuestas

## 🔄 Flujo Implementado

```
Command Service
    ↓ Publica evento a Kafka
Kafka Topic (inventory.items, inventory.stock)
    ↓ Listener Service consume evento
Listener Service
    ↓ Procesa evento y actualiza BD
SQLite Database (actualizada)
    ↓ Publica evento de confirmación a Kafka
Kafka Topic (mismo topic, evento con sufijo "Confirmed")
    ↓ Query Service consume evento de confirmación
Query Service
    ↓ Actualiza Redis con datos nuevos
Redis Cache (actualizado)
    ↓ Query Service responde desde Redis
Cliente (respuesta rápida desde Redis)
```

## 🧪 Pruebas Incluidas

### Test 1: Crear Item y Verificar Actualización en Redis
- Crea un item en Command Service
- Espera procesamiento por Listener Service
- Verifica que el item esté disponible en Query Service (desde Redis)

### Test 2: Ajustar Stock y Verificar Actualización en Redis
- Crea un item
- Ajusta el stock (aumenta cantidad)
- Verifica que el stock actualizado esté en Redis

### Test 3: Reservar Stock y Verificar Actualización en Redis
- Crea un item
- Reserva stock
- Verifica que la reserva se refleje en Redis (reserved y available actualizados)

### Test 4: Liberar Stock y Verificar Actualización en Redis
- Crea un item
- Reserva stock
- Libera stock parcial
- Verifica que la liberación se refleje en Redis

### Test 5: Query desde Redis (No Bloquea Actualizaciones)
- Crea un item
- Realiza múltiples consultas rápidas (deben responder desde Redis)
- Mientras tanto, ajusta stock (actualización asíncrona)
- Verifica que la actualización asíncrona se complete sin bloquear consultas

## 🚀 Ejecución

```bash
cd query-service
./scripts/test_redis_update.sh
```

## 📋 Prerequisitos

- Command Service corriendo en `http://localhost:8080`
- Query Service corriendo en `http://localhost:8081`
- Listener Service corriendo (puede estar solo el listener, no el API)
- Kafka corriendo en `localhost:9093`
- Redis corriendo en `localhost:6379` (opcional pero recomendado)

## ✅ Resultados Esperados

- **Test 1**: Item disponible en Redis después de creación
- **Test 2**: Stock actualizado en Redis después de ajuste
- **Test 3**: Reserva reflejada en Redis (reserved y available correctos)
- **Test 4**: Liberación reflejada en Redis (reserved y available correctos)
- **Test 5**: Consultas responden desde Redis sin bloquear actualizaciones asíncronas

## 🔍 Verificación Manual

### Verificar en Redis

```bash
# Conectar a Redis
redis-cli

# Ver todas las claves de items
KEYS item:*

# Ver item específico
GET item:id:<item-id>

# Ver stock específico
GET stock:<item-id>
```

### Verificar Logs

**Listener Service:**
- Buscar "Confirmation event published"
- Verificar que los eventos se publiquen después de actualizar BD

**Query Service:**
- Buscar "Updating Redis cache with new data"
- Verificar que Redis se actualice cuando se reciben eventos de confirmación

## 📝 Notas Importantes

1. **Eventos de Confirmación**: Los eventos de confirmación tienen el sufijo "Confirmed" (ej: `InventoryItemCreatedConfirmed`)

2. **Actualización Asíncrona**: Las actualizaciones de Redis son asíncronas y no bloquean las respuestas del Query Service

3. **Fallback**: Si no se puede leer desde el repository, se usa la data del evento para actualizar Redis

4. **Invalidación de Listas**: Las listas se invalidan para asegurar datos frescos en próximas consultas

5. **TTL**: Los datos en Redis tienen un TTL configurable (default: 5 minutos)

