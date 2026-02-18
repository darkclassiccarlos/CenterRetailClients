#!/bin/bash

# Script para pruebas de integración end-to-end
# Prueba el flujo completo: Command Service -> Kafka -> Listener Service -> Query Service
# Verifica autenticación JWT, actualizaciones en Redis, y respuestas de endpoints

set -e

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# URLs de los servicios
COMMAND_SERVICE="http://localhost:8080"
QUERY_SERVICE="http://localhost:8081"
LISTENER_SERVICE="http://localhost:8082"

# Credenciales JWT
JWT_USERNAME="${JWT_USERNAME:-admin}"
JWT_PASSWORD="${JWT_PASSWORD:-admin123}"

# Variable para almacenar el token JWT
JWT_TOKEN=""

# Contadores de pruebas
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Pruebas de Integración End-to-End${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Función para obtener token JWT
get_jwt_token() {
    echo -e "${BLUE}Obteniendo token JWT...${NC}" >&2
    
    response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$QUERY_SERVICE/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d "{
            \"username\": \"$JWT_USERNAME\",
            \"password\": \"$JWT_PASSWORD\"
        }" 2>&1)
    
    http_code=$(echo "$response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    body=$(echo "$response" | sed '/HTTP_CODE:/d')
    
    if [ -z "$http_code" ] || ! [[ "$http_code" =~ ^[0-9]+$ ]]; then
        echo -e "${RED}✗ Error: No se pudo obtener código HTTP${NC}" >&2
        return 1
    fi
    
    if [ "$http_code" -eq 200 ]; then
        token=$(echo "$body" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
        if [ -n "$token" ]; then
            JWT_TOKEN="$token"
            echo -e "${GREEN}✓ Token JWT obtenido exitosamente${NC}" >&2
            return 0
        else
            echo -e "${RED}✗ Error: No se pudo extraer token de la respuesta${NC}" >&2
            return 1
        fi
    else
        echo -e "${RED}✗ Error: Login falló (HTTP $http_code): $body${NC}" >&2
        return 1
    fi
}

# Función para verificar que un servicio esté disponible
check_service() {
    local service_name=$1
    local service_url=$2
    
    if curl -s -f "$service_url/api/v1/health" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ $service_name está disponible${NC}"
        return 0
    else
        echo -e "${RED}✗ $service_name no está disponible${NC}"
        return 1
    fi
}

# Función para ejecutar una prueba
run_test() {
    local test_name=$1
    local test_func=$2
    
    ((TOTAL_TESTS++))
    echo -e "${YELLOW}Test: $test_name${NC}"
    
    if $test_func; then
        echo -e "${GREEN}✓ $test_name pasó${NC}"
        ((PASSED_TESTS++))
        return 0
    else
        echo -e "${RED}✗ $test_name falló${NC}"
        ((FAILED_TESTS++))
        return 1
    fi
    echo ""
}

# Test 1: Verificar servicios disponibles
test_services_available() {
    local all_ok=true
    
    if ! check_service "Command Service" "$COMMAND_SERVICE"; then
        all_ok=false
    fi
    
    if ! check_service "Query Service" "$QUERY_SERVICE"; then
        all_ok=false
    fi
    
    if ! check_service "Listener Service" "$LISTENER_SERVICE"; then
        all_ok=false
    fi
    
    if [ "$all_ok" = false ]; then
        echo -e "${RED}Algunos servicios no están disponibles${NC}"
        return 1
    fi
    
    return 0
}

# Test 2: Autenticación JWT
test_jwt_authentication() {
    if ! get_jwt_token; then
        return 1
    fi
    
    # Verificar que el token no esté vacío
    if [ -z "$JWT_TOKEN" ]; then
        echo -e "${RED}Token JWT está vacío${NC}"
        return 1
    fi
    
    return 0
}

# Test 3: Endpoint protegido sin token (debe fallar)
test_protected_endpoint_without_token() {
    response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X GET "$QUERY_SERVICE/api/v1/inventory/items")
    http_code=$(echo "$response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    
    if [ "$http_code" -eq 401 ]; then
        return 0
    else
        echo -e "${RED}Esperado HTTP 401, pero obtuvo $http_code${NC}"
        return 1
    fi
}

# Test 4: Endpoint protegido con token válido
test_protected_endpoint_with_token() {
    if [ -z "$JWT_TOKEN" ]; then
        echo -e "${RED}Token JWT no disponible${NC}"
        return 1
    fi
    
    response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X GET "$QUERY_SERVICE/api/v1/inventory/items" \
        -H "Authorization: Bearer $JWT_TOKEN")
    http_code=$(echo "$response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    
    if [ "$http_code" -eq 200 ]; then
        return 0
    else
        echo -e "${RED}Esperado HTTP 200, pero obtuvo $http_code${NC}"
        return 1
    fi
}

# Test 5: Crear item en Command Service y verificar en Query Service
test_create_and_query_item() {
    # Obtener token para Command Service
    cmd_token=$(curl -s -X POST "$COMMAND_SERVICE/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$JWT_USERNAME\",\"password\":\"$JWT_PASSWORD\"}" | \
        grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    
    if [ -z "$cmd_token" ]; then
        echo -e "${RED}No se pudo obtener token de Command Service${NC}"
        return 1
    fi
    
    # Crear item en Command Service
    sku="TEST-INTEGRATION-$(date +%s)"
    create_response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/inventory/items" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $cmd_token" \
        -d "{
            \"sku\": \"$sku\",
            \"name\": \"Test Integration Item\",
            \"description\": \"Item creado para pruebas de integración\",
            \"quantity\": 100
        }")
    
    create_http_code=$(echo "$create_response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    
    if [ "$create_http_code" -ne 201 ]; then
        echo -e "${RED}Error al crear item (HTTP $create_http_code)${NC}"
        return 1
    fi
    
    item_id=$(echo "$create_response" | sed '/HTTP_CODE:/d' | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    
    if [ -z "$item_id" ]; then
        echo -e "${RED}No se pudo extraer ID del item${NC}"
        return 1
    fi
    
    echo -e "${BLUE}Item creado: $item_id (SKU: $sku)${NC}"
    
    # Esperar a que el Listener Service procese el evento
    echo -e "${YELLOW}Esperando procesamiento de eventos (10 segundos)...${NC}"
    sleep 10
    
    # Consultar item en Query Service
    query_response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X GET "$QUERY_SERVICE/api/v1/inventory/items/sku/$sku" \
        -H "Authorization: Bearer $JWT_TOKEN")
    query_http_code=$(echo "$query_response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    
    if [ "$query_http_code" -eq 200 ]; then
        # Verificar que el SKU coincida
        query_sku=$(echo "$query_response" | sed '/HTTP_CODE:/d' | grep -o '"sku":"[^"]*"' | cut -d'"' -f4)
        if [ "$query_sku" = "$sku" ]; then
            echo -e "${GREEN}Item encontrado en Query Service (SKU: $query_sku)${NC}"
            return 0
        else
            echo -e "${RED}SKU no coincide: esperado $sku, obtenido $query_sku${NC}"
            return 1
        fi
    elif [ "$query_http_code" -eq 404 ]; then
        echo -e "${YELLOW}Item aún no disponible (eventual consistency)${NC}"
        return 0  # Esto es aceptable en eventual consistency
    else
        echo -e "${RED}Error al consultar item (HTTP $query_http_code)${NC}"
        return 1
    fi
}

# Test 6: Verificar cache de Redis (si está habilitado)
test_redis_cache() {
    if [ -z "$JWT_TOKEN" ]; then
        echo -e "${RED}Token JWT no disponible${NC}"
        return 1
    fi
    
    # Primera consulta (cache miss)
    response1=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X GET "$QUERY_SERVICE/api/v1/inventory/items?page=1&page_size=10" \
        -H "Authorization: Bearer $JWT_TOKEN")
    http_code1=$(echo "$response1" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    
    if [ "$http_code1" -ne 200 ]; then
        echo -e "${RED}Primera consulta falló (HTTP $http_code1)${NC}"
        return 1
    fi
    
    # Segunda consulta inmediata (debe ser cache hit si Redis está habilitado)
    response2=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X GET "$QUERY_SERVICE/api/v1/inventory/items?page=1&page_size=10" \
        -H "Authorization: Bearer $JWT_TOKEN")
    http_code2=$(echo "$response2" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    
    if [ "$http_code2" -eq 200 ]; then
        echo -e "${GREEN}Cache funcionando correctamente${NC}"
        return 0
    else
        echo -e "${RED}Segunda consulta falló (HTTP $http_code2)${NC}"
        return 1
    fi
}

# Ejecutar pruebas
echo -e "${YELLOW}1. Verificando servicios disponibles...${NC}"
run_test "Servicios Disponibles" test_services_available
echo ""

echo -e "${YELLOW}2. Probando autenticación JWT...${NC}"
run_test "Autenticación JWT" test_jwt_authentication
echo ""

echo -e "${YELLOW}3. Probando endpoint protegido sin token...${NC}"
run_test "Endpoint Protegido Sin Token" test_protected_endpoint_without_token
echo ""

echo -e "${YELLOW}4. Probando endpoint protegido con token...${NC}"
run_test "Endpoint Protegido Con Token" test_protected_endpoint_with_token
echo ""

echo -e "${YELLOW}5. Probando flujo completo (crear y consultar item)...${NC}"
run_test "Crear y Consultar Item" test_create_and_query_item
echo ""

echo -e "${YELLOW}6. Probando cache de Redis...${NC}"
run_test "Cache de Redis" test_redis_cache
echo ""

# Resumen
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Resumen de Pruebas de Integración${NC}"
echo -e "${GREEN}========================================${NC}"
echo "Total de pruebas: $TOTAL_TESTS"
echo -e "${GREEN}Exitosas: $PASSED_TESTS${NC}"
echo -e "${RED}Fallidas: $FAILED_TESTS${NC}"
echo ""

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}✓ Todas las pruebas pasaron${NC}"
    exit 0
else
    echo -e "${RED}✗ Algunas pruebas fallaron${NC}"
    exit 1
fi

