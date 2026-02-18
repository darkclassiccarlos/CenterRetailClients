#!/bin/bash

# Script para probar actualización de Redis en query-service
# después de actualizaciones en BD realizadas por listener-service

set -e

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuración
COMMAND_SERVICE="http://localhost:8080"
QUERY_SERVICE="http://localhost:8081"
LISTENER_SERVICE="http://localhost:8082"

# Contadores
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Función para obtener token JWT
get_jwt_token() {
    local service=$1
    local response=$(curl -s -X POST "$service/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"username": "admin", "password": "admin123"}')
    
    echo "$response" | grep -o '"token":"[^"]*' | cut -d'"' -f4
}

# Función para esperar procesamiento de eventos
wait_for_event_processing() {
    local seconds=${1:-5}
    echo -e "${YELLOW}Esperando $seconds segundos para procesamiento de eventos...${NC}"
    sleep $seconds
}

# Función para verificar item en Redis (indirectamente a través de query-service)
verify_item_in_cache() {
    local item_id=$1
    local expected_quantity=$2
    local token=$3
    
    local response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X GET "$QUERY_SERVICE/api/v1/inventory/items/$item_id" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json")
    
    local http_code=$(echo "$response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    local body=$(echo "$response" | sed '/HTTP_CODE:/d')
    
    if [ "$http_code" -eq 200 ]; then
        local quantity=$(echo "$body" | grep -o '"quantity":[0-9]*' | cut -d':' -f2)
        if [ "$quantity" == "$expected_quantity" ]; then
            return 0
        else
            echo -e "${RED}✗ Cantidad incorrecta en cache: esperado $expected_quantity, obtenido $quantity${NC}"
            return 1
        fi
    else
        echo -e "${RED}✗ Error al consultar item: HTTP $http_code${NC}"
        return 1
    fi
}

# Test 1: Crear item y verificar actualización en Redis
test_create_item_redis_update() {
    local test_name="Crear Item y Verificar Actualización en Redis"
    echo -e "${BLUE}--- Ejecutando prueba: $test_name ---${NC}"
    
    # Obtener tokens para ambos servicios
    local cmd_token=$(get_jwt_token "$COMMAND_SERVICE")
    local query_token=$(get_jwt_token "$QUERY_SERVICE")
    if [ -z "$cmd_token" ] || [ -z "$query_token" ]; then
        echo -e "${RED}✗ No se pudo obtener token JWT${NC}"
        return 1
    fi
    
    # Crear item
    local sku="SKU-REDIS-TEST-$(date +%s)"
    local name="Item Redis Test"
    local quantity=100
    
    echo -e "${YELLOW}Creando item en Command Service...${NC}"
    local create_response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/inventory/items" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $cmd_token" \
        -d "{\"sku\": \"$sku\", \"name\": \"$name\", \"description\": \"Item para prueba de Redis\", \"quantity\": $quantity}")
    
    local create_http_code=$(echo "$create_response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    local create_body=$(echo "$create_response" | sed '/HTTP_CODE:/d')
    
    if [ "$create_http_code" -ne 201 ]; then
        echo -e "${RED}✗ Error al crear item: HTTP $create_http_code${NC}"
        echo "$create_body"
        return 1
    fi
    
    local item_id=$(echo "$create_body" | grep -o '"id":"[^"]*' | cut -d'"' -f4)
    echo -e "${GREEN}✓ Item creado: $item_id${NC}"
    
    # Esperar procesamiento
    wait_for_event_processing 5
    
    # Verificar en Query Service (debe estar en Redis)
    echo -e "${YELLOW}Verificando item en Query Service (Redis)...${NC}"
    if verify_item_in_cache "$item_id" "$quantity" "$query_token"; then
        echo -e "${GREEN}✓ Item encontrado en Redis con cantidad correcta: $quantity${NC}"
        return 0
    else
        return 1
    fi
}

# Test 2: Ajustar stock y verificar actualización en Redis
test_adjust_stock_redis_update() {
    local test_name="Ajustar Stock y Verificar Actualización en Redis"
    echo -e "${BLUE}--- Ejecutando prueba: $test_name ---${NC}"
    
    # Obtener tokens para ambos servicios
    local cmd_token=$(get_jwt_token "$COMMAND_SERVICE")
    local query_token=$(get_jwt_token "$QUERY_SERVICE")
    if [ -z "$cmd_token" ] || [ -z "$query_token" ]; then
        echo -e "${RED}✗ No se pudo obtener token JWT${NC}"
        return 1
    fi
    
    # Crear item primero
    local sku="SKU-ADJUST-TEST-$(date +%s)"
    local name="Item Adjust Test"
    local initial_quantity=100
    
    echo -e "${YELLOW}Creando item en Command Service...${NC}"
    local create_response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/inventory/items" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $cmd_token" \
        -d "{\"sku\": \"$sku\", \"name\": \"$name\", \"description\": \"Item para prueba de ajuste\", \"quantity\": $initial_quantity}")
    
    local create_http_code=$(echo "$create_response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    local create_body=$(echo "$create_response" | sed '/HTTP_CODE:/d')
    
    if [ "$create_http_code" -ne 201 ]; then
        echo -e "${RED}✗ Error al crear item: HTTP $create_http_code${NC}"
        return 1
    fi
    
    local item_id=$(echo "$create_body" | grep -o '"id":"[^"]*' | cut -d'"' -f4)
    echo -e "${GREEN}✓ Item creado: $item_id${NC}"
    
    # Esperar procesamiento inicial
    wait_for_event_processing 5
    
    # Ajustar stock (+50)
    local adjustment=50
    local expected_quantity=$((initial_quantity + adjustment))
    
    echo -e "${YELLOW}Ajustando stock (+$adjustment)...${NC}"
    local adjust_response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/inventory/items/$item_id/adjust" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $cmd_token" \
        -d "{\"quantity\": $adjustment}")
    
    local adjust_http_code=$(echo "$adjust_response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    
    if [ "$adjust_http_code" -ne 200 ]; then
        echo -e "${RED}✗ Error al ajustar stock: HTTP $adjust_http_code${NC}"
        return 1
    fi
    
    echo -e "${GREEN}✓ Stock ajustado${NC}"
    
    # Esperar procesamiento
    wait_for_event_processing 5
    
    # Verificar en Query Service (debe estar actualizado en Redis)
    echo -e "${YELLOW}Verificando stock actualizado en Query Service (Redis)...${NC}"
    if verify_item_in_cache "$item_id" "$expected_quantity" "$query_token"; then
        echo -e "${GREEN}✓ Stock actualizado en Redis: $expected_quantity${NC}"
        return 0
    else
        return 1
    fi
}

# Test 3: Reservar stock y verificar actualización en Redis
test_reserve_stock_redis_update() {
    local test_name="Reservar Stock y Verificar Actualización en Redis"
    echo -e "${BLUE}--- Ejecutando prueba: $test_name ---${NC}"
    
    # Obtener tokens para ambos servicios
    local cmd_token=$(get_jwt_token "$COMMAND_SERVICE")
    local query_token=$(get_jwt_token "$QUERY_SERVICE")
    if [ -z "$cmd_token" ] || [ -z "$query_token" ]; then
        echo -e "${RED}✗ No se pudo obtener token JWT${NC}"
        return 1
    fi
    
    # Crear item primero
    local sku="SKU-RESERVE-TEST-$(date +%s)"
    local name="Item Reserve Test"
    local initial_quantity=100
    
    echo -e "${YELLOW}Creando item en Command Service...${NC}"
    local create_response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/inventory/items" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $cmd_token" \
        -d "{\"sku\": \"$sku\", \"name\": \"$name\", \"description\": \"Item para prueba de reserva\", \"quantity\": $initial_quantity}")
    
    local create_http_code=$(echo "$create_response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    local create_body=$(echo "$create_response" | sed '/HTTP_CODE:/d')
    
    if [ "$create_http_code" -ne 201 ]; then
        echo -e "${RED}✗ Error al crear item: HTTP $create_http_code${NC}"
        return 1
    fi
    
    local item_id=$(echo "$create_body" | grep -o '"id":"[^"]*' | cut -d'"' -f4)
    echo -e "${GREEN}✓ Item creado: $item_id${NC}"
    
    # Esperar procesamiento inicial
    wait_for_event_processing 5
    
    # Reservar stock (30 unidades)
    local reserve_quantity=30
    local expected_reserved=$reserve_quantity
    local expected_available=$((initial_quantity - reserve_quantity))
    
    echo -e "${YELLOW}Reservando stock ($reserve_quantity unidades)...${NC}"
    local reserve_response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/inventory/items/$item_id/reserve" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $cmd_token" \
        -d "{\"quantity\": $reserve_quantity}")
    
    local reserve_http_code=$(echo "$reserve_response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    
    if [ "$reserve_http_code" -ne 200 ]; then
        echo -e "${RED}✗ Error al reservar stock: HTTP $reserve_http_code${NC}"
        return 1
    fi
    
    echo -e "${GREEN}✓ Stock reservado${NC}"
    
    # Esperar procesamiento
    wait_for_event_processing 5
    
    # Verificar en Query Service (debe estar actualizado en Redis)
    echo -e "${YELLOW}Verificando reserva en Query Service (Redis)...${NC}"
    local query_response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X GET "$QUERY_SERVICE/api/v1/inventory/items/$item_id/stock" \
        -H "Authorization: Bearer $query_token" \
        -H "Content-Type: application/json")
    
    local query_http_code=$(echo "$query_response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    local query_body=$(echo "$query_response" | sed '/HTTP_CODE:/d')
    
    if [ "$query_http_code" -eq 200 ]; then
        local reserved=$(echo "$query_body" | grep -o '"reserved":[0-9]*' | cut -d':' -f2)
        local available=$(echo "$query_body" | grep -o '"available":[0-9]*' | cut -d':' -f2)
        
        if [ "$reserved" == "$expected_reserved" ] && [ "$available" == "$expected_available" ]; then
            echo -e "${GREEN}✓ Reserva actualizada en Redis: reserved=$reserved, available=$available${NC}"
            return 0
        else
            echo -e "${RED}✗ Valores incorrectos en Redis: reserved=$reserved (esperado $expected_reserved), available=$available (esperado $expected_available)${NC}"
            return 1
        fi
    else
        echo -e "${RED}✗ Error al consultar stock: HTTP $query_http_code${NC}"
        return 1
    fi
}

# Test 4: Liberar stock y verificar actualización en Redis
test_release_stock_redis_update() {
    local test_name="Liberar Stock y Verificar Actualización en Redis"
    echo -e "${BLUE}--- Ejecutando prueba: $test_name ---${NC}"
    
    # Obtener tokens para ambos servicios
    local cmd_token=$(get_jwt_token "$COMMAND_SERVICE")
    local query_token=$(get_jwt_token "$QUERY_SERVICE")
    if [ -z "$cmd_token" ] || [ -z "$query_token" ]; then
        echo -e "${RED}✗ No se pudo obtener token JWT${NC}"
        return 1
    fi
    
    # Crear item primero
    local sku="SKU-RELEASE-TEST-$(date +%s)"
    local name="Item Release Test"
    local initial_quantity=100
    
    echo -e "${YELLOW}Creando item en Command Service...${NC}"
    local create_response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/inventory/items" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $cmd_token" \
        -d "{\"sku\": \"$sku\", \"name\": \"$name\", \"description\": \"Item para prueba de liberación\", \"quantity\": $initial_quantity}")
    
    local create_http_code=$(echo "$create_response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    local create_body=$(echo "$create_response" | sed '/HTTP_CODE:/d')
    
    if [ "$create_http_code" -ne 201 ]; then
        echo -e "${RED}✗ Error al crear item: HTTP $create_http_code${NC}"
        return 1
    fi
    
    local item_id=$(echo "$create_body" | grep -o '"id":"[^"]*' | cut -d'"' -f4)
    echo -e "${GREEN}✓ Item creado: $item_id${NC}"
    
    # Esperar procesamiento inicial
    wait_for_event_processing 5
    
    # Reservar stock primero (50 unidades)
    local reserve_quantity=50
    echo -e "${YELLOW}Reservando stock ($reserve_quantity unidades)...${NC}"
    local reserve_response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/inventory/items/$item_id/reserve" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $cmd_token" \
        -d "{\"quantity\": $reserve_quantity}")
    
    local reserve_http_code=$(echo "$reserve_response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    
    if [ "$reserve_http_code" -ne 200 ]; then
        echo -e "${RED}✗ Error al reservar stock: HTTP $reserve_http_code${NC}"
        return 1
    fi
    
    wait_for_event_processing 5
    
    # Liberar stock (20 unidades)
    local release_quantity=20
    local expected_reserved=$((reserve_quantity - release_quantity))
    local expected_available=$((initial_quantity - expected_reserved))
    
    echo -e "${YELLOW}Liberando stock ($release_quantity unidades)...${NC}"
    local release_response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/inventory/items/$item_id/release" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $cmd_token" \
        -d "{\"quantity\": $release_quantity}")
    
    local release_http_code=$(echo "$release_response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    
    if [ "$release_http_code" -ne 200 ]; then
        echo -e "${RED}✗ Error al liberar stock: HTTP $release_http_code${NC}"
        return 1
    fi
    
    echo -e "${GREEN}✓ Stock liberado${NC}"
    
    # Esperar procesamiento
    wait_for_event_processing 5
    
    # Verificar en Query Service (debe estar actualizado en Redis)
    echo -e "${YELLOW}Verificando liberación en Query Service (Redis)...${NC}"
    local query_response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X GET "$QUERY_SERVICE/api/v1/inventory/items/$item_id/stock" \
        -H "Authorization: Bearer $query_token" \
        -H "Content-Type: application/json")
    
    local query_http_code=$(echo "$query_response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    local query_body=$(echo "$query_response" | sed '/HTTP_CODE:/d')
    
    if [ "$query_http_code" -eq 200 ]; then
        local reserved=$(echo "$query_body" | grep -o '"reserved":[0-9]*' | cut -d':' -f2)
        local available=$(echo "$query_body" | grep -o '"available":[0-9]*' | cut -d':' -f2)
        
        if [ "$reserved" == "$expected_reserved" ] && [ "$available" == "$expected_available" ]; then
            echo -e "${GREEN}✓ Liberación actualizada en Redis: reserved=$reserved, available=$available${NC}"
            return 0
        else
            echo -e "${RED}✗ Valores incorrectos en Redis: reserved=$reserved (esperado $expected_reserved), available=$available (esperado $expected_available)${NC}"
            return 1
        fi
    else
        echo -e "${RED}✗ Error al consultar stock: HTTP $query_http_code${NC}"
        return 1
    fi
}

# Test 5: Verificar que Query Service responde desde Redis (no bloquea actualizaciones)
test_query_from_redis_non_blocking() {
    local test_name="Query Service Responde desde Redis (No Bloquea Actualizaciones)"
    echo -e "${BLUE}--- Ejecutando prueba: $test_name ---${NC}"
    
    # Obtener tokens para ambos servicios
    local cmd_token=$(get_jwt_token "$COMMAND_SERVICE")
    local query_token=$(get_jwt_token "$QUERY_SERVICE")
    if [ -z "$cmd_token" ] || [ -z "$query_token" ]; then
        echo -e "${RED}✗ No se pudo obtener token JWT${NC}"
        return 1
    fi
    
    # Crear item
    local sku="SKU-NONBLOCK-TEST-$(date +%s)"
    local name="Item Non-Blocking Test"
    local initial_quantity=200
    
    echo -e "${YELLOW}Creando item en Command Service...${NC}"
    local create_response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/inventory/items" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $cmd_token" \
        -d "{\"sku\": \"$sku\", \"name\": \"$name\", \"description\": \"Item para prueba de no bloqueo\", \"quantity\": $initial_quantity}")
    
    local create_http_code=$(echo "$create_response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    local create_body=$(echo "$create_response" | sed '/HTTP_CODE:/d')
    
    if [ "$create_http_code" -ne 201 ]; then
        echo -e "${RED}✗ Error al crear item: HTTP $create_http_code${NC}"
        return 1
    fi
    
    local item_id=$(echo "$create_body" | grep -o '"id":"[^"]*' | cut -d'"' -f4)
    echo -e "${GREEN}✓ Item creado: $item_id${NC}"
    
    # Esperar procesamiento inicial
    wait_for_event_processing 5
    
    # Realizar múltiples consultas rápidas (deben responder desde Redis)
    echo -e "${YELLOW}Realizando múltiples consultas rápidas (deben responder desde Redis)...${NC}"
    local query_count=5
    local all_success=true
    
    for i in $(seq 1 $query_count); do
        local query_response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X GET "$QUERY_SERVICE/api/v1/inventory/items/$item_id" \
            -H "Authorization: Bearer $query_token" \
            -H "Content-Type: application/json")
        
        local query_http_code=$(echo "$query_response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
        
        if [ "$query_http_code" -ne 200 ]; then
            echo -e "${RED}✗ Consulta $i falló: HTTP $query_http_code${NC}"
            all_success=false
        fi
    done
    
    if [ "$all_success" == true ]; then
        echo -e "${GREEN}✓ Todas las consultas respondieron correctamente desde Redis${NC}"
        
        # Mientras tanto, ajustar stock (actualización asíncrona)
        echo -e "${YELLOW}Ajustando stock mientras se consulta (actualización asíncrona)...${NC}"
        local adjust_response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/inventory/items/$item_id/adjust" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $cmd_token" \
            -d "{\"quantity\": 25}")
        
        local adjust_http_code=$(echo "$adjust_response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
        
        if [ "$adjust_http_code" -eq 200 ]; then
            echo -e "${GREEN}✓ Stock ajustado (actualización asíncrona iniciada)${NC}"
            
            # Esperar procesamiento
            wait_for_event_processing 5
            
            # Verificar que la actualización se reflejó
            local expected_quantity=$((initial_quantity + 25))
            if verify_item_in_cache "$item_id" "$expected_quantity" "$query_token"; then
                echo -e "${GREEN}✓ Actualización asíncrona completada: cantidad actualizada a $expected_quantity${NC}"
                return 0
            else
                echo -e "${RED}✗ Actualización asíncrona no se reflejó en Redis${NC}"
                return 1
            fi
        else
            echo -e "${RED}✗ Error al ajustar stock: HTTP $adjust_http_code${NC}"
            return 1
        fi
    else
        return 1
    fi
}

# Función principal
main() {
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}Pruebas de Actualización de Redis${NC}"
    echo -e "${GREEN}Query Service - Listener Service${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    
    # Verificar servicios
    echo -e "${YELLOW}Verificando servicios...${NC}"
    
    if ! curl -s -f "$COMMAND_SERVICE/api/v1/health" > /dev/null; then
        echo -e "${RED}✗ Command Service no está disponible en $COMMAND_SERVICE${NC}"
        exit 1
    fi
    
    if ! curl -s -f "$QUERY_SERVICE/api/v1/health" > /dev/null; then
        echo -e "${RED}✗ Query Service no está disponible en $QUERY_SERVICE${NC}"
        exit 1
    fi
    
    if ! curl -s -f "$LISTENER_SERVICE/api/v1/health" > /dev/null; then
        echo -e "${YELLOW}⚠ Listener Service no está disponible en $LISTENER_SERVICE (puede estar corriendo solo el listener)${NC}"
    fi
    
    echo -e "${GREEN}✓ Servicios verificados${NC}"
    echo ""
    
    # Ejecutar pruebas
    echo -e "${BLUE}=== Ejecutando Pruebas ===${NC}"
    echo ""
    
    # Test 1: Crear item
    ((TOTAL_TESTS++))
    if test_create_item_redis_update; then
        ((PASSED_TESTS++))
        echo -e "${GREEN}✓ Test 1: PASÓ${NC}"
    else
        ((FAILED_TESTS++))
        echo -e "${RED}✗ Test 1: FALLÓ${NC}"
    fi
    echo ""
    
    # Test 2: Ajustar stock
    ((TOTAL_TESTS++))
    if test_adjust_stock_redis_update; then
        ((PASSED_TESTS++))
        echo -e "${GREEN}✓ Test 2: PASÓ${NC}"
    else
        ((FAILED_TESTS++))
        echo -e "${RED}✗ Test 2: FALLÓ${NC}"
    fi
    echo ""
    
    # Test 3: Reservar stock
    ((TOTAL_TESTS++))
    if test_reserve_stock_redis_update; then
        ((PASSED_TESTS++))
        echo -e "${GREEN}✓ Test 3: PASÓ${NC}"
    else
        ((FAILED_TESTS++))
        echo -e "${RED}✗ Test 3: FALLÓ${NC}"
    fi
    echo ""
    
    # Test 4: Liberar stock
    ((TOTAL_TESTS++))
    if test_release_stock_redis_update; then
        ((PASSED_TESTS++))
        echo -e "${GREEN}✓ Test 4: PASÓ${NC}"
    else
        ((FAILED_TESTS++))
        echo -e "${RED}✗ Test 4: FALLÓ${NC}"
    fi
    echo ""
    
    # Test 5: Query desde Redis (no bloquea)
    ((TOTAL_TESTS++))
    if test_query_from_redis_non_blocking; then
        ((PASSED_TESTS++))
        echo -e "${GREEN}✓ Test 5: PASÓ${NC}"
    else
        ((FAILED_TESTS++))
        echo -e "${RED}✗ Test 5: FALLÓ${NC}"
    fi
    echo ""
    
    # Resumen
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}Resumen de Pruebas${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo -e "Total: $TOTAL_TESTS"
    echo -e "${GREEN}Exitosas: $PASSED_TESTS${NC}"
    echo -e "${RED}Fallidas: $FAILED_TESTS${NC}"
    echo ""
    
    if [ $FAILED_TESTS -eq 0 ]; then
        echo -e "${GREEN}✅ Todas las pruebas pasaron${NC}"
        exit 0
    else
        echo -e "${RED}❌ Algunas pruebas fallaron${NC}"
        exit 1
    fi
}

# Ejecutar función principal
main
