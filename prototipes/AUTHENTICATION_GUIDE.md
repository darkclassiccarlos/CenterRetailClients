# Guía de Autenticación - Uso Correcto del Token JWT

## 🔐 Formato Correcto del Header Authorization

El token JWT debe enviarse en el header `Authorization` con el formato:

```
Authorization: Bearer <token>
```

**Importante:** Debe incluir la palabra "Bearer" seguida de un espacio y luego el token.

## ✅ Ejemplos Correctos con curl

### 1. Obtener Token (Login)

```bash
# Query Service
curl -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin123"
  }'

# Command Service
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin123"
  }'
```

**Respuesta esperada:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "type": "Bearer",
  "expires_in": 600,
  "expires_at": "2025-11-09T13:30:43.547658-05:00"
}
```

### 2. Usar Token en Endpoints Protegidos

#### Query Service - Listar Items

```bash
# Guardar el token en una variable
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImFkbWluIiwiaXNzIjoicXVlcnktc2VydmljZSIsInN1YiI6ImFkbWluIiwiZXhwIjoxNzYyNzEzMDQzLCJuYmYiOjE3NjI3MTI0NDMsImlhdCI6MTc2MjcxMjQ0M30.YhA9gyxnk53I-BkuNgqswMFvqsMzXngZrqJCxRarhIE"

# Usar el token en el header Authorization
curl -X GET http://localhost:8081/api/v1/inventory/items \
  -H "Authorization: Bearer $TOKEN"
```

#### Query Service - Obtener Item por ID

```bash
curl -X GET http://localhost:8081/api/v1/inventory/items/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer $TOKEN"
```

#### Command Service - Crear Item

```bash
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

curl -X POST http://localhost:8080/api/v1/inventory/items \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "sku": "SKU-001",
    "name": "Test Product",
    "description": "Test Description",
    "quantity": 100
  }'
```

## ❌ Errores Comunes

### Error 1: Token sin prefijo "Bearer"

**❌ Incorrecto:**
```bash
curl -X GET http://localhost:8081/api/v1/inventory/items \
  -H "Authorization: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

**✅ Correcto:**
```bash
curl -X GET http://localhost:8081/api/v1/inventory/items \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### Error 2: Token sin espacio después de "Bearer"

**❌ Incorrecto:**
```bash
curl -X GET http://localhost:8081/api/v1/inventory/items \
  -H "Authorization: BearereyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

**✅ Correcto:**
```bash
curl -X GET http://localhost:8081/api/v1/inventory/items \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### Error 3: Token en header diferente

**❌ Incorrecto:**
```bash
curl -X GET http://localhost:8081/api/v1/inventory/items \
  -H "X-Auth-Token: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

**✅ Correcto:**
```bash
curl -X GET http://localhost:8081/api/v1/inventory/items \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

## 📝 Script de Ejemplo Completo

### Query Service

```bash
#!/bin/bash

# 1. Obtener token
echo "Obteniendo token..."
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin123"
  }')

# 2. Extraer token de la respuesta
TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*' | cut -d'"' -f4)

echo "Token obtenido: $TOKEN"
echo ""

# 3. Usar token en endpoint protegido
echo "Consultando items..."
curl -X GET http://localhost:8081/api/v1/inventory/items \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json"
```

### Command Service

```bash
#!/bin/bash

# 1. Obtener token
echo "Obteniendo token..."
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin123"
  }')

# 2. Extraer token de la respuesta
TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*' | cut -d'"' -f4)

echo "Token obtenido: $TOKEN"
echo ""

# 3. Usar token en endpoint protegido
echo "Creando item..."
curl -X POST http://localhost:8080/api/v1/inventory/items \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "sku": "SKU-001",
    "name": "Test Product",
    "description": "Test Description",
    "quantity": 100
  }'
```

## 🔍 Verificación del Token

### Verificar que el token es válido

```bash
# Decodificar el token (solo para verificación, no para validación)
# El token JWT tiene 3 partes separadas por puntos: header.payload.signature

TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImFkbWluIiwiaXNzIjoicXVlcnktc2VydmljZSIsInN1YiI6ImFkbWluIiwiZXhwIjoxNzYyNzEzMDQzLCJuYmYiOjE3NjI3MTI0NDMsImlhdCI6MTc2MjcxMjQ0M30.YhA9gyxnk53I-BkuNgqswMFvqsMzXngZrqJCxRarhIE"

# Extraer el payload (segunda parte)
PAYLOAD=$(echo $TOKEN | cut -d'.' -f2)

# Decodificar base64 (agregar padding si es necesario)
echo $PAYLOAD | base64 -d 2>/dev/null || echo $PAYLOAD | base64 -d
```

## ⚠️ Errores Comunes y Soluciones

### Error: "invalid authorization header format"

**Causa:** El header `Authorization` no tiene el formato correcto `Bearer <token>`

**Solución:** Asegúrate de incluir "Bearer" seguido de un espacio y luego el token:

```bash
-H "Authorization: Bearer $TOKEN"
```

### Error: "token expired"

**Causa:** El token ha expirado (válido por 10 minutos)

**Solución:** Obtén un nuevo token desde el endpoint de login:

```bash
curl -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "admin123"}'
```

### Error: "invalid token"

**Causa:** El token es inválido o fue firmado con un secret diferente

**Solución:** 
1. Verifica que estés usando el token del servicio correcto (Query Service vs Command Service)
2. Obtén un nuevo token desde el endpoint de login
3. Verifica que el `JWT_SECRET` sea el mismo en ambos servicios

## 📚 Referencias

- **Query Service README**: Ver `query-service/README.md`
- **Command Service README**: Ver `command-service/README.md`
- **Swagger UI**: `http://localhost:8081/swagger/index.html` (Query Service) o `http://localhost:8080/swagger/index.html` (Command Service)

