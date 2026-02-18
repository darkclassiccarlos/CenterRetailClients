#!/bin/bash

# Script de pruebas para X-Request-ID en Query Service
# Prueba la generación automática, uso proporcionado y trazabilidad

set -e

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuración
QUERY_SERVICE="${QUERY_SERVICE:-http://localhost:8081}"
JWT_USERNAME="${JWT_USERNAME:-admin}"
JWT_PASSWORD="${JWT_PASSWORD:-admin123}"
JWT_TOKEN=""

# Contadores
TESTS_PASSED=0
TESTS_FAILED=0
TOTAL_TESTS=0

# Función para imprimir resultados
print_result() {
    local test_name=$1
    local status=$2
    local details=$3
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    if [ "$status" = "PASS" ]; then
        echo -e "${GREEN}✓ PASS${NC}: $test_name"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}✗ FAIL${NC}: $test_name"
        echo -e "  ${RED}Details:${NC} $details"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# Función para obtener token JWT
get_jwt_token() {
    echo -e "${BLUE}🔐 Obteniendo token JWT...${NC}"
    
    response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$QUERY_SERVICE/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d "{
            \"username\": \"$JWT_USERNAME\",
            \"password\": \"$JWT_PASSWORD\"
        }")
    
    http_code=$(echo "$response" | grep "HTTP_CODE" | cut -d: -f2)
    body=$(echo "$response" | sed '/HTTP_CODE/d')
    
    if [ "$http_code" = "200" ]; then
        JWT_TOKEN=$(echo "$body" | grep -o '"token":"[^"]*' | cut -d'"' -f4)
        if [ -z "$JWT_TOKEN" ]; then
            echo -e "${RED}Error: No se pudo extraer el token JWT${NC}"
            exit 1
        fi
        echo -e "${GREEN}✅ Token JWT obtenido exitosamente${NC}"
    else
        echo -e "${RED}Error: Fallo al obtener token JWT (HTTP $http_code)${NC}"
        echo "Response: $body"
        exit 1
    fi
}

# Función para generar UUID
generate_uuid() {
    if command -v uuidgen &> /dev/null; then
        uuidgen
    elif command -v python3 &> /dev/null; then
        python3 -c "import uuid; print(uuid.uuid4())"
    else
        # Fallback: generar UUID simple
        cat /dev/urandom | tr -dc 'a-f0-9' | fold -w 32 | sed 's/\(........\)\(....\)\(....\)\(....\)\(............\)/\1-\2-\3-\4-\5/'
    fi
}

# Test 1: Generación automática de X-Request-ID
test_auto_generate_request_id() {
    echo -e "\n${BLUE}=== Test 1: Generación automática de X-Request-ID ===${NC}"
    
    response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X GET "$QUERY_SERVICE/api/v1/health" \
        -H "Content-Type: application/json")
    
    http_code=$(echo "$response" | grep "HTTP_CODE" | cut -d: -f2)
    headers=$(curl -s -I -X GET "$QUERY_SERVICE/api/v1/health" | grep -i "x-request-id")
    
    if [ "$http_code" = "200" ] && [ -n "$headers" ]; then
        request_id=$(echo "$headers" | grep -i "x-request-id" | cut -d: -f2 | tr -d ' \r\n')
        if [ -n "$request_id" ]; then
            print_result "Generación automática de X-Request-ID" "PASS" "Request ID generado: $request_id"
        else
            print_result "Generación automática de X-Request-ID" "FAIL" "Header X-Request-ID vacío"
        fi
    else
        print_result "Generación automática de X-Request-ID" "FAIL" "HTTP $http_code o header no encontrado"
    fi
}

# Test 2: Uso de X-Request-ID proporcionado
test_provided_request_id() {
    echo -e "\n${BLUE}=== Test 2: Uso de X-Request-ID proporcionado ===${NC}"
    
    provided_id=$(generate_uuid)
    
    response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X GET "$QUERY_SERVICE/api/v1/health" \
        -H "Content-Type: application/json" \
        -H "X-Request-ID: $provided_id")
    
    http_code=$(echo "$response" | grep "HTTP_CODE" | cut -d: -f2)
    headers=$(curl -s -I -X GET "$QUERY_SERVICE/api/v1/health" \
        -H "X-Request-ID: $provided_id" | grep -i "x-request-id")
    
    if [ "$http_code" = "200" ]; then
        returned_id=$(echo "$headers" | grep -i "x-request-id" | cut -d: -f2 | tr -d ' \r\n')
        if [ "$returned_id" = "$provided_id" ]; then
            print_result "Uso de X-Request-ID proporcionado" "PASS" "Request ID retornado correctamente: $returned_id"
        else
            print_result "Uso de X-Request-ID proporcionado" "FAIL" "Request ID no coincide. Esperado: $provided_id, Obtenido: $returned_id"
        fi
    else
        print_result "Uso de X-Request-ID proporcionado" "FAIL" "HTTP $http_code"
    fi
}

# Test 3: X-Request-ID en consulta de items
test_request_id_list_items() {
    echo -e "\n${BLUE}=== Test 3: X-Request-ID en consulta de items ===${NC}"
    
    provided_id=$(generate_uuid)
    
    response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X GET "$QUERY_SERVICE/api/v1/inventory/items?page=1&page_size=10" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -H "X-Request-ID: $provided_id")
    
    http_code=$(echo "$response" | grep "HTTP_CODE" | cut -d: -f2)
    headers=$(curl -s -I -X GET "$QUERY_SERVICE/api/v1/inventory/items?page=1&page_size=10" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -H "X-Request-ID: $provided_id" | grep -i "x-request-id")
    
    if [ "$http_code" = "200" ]; then
        returned_id=$(echo "$headers" | grep -i "x-request-id" | cut -d: -f2 | tr -d ' \r\n')
        if [ "$returned_id" = "$provided_id" ]; then
            print_result "X-Request-ID en consulta de items" "PASS" "Request ID presente en respuesta: $returned_id"
        else
            print_result "X-Request-ID en consulta de items" "FAIL" "Request ID no coincide"
        fi
    else
        print_result "X-Request-ID en consulta de items" "FAIL" "HTTP $http_code"
    fi
}

# Test 4: X-Request-ID en consulta por ID
test_request_id_get_item_by_id() {
    echo -e "\n${BLUE}=== Test 4: X-Request-ID en consulta por ID ===${NC}"
    
    # Primero obtener un item ID de la lista
    list_response=$(curl -s -X GET "$QUERY_SERVICE/api/v1/inventory/items?page=1&page_size=1" \
        -H "Authorization: Bearer $JWT_TOKEN")
    
    item_id=$(echo "$list_response" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)
    
    if [ -z "$item_id" ]; then
        print_result "X-Request-ID en consulta por ID (setup)" "FAIL" "No se encontró ningún item para probar"
        return
    fi
    
    provided_id=$(generate_uuid)
    
    response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X GET "$QUERY_SERVICE/api/v1/inventory/items/$item_id" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -H "X-Request-ID: $provided_id")
    
    http_code=$(echo "$response" | grep "HTTP_CODE" | cut -d: -f2)
    headers=$(curl -s -I -X GET "$QUERY_SERVICE/api/v1/inventory/items/$item_id" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -H "X-Request-ID: $provided_id" | grep -i "x-request-id")
    
    if [ "$http_code" = "200" ] || [ "$http_code" = "404" ]; then
        returned_id=$(echo "$headers" | grep -i "x-request-id" | cut -d: -f2 | tr -d ' \r\n')
        if [ "$returned_id" = "$provided_id" ]; then
            print_result "X-Request-ID en consulta por ID" "PASS" "Request ID presente en respuesta: $returned_id"
        else
            print_result "X-Request-ID en consulta por ID" "FAIL" "Request ID no coincide"
        fi
    else
        print_result "X-Request-ID en consulta por ID" "FAIL" "HTTP $http_code"
    fi
}

# Test 5: Verificar X-Request-ID en headers de respuesta
test_request_id_in_response_headers() {
    echo -e "\n${BLUE}=== Test 5: Verificar X-Request-ID en headers de respuesta ===${NC}"
    
    provided_id=$(generate_uuid)
    
    headers=$(curl -s -I -X GET "$QUERY_SERVICE/api/v1/health" \
        -H "X-Request-ID: $provided_id")
    
    x_request_id=$(echo "$headers" | grep -i "x-request-id" | cut -d: -f2 | tr -d ' \r\n')
    
    if [ -n "$x_request_id" ] && [ "$x_request_id" = "$provided_id" ]; then
        print_result "X-Request-ID en headers de respuesta" "PASS" "Header presente y correcto: $x_request_id"
    else
        print_result "X-Request-ID en headers de respuesta" "FAIL" "Header no encontrado o incorrecto. Esperado: $provided_id, Obtenido: $x_request_id"
    fi
}

# Test 6: Múltiples requests con mismo X-Request-ID (trazabilidad)
test_multiple_requests_same_id() {
    echo -e "\n${BLUE}=== Test 6: Múltiples requests con mismo X-Request-ID (trazabilidad) ===${NC}"
    
    provided_id=$(generate_uuid)
    
    # Realizar múltiples requests con el mismo X-Request-ID
    response1=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X GET "$QUERY_SERVICE/api/v1/inventory/items?page=1&page_size=5" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -H "X-Request-ID: $provided_id")
    
    http_code1=$(echo "$response1" | grep "HTTP_CODE" | cut -d: -f2)
    
    sleep 0.5
    
    response2=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X GET "$QUERY_SERVICE/api/v1/inventory/items?page=2&page_size=5" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -H "X-Request-ID: $provided_id")
    
    http_code2=$(echo "$response2" | grep "HTTP_CODE" | cut -d: -f2)
    
    # Verificar que ambos requests retornaron el mismo X-Request-ID
    headers1=$(curl -s -I -X GET "$QUERY_SERVICE/api/v1/inventory/items?page=1&page_size=5" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -H "X-Request-ID: $provided_id" | grep -i "x-request-id")
    
    headers2=$(curl -s -I -X GET "$QUERY_SERVICE/api/v1/inventory/items?page=2&page_size=5" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -H "X-Request-ID: $provided_id" | grep -i "x-request-id")
    
    id1=$(echo "$headers1" | grep -i "x-request-id" | cut -d: -f2 | tr -d ' \r\n')
    id2=$(echo "$headers2" | grep -i "x-request-id" | cut -d: -f2 | tr -d ' \r\n')
    
    if [ "$http_code1" = "200" ] && [ "$http_code2" = "200" ] && [ "$id1" = "$provided_id" ] && [ "$id2" = "$provided_id" ]; then
        print_result "Múltiples requests con mismo X-Request-ID" "PASS" "Todas las requests retornaron el mismo Request ID: $provided_id"
    else
        print_result "Múltiples requests con mismo X-Request-ID" "FAIL" "Request IDs no coinciden o HTTP codes incorrectos"
    fi
}

# Función principal
main() {
    echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║     Pruebas de X-Request-ID y Trazabilidad - Query Service║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
    
    # Verificar que el servicio esté disponible
    if ! curl -s -f "$QUERY_SERVICE/api/v1/health" > /dev/null; then
        echo -e "${RED}Error: El servicio Query Service no está disponible en $QUERY_SERVICE${NC}"
        exit 1
    fi
    
    # Obtener token JWT
    get_jwt_token
    
    # Ejecutar tests
    test_auto_generate_request_id
    test_provided_request_id
    test_request_id_list_items
    test_request_id_get_item_by_id
    test_request_id_in_response_headers
    test_multiple_requests_same_id
    
    # Resumen
    echo -e "\n${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║                      RESUMEN DE PRUEBAS                     ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
    echo -e "Total de pruebas: ${BLUE}$TOTAL_TESTS${NC}"
    echo -e "Pruebas exitosas: ${GREEN}$TESTS_PASSED${NC}"
    echo -e "Pruebas fallidas: ${RED}$TESTS_FAILED${NC}"
    
    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "\n${GREEN}✅ Todas las pruebas pasaron exitosamente${NC}"
        exit 0
    else
        echo -e "\n${RED}❌ Algunas pruebas fallaron${NC}"
        exit 1
    fi
}

# Ejecutar función principal
main

