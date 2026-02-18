# Actualización de Scripts de Prueba con Autenticación JWT

## Resumen

Todos los scripts de prueba han sido actualizados para incluir autenticación JWT/OAuth2. Los scripts ahora:

1. **Obtienen un token JWT** al inicio mediante login
2. **Incluyen el token** en todas las peticiones a endpoints protegidos
3. **Manejan errores de autenticación** apropiadamente

## Scripts Actualizados

### 1. `test_release_stock.sh`
- ✅ Agregada función `get_jwt_token()`
- ✅ Autenticación al inicio del script
- ✅ Header `Authorization: Bearer $JWT_TOKEN` en:
  - `POST /api/v1/inventory/items` (crear item)
  - `POST /api/v1/inventory/items/{id}/reserve` (reservar stock)
  - `POST /api/v1/inventory/items/{id}/release` (liberar stock)

### 2. `test_stock_operations.sh`
- ✅ Agregada función `get_jwt_token()`
- ✅ Autenticación al inicio del script
- ✅ Header `Authorization: Bearer $JWT_TOKEN` en:
  - `POST /api/v1/inventory/items` (crear item)
  - `POST /api/v1/inventory/items/{id}/adjust` (ajustar stock)
  - `POST /api/v1/inventory/items/{id}/reserve` (reservar stock)
  - `POST /api/v1/inventory/items/{id}/release` (liberar stock)

### 3. `test_e2e_flow.sh`
- ✅ Agregada función `get_jwt_token()`
- ✅ Autenticación al inicio del script
- ✅ Header `Authorization: Bearer $JWT_TOKEN` en:
  - `POST /api/v1/inventory/items` (crear item)
  - `POST /api/v1/inventory/items/{id}/adjust` (ajustar stock)
  - `POST /api/v1/inventory/items/{id}/reserve` (reservar stock)

### 4. `auth_helper.sh` (Nuevo)
- Script helper con funciones reutilizables para autenticación JWT
- Funciones:
  - `get_jwt_token(username, password)` - Obtiene token JWT
  - `login(username, password)` - Hace login y almacena token
  - `get_stored_token()` - Obtiene token almacenado
  - `set_token(token)` - Establece token
  - `is_token_set()` - Verifica si token está configurado

## Credenciales por Defecto

Los scripts usan las siguientes credenciales por defecto:
- **Usuario**: `admin`
- **Password**: `admin123`

Estas pueden ser sobrescritas usando variables de entorno:
```bash
export JWT_USERNAME="user"
export JWT_PASSWORD="user123"
./scripts/test_release_stock.sh
```

## Endpoints Protegidos

Los siguientes endpoints requieren autenticación JWT:
- `POST /api/v1/inventory/items` - Crear item
- `PUT /api/v1/inventory/items/{id}` - Actualizar item
- `DELETE /api/v1/inventory/items/{id}` - Eliminar item
- `POST /api/v1/inventory/items/{id}/adjust` - Ajustar stock
- `POST /api/v1/inventory/items/{id}/reserve` - Reservar stock
- `POST /api/v1/inventory/items/{id}/release` - Liberar stock

## Endpoints Públicos

Los siguientes endpoints NO requieren autenticación:
- `GET /api/v1/health` - Health check
- `POST /api/v1/auth/login` - Login (obtener token)

## Ejecución de Scripts

### Prerequisitos
1. El Command Service debe estar corriendo en `http://localhost:8080`
2. Para pruebas E2E, también se necesitan:
   - Query Service en `http://localhost:8081`
   - Listener Service en `http://localhost:8082`
   - Kafka en `localhost:9093`

### Ejecutar Scripts

```bash
# Desde el directorio command-service
cd command-service

# Test de liberación de stock
./scripts/test_release_stock.sh

# Test de operaciones de stock
./scripts/test_stock_operations.sh

# Test end-to-end completo
./scripts/test_e2e_flow.sh
```

### Con Credenciales Personalizadas

```bash
JWT_USERNAME="user" JWT_PASSWORD="user123" ./scripts/test_release_stock.sh
```

## Manejo de Errores

Los scripts manejan los siguientes errores:

1. **Error de conexión**: Si el servicio no está disponible, el script aborta
2. **Error de autenticación**: Si el login falla, el script aborta
3. **Token inválido**: Si el token no es válido, las peticiones fallan con 401
4. **Token expirado**: Si el token expiró (10 minutos), se requiere nuevo login

## Notas

- Los tokens JWT tienen una expiración de **10 minutos**
- Si un script tarda más de 10 minutos, el token puede expirar
- Los scripts obtienen un nuevo token al inicio, por lo que esto no debería ser un problema en la mayoría de los casos
- Para scripts muy largos, se puede implementar renovación automática de tokens

