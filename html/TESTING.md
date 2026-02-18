# 🧪 Tests Unitarios: Dashboard Server (Proxy)

Este documento describe los tests unitarios implementados para el servidor proxy del Dashboard HTML.

## 📋 Casos de Uso Testeados

Los tests cubren todos los casos de uso principales expuestos desde el servicio HTML:

### 1. **Consultas (GET) - Query Service (8081)**

#### ✅ `TestProxyRouting_GET_InventoryItems`
- **Caso de uso**: Listar items de inventario con paginación
- **Ruta**: `GET /api/v1/inventory/items?page=1&page_size=100`
- **Verifica**: 
  - La petición se redirige al Query Service
  - La ruta se preserva correctamente
  - Los query params se preservan

#### ✅ `TestProxyRouting_GET_InventoryItemsByID`
- **Caso de uso**: Obtener un item por su ID
- **Ruta**: `GET /api/v1/inventory/items/:id`
- **Verifica**: 
  - La petición se redirige al Query Service
  - La ruta con parámetros dinámicos se preserva correctamente

#### ✅ `TestProxyRouting_GET_InventoryItemsBySKU`
- **Caso de uso**: Buscar un item por SKU
- **Ruta**: `GET /api/v1/inventory/items/sku/:sku`
- **Verifica**: 
  - La petición se redirige al Query Service
  - La ruta con parámetros dinámicos se preserva correctamente

### 2. **Comandos (POST) - Command Service (8080)**

#### ✅ `TestProxyRouting_POST_CreateItem`
- **Caso de uso**: Crear un nuevo item de inventario
- **Ruta**: `POST /api/v1/inventory/items`
- **Verifica**: 
  - La petición se redirige al Command Service
  - El body de la petición se preserva correctamente
  - El método HTTP se preserva

#### ✅ `TestProxyRouting_POST_ReserveStock`
- **Caso de uso**: Reservar stock de un item
- **Ruta**: `POST /api/v1/inventory/items/:id/reserve`
- **Verifica**: 
  - La petición se redirige al Command Service
  - La ruta completa con parámetros dinámicos se preserva correctamente
  - El body de la petición se preserva

#### ✅ `TestProxyRouting_POST_ReleaseStock`
- **Caso de uso**: Liberar stock reservado de un item
- **Ruta**: `POST /api/v1/inventory/items/:id/release`
- **Verifica**: 
  - La petición se redirige al Command Service
  - La ruta completa con parámetros dinámicos se preserva correctamente

#### ✅ `TestProxyRouting_POST_AdjustStock`
- **Caso de uso**: Ajustar stock de un item
- **Ruta**: `POST /api/v1/inventory/items/:id/adjust`
- **Verifica**: 
  - La petición se redirige al Command Service
  - La ruta completa con parámetros dinámicos se preserva correctamente

### 3. **Actualizaciones (PUT) - Command Service (8080)**

#### ✅ `TestProxyRouting_PUT_UpdateItem`
- **Caso de uso**: Actualizar un item de inventario
- **Ruta**: `PUT /api/v1/inventory/items/:id`
- **Verifica**: 
  - La petición se redirige al Command Service
  - La ruta con parámetros dinámicos se preserva correctamente
  - El método HTTP se preserva

### 4. **Eliminaciones (DELETE) - Command Service (8080)**

#### ✅ `TestProxyRouting_DELETE_DeleteItem`
- **Caso de uso**: Eliminar un item de inventario
- **Ruta**: `DELETE /api/v1/inventory/items/:id`
- **Verifica**: 
  - La petición se redirige al Command Service
  - La ruta con parámetros dinámicos se preserva correctamente
  - El método HTTP se preserva

### 5. **Autenticación (POST) - Command Service (8080)**

#### ✅ `TestProxyRouting_POST_Login`
- **Caso de uso**: Autenticarse con el sistema
- **Ruta**: `POST /api/v1/auth/login`
- **Verifica**: 
  - La petición se redirige al Command Service
  - El body de la petición se preserva correctamente

### 6. **Preservación de Query Params**

#### ✅ `TestProxyRouting_QueryParamsPreservation`
- **Caso de uso**: Verificar que los query params se preserven en las peticiones
- **Ruta**: `GET /api/v1/inventory/items?page=1&page_size=100`
- **Verifica**: 
  - Los query params se preservan correctamente en el proxy
  - Se pasan correctamente al servicio backend

### 7. **Headers CORS**

#### ✅ `TestCORSHeaders`
- **Caso de uso**: Verificar que los headers CORS se agreguen correctamente
- **Verifica**: 
  - `Access-Control-Allow-Origin: *`
  - `Access-Control-Allow-Methods` está presente
  - Los headers se agregan a todas las respuestas

#### ✅ `TestCORS_PreflightRequest`
- **Caso de uso**: Verificar que las peticiones OPTIONS (preflight) se manejen correctamente
- **Ruta**: `OPTIONS /api/v1/inventory/items`
- **Verifica**: 
  - Las peticiones OPTIONS retornan 200 OK
  - Los headers CORS se incluyen en la respuesta

## 🚀 Ejecutar los Tests

### Ejecutar todos los tests:
```bash
cd html
go test -v
```

### Ejecutar un test específico:
```bash
cd html
go test -v -run TestProxyRouting_GET_InventoryItems
```

### Ejecutar tests con cobertura:
```bash
cd html
go test -v -cover
```

### Ejecutar tests con cobertura detallada:
```bash
cd html
go test -v -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## 📊 Resultados Esperados

Todos los tests deberían pasar exitosamente:

```
=== RUN   TestProxyRouting_GET_InventoryItems
--- PASS: TestProxyRouting_GET_InventoryItems (0.00s)
=== RUN   TestProxyRouting_GET_InventoryItemsByID
--- PASS: TestProxyRouting_GET_InventoryItemsByID (0.00s)
=== RUN   TestProxyRouting_GET_InventoryItemsBySKU
--- PASS: TestProxyRouting_GET_InventoryItemsBySKU (0.00s)
=== RUN   TestProxyRouting_POST_CreateItem
--- PASS: TestProxyRouting_POST_CreateItem (0.00s)
=== RUN   TestProxyRouting_POST_ReserveStock
--- PASS: TestProxyRouting_POST_ReserveStock (0.00s)
=== RUN   TestProxyRouting_POST_ReleaseStock
--- PASS: TestProxyRouting_POST_ReleaseStock (0.00s)
=== RUN   TestProxyRouting_POST_AdjustStock
--- PASS: TestProxyRouting_POST_AdjustStock (0.00s)
=== RUN   TestProxyRouting_PUT_UpdateItem
--- PASS: TestProxyRouting_PUT_UpdateItem (0.00s)
=== RUN   TestProxyRouting_DELETE_DeleteItem
--- PASS: TestProxyRouting_DELETE_DeleteItem (0.00s)
=== RUN   TestProxyRouting_POST_Login
--- PASS: TestProxyRouting_POST_Login (0.00s)
=== RUN   TestProxyRouting_QueryParamsPreservation
--- PASS: TestProxyRouting_QueryParamsPreservation (0.00s)
=== RUN   TestCORSHeaders
--- PASS: TestCORSHeaders (0.00s)
=== RUN   TestCORS_PreflightRequest
--- PASS: TestCORS_PreflightRequest (0.00s)
PASS
```

## 🔍 Qué Verifican los Tests

### 1. **Enrutamiento Correcto**
- Las peticiones GET se redirigen al Query Service (8081)
- Las peticiones POST/PUT/DELETE se redirigen al Command Service (8080)
- Las peticiones de autenticación se redirigen al Command Service (8080)

### 2. **Preservación de Rutas**
- Las rutas con parámetros dinámicos se preservan correctamente
- Las rutas completas (incluyendo `/reserve`, `/release`, `/adjust`) se preservan
- Los query params se preservan en las peticiones GET

### 3. **Preservación de Métodos HTTP**
- GET se preserva para consultas
- POST se preserva para comandos
- PUT se preserva para actualizaciones
- DELETE se preserva para eliminaciones

### 4. **Preservación de Body**
- El body de las peticiones POST/PUT se preserva correctamente
- El contenido del body se verifica en los tests

### 5. **Headers CORS**
- Los headers CORS se agregan a todas las respuestas
- Las peticiones OPTIONS (preflight) se manejan correctamente

## 🛠️ Arquitectura de los Tests

Los tests utilizan:
- **`httptest.NewServer`**: Para crear servidores mock de Query y Command Services
- **`httptest.NewRequest`**: Para crear peticiones HTTP de prueba
- **`httptest.NewRecorder`**: Para capturar las respuestas HTTP
- **`httputil.ReverseProxy`**: Para crear proxies de prueba

Cada test:
1. Crea servidores mock para Query y Command Services
2. Crea el proxy con los servidores mock
3. Crea una petición HTTP de prueba
4. Ejecuta la petición a través del proxy
5. Verifica que la petición se redirija al servicio correcto
6. Verifica que la ruta, método, y body se preserven correctamente

## 📝 Notas

- Los tests son **unitarios** y no requieren que los servicios reales estén corriendo
- Los tests utilizan servidores mock para simular los servicios backend
- Los tests verifican el comportamiento del proxy, no de los servicios backend
- Los tests cubren todos los casos de uso principales del Dashboard HTML

## 🔄 Agregar Nuevos Tests

Para agregar un nuevo test:

1. Crear una función de test con el prefijo `TestProxyRouting_`
2. Crear servidores mock para Query y Command Services
3. Crear el proxy con los servidores mock
4. Crear una petición HTTP de prueba
5. Ejecutar la petición y verificar los resultados

Ejemplo:
```go
func TestProxyRouting_NEW_FEATURE(t *testing.T) {
    // Crear servidor mock
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verificaciones
    }))
    defer server.Close()

    // Crear proxy y handler
    proxy := createProxy(server.URL)
    handler := createProxyHandler(proxy, proxy)

    // Crear petición
    req := httptest.NewRequest("METHOD", "/path", nil)
    w := httptest.NewRecorder()

    // Ejecutar
    handler.ServeHTTP(w, req)

    // Verificar
    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
    }
}
```

---

**Última actualización**: 2025
**Versión**: 1.0

