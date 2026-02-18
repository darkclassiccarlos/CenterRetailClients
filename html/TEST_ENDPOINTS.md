# 🧪 Script de Pruebas de Endpoints - Dashboard HTML

Este documento describe el script de pruebas bash que prueba todos los casos de uso de los endpoints disponibles en el Dashboard HTML.

## 📋 Descripción

El script `test-endpoints.sh` prueba todos los endpoints disponibles en el Dashboard HTML, simulando las interacciones que un usuario realizaría desde la interfaz web.

## 🚀 Uso

### Ejecutar todas las pruebas:

```bash
cd html
./test-endpoints.sh
```

### Requisitos previos:

1. **Servicios corriendo:**
   - Command Service (puerto 8080)
   - Query Service (puerto 8081)
   - Dashboard Server/Proxy (puerto 8000) - opcional

2. **Herramientas instaladas:**
   - `curl` - Para hacer peticiones HTTP
   - `jq` - Para parsear JSON
     - macOS: `brew install jq`
     - Linux: `apt-get install jq` o `yum install jq`

## 📊 Casos de Uso Probados

El script prueba los siguientes casos de uso en orden:

### 1. ✅ Autenticación
- **Endpoint**: `POST /api/v1/auth/login`
- **Descripción**: Autenticarse con el sistema
- **Body**: `{"username":"admin","password":"admin123"}`
- **Resultado esperado**: Token JWT

### 2. ✅ Listar Items
- **Endpoint**: `GET /api/v1/inventory/items?page=1&page_size=100`
- **Descripción**: Listar items de inventario con paginación
- **Resultado esperado**: Lista de items con paginación

### 3. ✅ Crear Item
- **Endpoint**: `POST /api/v1/inventory/items`
- **Descripción**: Crear un nuevo item de inventario
- **Body**: `{"sku":"TEST-SKU-...","name":"Item de Prueba","description":"...","quantity":100}`
- **Resultado esperado**: Item creado con ID

### 4. ✅ Buscar Item por SKU
- **Endpoint**: `GET /api/v1/inventory/items/sku/:sku`
- **Descripción**: Buscar un item por su SKU
- **Resultado esperado**: Item encontrado

### 5. ✅ Obtener Item por ID
- **Endpoint**: `GET /api/v1/inventory/items/:id`
- **Descripción**: Obtener un item por su ID
- **Resultado esperado**: Item encontrado

### 6. ✅ Actualizar Item
- **Endpoint**: `PUT /api/v1/inventory/items/:id`
- **Descripción**: Actualizar nombre y descripción de un item
- **Body**: `{"name":"Item Actualizado","description":"..."}`
- **Resultado esperado**: Item actualizado

### 7. ✅ Reservar Stock
- **Endpoint**: `POST /api/v1/inventory/items/:id/reserve`
- **Descripción**: Reservar 5 unidades de stock
- **Body**: `{"quantity":5}`
- **Resultado esperado**: Stock reservado exitosamente

### 8. ✅ Liberar Stock
- **Endpoint**: `POST /api/v1/inventory/items/:id/release`
- **Descripción**: Liberar 2 unidades de stock reservado
- **Body**: `{"quantity":2}`
- **Resultado esperado**: Stock liberado exitosamente

### 9. ✅ Ajustar Stock
- **Endpoint**: `POST /api/v1/inventory/items/:id/adjust`
- **Descripción**: Aumentar stock en 10 unidades
- **Body**: `{"quantity":10}`
- **Resultado esperado**: Stock ajustado exitosamente

### 10. ✅ Eliminar Item
- **Endpoint**: `DELETE /api/v1/inventory/items/:id`
- **Descripción**: Eliminar el item creado durante las pruebas
- **Resultado esperado**: Item eliminado exitosamente

## 🔧 Configuración

El script está configurado para usar el proxy por defecto. Puedes modificar las siguientes variables en el script:

```bash
PROXY_URL="http://localhost:8000"
COMMAND_SERVICE_URL="http://localhost:8080"
QUERY_SERVICE_URL="http://localhost:8081"
USE_PROXY=true
```

### Usar servicios directamente (sin proxy):

Edita el script y cambia:
```bash
USE_PROXY=false
```

## 📝 Salida del Script

El script muestra:
- ✅ **Verde**: Tests que pasaron exitosamente
- ❌ **Rojo**: Tests que fallaron
- ℹ️ **Azul**: Información sobre las peticiones
- 🟡 **Amarillo**: Secciones y títulos

### Ejemplo de salida:

```
=======================================================
🧪 Pruebas de Casos de Uso - Dashboard HTML
Sistema de Gestión de Inventario (CQRS + EDA)
=======================================================

=======================================================
Verificando Servicios
=======================================================

ℹ️  Verificando Command Service (8080)...
✅ Command Service está corriendo
ℹ️  Verificando Query Service (8081)...
✅ Query Service está corriendo
ℹ️  Verificando Proxy Server (8000)...
✅ Proxy Server está corriendo

=======================================================
Test 1: Autenticación (POST /api/v1/auth/login)
=======================================================

ℹ️  Probando: Autenticación con Command Service
ℹ️  URL: http://localhost:8000/api/v1/auth/login
ℹ️  Método: POST
ℹ️  Body: {"username":"admin","password":"admin123"}
✅ HTTP 200 - Autenticación con Command Service
✅ Token obtenido: eyJhbGciOiJIUzI1NiIs...

...

=======================================================
Resumen de Pruebas
=======================================================

Total de tests: 10
✅ Tests pasados: 10
✅ Tests fallidos: 0

✅ ¡Todos los tests pasaron exitosamente! 🎉
```

## 🔍 Verificación de Servicios

El script verifica automáticamente que los servicios estén corriendo antes de ejecutar las pruebas:

1. **Command Service** (puerto 8080)
   - Verifica: `GET /api/v1/health`

2. **Query Service** (puerto 8081)
   - Verifica: `GET /api/v1/health`

3. **Proxy Server** (puerto 8000) - si `USE_PROXY=true`
   - Verifica: `GET /index.html`

Si algún servicio no está disponible, el script se detiene con un error.

## 🛠️ Funcionalidades del Script

### 1. **Autenticación Automática**
- El script se autentica automáticamente al inicio
- El token se guarda y se usa en todas las peticiones posteriores

### 2. **Gestión de Estado**
- El script crea un item de prueba y guarda su ID
- Usa el ID para las pruebas que requieren un item existente
- Elimina el item al final para limpiar

### 3. **Validación de Respuestas**
- Verifica códigos de estado HTTP (200-299 = éxito)
- Parsea y muestra respuestas JSON
- Extrae IDs y tokens de las respuestas

### 4. **Manejo de Errores**
- Si la autenticación falla, el script se detiene
- Si un test falla, continúa con los siguientes
- Muestra un resumen al final con tests pasados/fallidos

## 📊 Orden de Ejecución

El script ejecuta los tests en el siguiente orden:

1. **Autenticación** (requerido)
2. **Listar Items** (no requiere item creado)
3. **Crear Item** (crea el item de prueba)
4. **Buscar por SKU** (usa el item creado)
5. **Obtener por ID** (usa el item creado)
6. **Actualizar Item** (usa el item creado)
7. **Reservar Stock** (usa el item creado)
8. **Liberar Stock** (usa el item creado)
9. **Ajustar Stock** (usa el item creado)
10. **Eliminar Item** (limpia el item creado)

## 🐛 Solución de Problemas

### Error: "Command Service no está disponible"
- Verifica que el Command Service esté corriendo en el puerto 8080
- Ejecuta: `curl http://localhost:8080/api/v1/health`

### Error: "Query Service no está disponible"
- Verifica que el Query Service esté corriendo en el puerto 8081
- Ejecuta: `curl http://localhost:8081/api/v1/health`

### Error: "Proxy Server no está disponible"
- Verifica que el Dashboard Server esté corriendo en el puerto 8000
- Ejecuta: `curl http://localhost:8000/index.html`
- O cambia `USE_PROXY=false` para usar los servicios directamente

### Error: "jq no está instalado"
- macOS: `brew install jq`
- Linux: `apt-get install jq` o `yum install jq`

### Error: "La autenticación falló"
- Verifica que las credenciales sean correctas (admin/admin123)
- Verifica que el Command Service esté corriendo
- Verifica que el proxy esté configurado correctamente

## 📝 Notas

- El script crea un item de prueba con un SKU único basado en timestamp
- El item se elimina al final para mantener la base de datos limpia
- Si el script se interrumpe, el item de prueba puede quedar en la base de datos
- Puedes ejecutar el script múltiples veces sin problemas

## 🔄 Integración con CI/CD

El script puede integrarse en pipelines de CI/CD:

```yaml
# Ejemplo para GitHub Actions
- name: Run Endpoint Tests
  run: |
    cd html
    ./test-endpoints.sh
```

## 📚 Referencias

- [Documentación de Tests Unitarios](./TESTING.md)
- [README del Dashboard](./README.md)
- [Guía de Ejecución del Proyecto](../run.md)

---

**Última actualización**: 2025  
**Versión**: 1.0

