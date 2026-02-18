#!/bin/bash

# Script de pruebas para X-Request-ID e idempotencia en Command Service
# Prueba la generación automática, uso proporcionado y detección de duplicados

set -e

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuración
COMMAND_SERVICE="${COMMAND_SERVICE:-http://localhost:8080}"
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
    
    response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/auth/login" \
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
    
    response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X GET "$COMMAND_SERVICE/api/v1/health" \
        -H "Content-Type: application/json")
    
    http_code=$(echo "$response" | grep "HTTP_CODE" | cut -d: -f2)
    headers=$(curl -s -I -X GET "$COMMAND_SERVICE/api/v1/health" | grep -i "x-request-id")
    
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
    
    response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X GET "$COMMAND_SERVICE/api/v1/health" \
        -H "Content-Type: application/json" \
        -H "X-Request-ID: $provided_id")
    
    http_code=$(echo "$response" | grep "HTTP_CODE" | cut -d: -f2)
    headers=$(curl -s -I -X GET "$COMMAND_SERVICE/api/v1/health" \
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

# Test 3: Idempotencia - Crear item con X-Request-ID
test_idempotency_create_item() {
    echo -e "\n${BLUE}=== Test 3: Idempotencia - Crear item con X-Request-ID ===${NC}"
    
    request_id=$(generate_uuid)
    sku="SKU-TEST-IDEMPOTENCY-$(date +%s)"
    
    # Primera request
    echo -e "${YELLOW}Enviando primera request...${NC}"
    response1=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/inventory/items" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -H "X-Request-ID: $request_id" \
        -d "{
            \"sku\": \"$sku\",
            \"name\": \"Item Test Idempotencia\",
            \"description\": \"Item para probar idempotencia\",
            \"quantity\": 100
        }")
    
    http_code1=$(echo "$response1" | grep "HTTP_CODE" | cut -d: -f2)
    body1=$(echo "$response1" | sed '/HTTP_CODE/d')
    
    if [ "$http_code1" != "201" ] && [ "$http_code1" != "200" ]; then
        print_result "Idempotencia - Primera request" "FAIL" "HTTP $http_code1 - $body1"
        return
    fi
    
    # Segunda request (duplicada)
    echo -e "${YELLOW}Enviando segunda request (duplicada)...${NC}"
    sleep 1  # Pequeña pausa para asegurar que la primera request se procesó
    
    response2=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/inventory/items" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -H "X-Request-ID: $request_id" \
        -d "{
            \"sku\": \"$sku\",
            \"name\": \"Item Test Idempotencia\",
            \"description\": \"Item para probar idempotencia\",
            \"quantity\": 100
        }")
    
    http_code2=$(echo "$response2" | grep "HTTP_CODE" | cut -d: -f2)
    body2=$(echo "$response2" | sed '/HTTP_CODE/d')
    
    # Verificar que la segunda request retorna respuesta cacheada (200) o la misma respuesta
    if [ "$http_code2" = "200" ]; then
        # Comparar respuestas (deben ser idénticas)
        if [ "$body1" = "$body2" ]; then
            print_result "Idempotencia - Crear item" "PASS" "Request duplicado retornó respuesta cacheada (HTTP 200)"
        else
            print_result "Idempotencia - Crear item" "FAIL" "Request duplicado retornó respuesta diferente"
            echo -e "  ${YELLOW}Primera respuesta:${NC} $body1"
            echo -e "  ${YELLOW}Segunda respuesta:${NC} $body2"
        fi
    else
        print_result "Idempotencia - Crear item" "FAIL" "Segunda request retornó HTTP $http_code2 (esperado 200 para duplicado)"
        echo -e "  ${YELLOW}Respuesta:${NC} $body2"
    fi
}

# Test 4: Idempotencia - Ajustar stock con X-Request-ID
test_idempotency_adjust_stock() {
    echo -e "\n${BLUE}=== Test 4: Idempotencia - Ajustar stock con X-Request-ID ===${NC}"
    
    # Primero crear un item
    sku="SKU-TEST-ADJUST-$(date +%s)"
    create_response=$(curl -s -X POST "$COMMAND_SERVICE/api/v1/inventory/items" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -d "{
            \"sku\": \"$sku\",
            \"name\": \"Item Test Adjust\",
            \"quantity\": 100
        }")
    
    item_id=$(echo "$create_response" | grep -o '"id":"[^"]*' | cut -d'"' -f4)
    if [ -z "$item_id" ]; then
        print_result "Idempotencia - Ajustar stock (setup)" "FAIL" "No se pudo crear item para prueba"
        return
    fi
    
    request_id=$(generate_uuid)
    
    # Primera request
    echo -e "${YELLOW}Enviando primera request de ajuste...${NC}"
    response1=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/inventory/items/$item_id/adjust" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -H "X-Request-ID: $request_id" \
        -d "{\"quantity\": 10}")
    
    http_code1=$(echo "$response1" | grep "HTTP_CODE" | cut -d: -f2)
    body1=$(echo "$response1" | sed '/HTTP_CODE/d')
    
    if [ "$http_code1" != "200" ]; then
        print_result "Idempotencia - Primera request de ajuste" "FAIL" "HTTP $http_code1 - $body1"
        return
    fi
    
    # Segunda request (duplicada)
    echo -e "${YELLOW}Enviando segunda request de ajuste (duplicada)...${NC}"
    sleep 1
    
    response2=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/inventory/items/$item_id/adjust" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -H "X-Request-ID: $request_id" \
        -d "{\"quantity\": 10}")
    
    http_code2=$(echo "$response2" | grep "HTTP_CODE" | cut -d: -f2)
    body2=$(echo "$response2" | sed '/HTTP_CODE/d')
    
    if [ "$http_code2" = "200" ] && [ "$body1" = "$body2" ]; then
        print_result "Idempotencia - Ajustar stock" "PASS" "Request duplicado retornó respuesta cacheada"
    else
        print_result "Idempotency - Ajustar stock" "FAIL" "HTTP $http_code2 o respuestas diferentes"
    fi
}

# Test 5: Verificar X-Request-ID en headers de respuesta
test_request_id_in_response_headers() {
    echo -e "\n${BLUE}=== Test 5: Verificar X-Request-ID en headers de respuesta ===${NC}"
    
    provided_id=$(generate_uuid)
    
    headers=$(curl -s -I -X GET "$COMMAND_SERVICE/api/v1/health" \
        -H "X-Request-ID: $provided_id")
    
    x_request_id=$(echo "$headers" | grep -i "x-request-id" | cut -d: -f2 | tr -d ' \r\n')
    
    if [ -n "$x_request_id" ] && [ "$x_request_id" = "$provided_id" ]; then
        print_result "X-Request-ID en headers de respuesta" "PASS" "Header presente y correcto: $x_request_id"
    else
        print_result "X-Request-ID en headers de respuesta" "FAIL" "Header no encontrado o incorrecto. Esperado: $provided_id, Obtenido: $x_request_id"
    fi
}

# Test 6: Requests diferentes con mismo X-Request-ID (debe detectar duplicado)
test_different_requests_same_id() {
    echo -e "\n${BLUE}=== Test 6: Requests diferentes con mismo X-Request-ID ===${NC}"
    
    request_id=$(generate_uuid)
    sku1="SKU-TEST-DIFF1-$(date +%s)"
    sku2="SKU-TEST-DIFF2-$(date +%s)"
    
    # Primera request
    echo -e "${YELLOW}Enviando primera request...${NC}"
    response1=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/inventory/items" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -H "X-Request-ID: $request_id" \
        -d "{
            \"sku\": \"$sku1\",
            \"name\": \"Item Test 1\",
            \"quantity\": 50
        }")
    
    http_code1=$(echo "$response1" | grep "HTTP_CODE" | cut -d: -f2)
    body1=$(echo "$response1" | sed '/HTTP_CODE/d')
    
    if [ "$http_code1" != "201" ] && [ "$http_code1" != "200" ]; then
        print_result "Requests diferentes con mismo ID (setup)" "FAIL" "HTTP $http_code1"
        return
    fi
    
    # Segunda request con mismo X-Request-ID pero datos diferentes
    echo -e "${YELLOW}Enviando segunda request con mismo X-Request-ID pero datos diferentes...${NC}"
    sleep 1
    
    response2=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/inventory/items" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -H "X-Request-ID: $request_id" \
        -d "{
            \"sku\": \"$sku2\",
            \"name\": \"Item Test 2\",
            \"quantity\": 75
        }")
    
    http_code2=$(echo "$response2" | grep "HTTP_CODE" | cut -d: -f2)
    body2=$(echo "$response2" | sed '/HTTP_CODE/d')
    
    # Debe retornar la respuesta cacheada de la primera request (idempotencia)
    if [ "$http_code2" = "200" ] && [ "$body1" = "$body2" ]; then
        print_result "Requests diferentes con mismo X-Request-ID" "PASS" "Sistema detectó duplicado y retornó respuesta cacheada (idempotencia funcionando)"
    else
        print_result "Requests diferentes con mismo X-Request-ID" "FAIL" "HTTP $http_code2 - Sistema no detectó duplicado correctamente"
        echo -e "  ${YELLOW}Nota:${NC} Esto es esperado - el sistema debe retornar la respuesta cacheada basada en X-Request-ID, no en el contenido"
    fi
}

# Función principal
main() {
    echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║  Pruebas de X-Request-ID e Idempotencia - Command Service ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
    
    # Verificar que el servicio esté disponible
    if ! curl -s -f "$COMMAND_SERVICE/api/v1/health" > /dev/null; then
        echo -e "${RED}Error: El servicio Command Service no está disponible en $COMMAND_SERVICE${NC}"
        exit 1
    fi
    
    # Obtener token JWT
    get_jwt_token
    
    # Ejecutar tests
    test_auto_generate_request_id
    test_provided_request_id
    test_idempotency_create_item
    test_idempotency_adjust_stock
    test_request_id_in_response_headers
    test_different_requests_same_id
    
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

