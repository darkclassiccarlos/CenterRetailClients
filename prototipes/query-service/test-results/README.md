# Pruebas Unitarias y de Integración - Query Service

## Resumen

Este directorio contiene scripts y resultados de pruebas unitarias y de integración para el Query Service.

## Estructura

```
test-results/
├── run_tests.sh          # Script para ejecutar pruebas unitarias
├── test_integration.sh   # Script para pruebas de integración end-to-end
├── README.md             # Este archivo
└── [TIMESTAMP]/          # Resultados de ejecuciones anteriores
    ├── coverage/         # Reportes de cobertura
    ├── handlers/         # Resultados de pruebas de handlers
    ├── auth/             # Resultados de pruebas de autenticación
    └── middleware/        # Resultados de pruebas de middleware
```

## Pruebas Unitarias

### Ejecutar Pruebas Unitarias

```bash
cd query-service
./test-results/run_tests.sh
```

Este script:
- Ejecuta pruebas unitarias para todos los paquetes
- Genera reportes de cobertura
- Crea reportes HTML de cobertura
- Guarda resultados en `test-results/[TIMESTAMP]/`

### Cobertura de Pruebas

Las pruebas unitarias cubren:

1. **Handlers** (`internal/handlers`)
   - ✅ ListItems (cache hit/miss, paginación)
   - ✅ GetItemByID (cache hit/miss, not found, invalid UUID)
   - ✅ GetItemBySKU (cache hit/miss)
   - ✅ GetStockStatus (cache hit/miss)
   - ✅ Validación de paginación

2. **Autenticación** (`internal/auth`)
   - ✅ Login exitoso
   - ✅ Login con credenciales inválidas
   - ✅ Generación y validación de tokens JWT
   - ✅ Tokens inválidos y expirados

3. **Middleware** (`pkg/middleware`)
   - ✅ Validación de tokens válidos
   - ✅ Rechazo de tokens faltantes
   - ✅ Rechazo de tokens inválidos
   - ✅ Establecimiento de valores en contexto

### Mocks Utilizados

- **MockCache**: Mock de la interfaz `cache.Cache` para probar comportamiento de cache
- **MockRepository**: Mock de la interfaz `repository.ReadRepository` para probar handlers sin base de datos real

## Pruebas de Integración

### Ejecutar Pruebas de Integración

```bash
cd query-service
./test-results/test_integration.sh
```

**Prerequisitos:**
- Command Service corriendo en `http://localhost:8080`
- Query Service corriendo en `http://localhost:8081`
- Listener Service corriendo en `http://localhost:8082`
- Kafka corriendo en `localhost:9093`
- Redis corriendo en `localhost:6379` (opcional, pero recomendado)

### Pruebas de Integración Incluidas

1. **Servicios Disponibles**
   - Verifica que todos los servicios estén corriendo

2. **Autenticación JWT**
   - Obtiene token JWT del endpoint de login
   - Verifica que el token sea válido

3. **Endpoint Protegido Sin Token**
   - Verifica que endpoints protegidos rechacen peticiones sin token (HTTP 401)

4. **Endpoint Protegido Con Token**
   - Verifica que endpoints protegidos acepten peticiones con token válido (HTTP 200)

5. **Crear y Consultar Item**
   - Crea un item en Command Service
   - Espera procesamiento de eventos
   - Consulta el item en Query Service
   - Verifica eventual consistency

6. **Cache de Redis**
   - Realiza dos consultas consecutivas
   - Verifica que el cache funcione correctamente

## Verificación de Actualizaciones en Memoria

Las pruebas de integración verifican:

1. **Actualizaciones en Redis**: 
   - Primera consulta (cache miss) → consulta base de datos
   - Segunda consulta (cache hit) → retorna desde cache
   - Esto verifica que Redis está actualizando correctamente

2. **Sincronización de Datos**:
   - Item creado en Command Service
   - Evento publicado a Kafka
   - Listener Service procesa evento
   - Query Service puede consultar el item
   - Verifica que los datos se sincronicen correctamente

3. **Respuestas de Endpoints**:
   - Verifica que los endpoints retornen datos correctos
   - Verifica formato JSON válido
   - Verifica códigos HTTP correctos

## Resultados

### Ver Reportes de Cobertura

```bash
# Abrir reporte HTML de cobertura
open test-results/[TIMESTAMP]/coverage/total.html
```

### Ver Resumen de Pruebas

```bash
# Ver resumen de pruebas
cat test-results/[TIMESTAMP]/summary.txt
```

### Ver Output de Pruebas

```bash
# Ver output de pruebas de handlers
cat test-results/[TIMESTAMP]/handlers/test_output.txt

# Ver output de pruebas de autenticación
cat test-results/[TIMESTAMP]/auth/test_output.txt

# Ver output de pruebas de middleware
cat test-results/[TIMESTAMP]/middleware/test_output.txt
```

## Cobertura Esperada

- **Handlers**: > 65% (objetivo: 70%+)
- **Auth**: > 80% (objetivo: 85%+)
- **Middleware**: > 55% (objetivo: 60%+)
- **Total**: > 60% (objetivo: 65%+)

## Notas

- Las pruebas unitarias usan mocks para aislar componentes
- Las pruebas de integración requieren servicios corriendo
- Las pruebas de integración verifican el flujo completo end-to-end
- Los resultados se guardan con timestamp para mantener historial

