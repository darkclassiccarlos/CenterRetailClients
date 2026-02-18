#!/bin/bash

# Script específico para probar la liberación de stock reservado
# Verifica que ReleaseStock funcione correctamente con autenticación JWT

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

# Credenciales JWT
JWT_USERNAME="${JWT_USERNAME:-admin}"
JWT_PASSWORD="${JWT_PASSWORD:-admin123}"

# Variable para almacenar el token JWT
JWT_TOKEN=""

# Función para obtener token JWT
get_jwt_token() {
    echo -e "${BLUE}Obteniendo token JWT...${NC}" >&2
    
    response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/auth/login" \
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

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Prueba de Liberación de Stock Reservado${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Obtener token JWT
echo -e "${YELLOW}0. Autenticando con JWT...${NC}"
if ! get_jwt_token; then
    echo -e "${RED}No se pudo obtener token JWT. Abortando.${NC}"
    exit 1
fi
echo ""

# Función para crear un item
create_item() {
    local sku=$1
    local name=$2
    local quantity=$3
    
    echo -e "${BLUE}Creando item: $name (SKU: $sku, Cantidad inicial: $quantity)${NC}" >&2
    
    response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/inventory/items" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -d "{
            \"sku\": \"$sku\",
            \"name\": \"$name\",
            \"description\": \"Item de prueba para liberación de stock\",
            \"quantity\": $quantity
        }")
    
    http_code=$(echo "$response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    body=$(echo "$response" | sed '/HTTP_CODE:/d')
    
    if [ -z "$http_code" ] || ! [[ "$http_code" =~ ^[0-9]+$ ]]; then
        echo -e "${RED}✗ Error: No se pudo obtener código HTTP${NC}" >&2
        echo ""
        return
    fi
    
    if [ "$http_code" -eq 201 ]; then
        item_id=$(echo "$body" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
        echo -e "${GREEN}✓ Item creado exitosamente (ID: $item_id)${NC}" >&2
        echo "$item_id"
    else
        echo -e "${RED}✗ Error al crear item (HTTP $http_code): $body${NC}" >&2
        echo ""
    fi
}

# Función para obtener estado del item
get_item_status() {
    local item_id=$1
    local source=$2
    
    if [ "$source" = "command" ]; then
        url="$COMMAND_SERVICE/api/v1/inventory/items/$item_id"
    else
        url="$QUERY_SERVICE/api/v1/inventory/items/$item_id"
    fi
    
    response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X GET "$url")
    http_code=$(echo "$response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    body=$(echo "$response" | sed '/HTTP_CODE:/d')
    
    if [ -z "$http_code" ] || ! [[ "$http_code" =~ ^[0-9]+$ ]]; then
        echo "ERROR"
        return
    fi
    
    if [ "$http_code" -eq 200 ]; then
        quantity=$(echo "$body" | grep -o '"quantity":[0-9]*' | cut -d':' -f2)
        reserved=$(echo "$body" | grep -o '"reserved":[0-9]*' | cut -d':' -f2)
        available=$(echo "$body" | grep -o '"available":[0-9]*' | cut -d':' -f2)
        quantity=${quantity:-0}
        reserved=${reserved:-0}
        available=${available:-0}
        echo "$quantity|$reserved|$available"
    else
        echo "ERROR"
    fi
}

# Función para reservar stock
reserve_stock() {
    local item_id=$1
    local quantity=$2
    
    echo -e "${BLUE}Reservando $quantity unidades...${NC}"
    
    response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/inventory/items/$item_id/reserve" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -d "{
            \"quantity\": $quantity
        }" 2>&1)
    
    if [ -z "$response" ]; then
        echo -e "${RED}✗ Error: La respuesta de curl está vacía${NC}"
        return 1
    fi
    
    http_code=$(echo "$response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    body=$(echo "$response" | sed '/HTTP_CODE:/d')
    
    if [ -z "$http_code" ] || ! [[ "$http_code" =~ ^[0-9]+$ ]]; then
        echo -e "${RED}✗ Error: No se pudo obtener código HTTP${NC}"
        return 1
    fi
    
    if [ "$http_code" -eq 200 ]; then
        reserved=$(echo "$body" | grep -o '"reserved":[0-9]*' | cut -d':' -f2)
        available=$(echo "$body" | grep -o '"available":[0-9]*' | cut -d':' -f2)
        reserved=${reserved:-0}
        available=${available:-0}
        echo -e "${GREEN}✓ Stock reservado exitosamente${NC}"
        echo -e "   Reservado: $reserved"
        echo -e "   Disponible: $available"
        return 0
    else
        echo -e "${RED}✗ Error al reservar stock (HTTP $http_code): $body${NC}"
        return 1
    fi
}

# Función para liberar stock
release_stock() {
    local item_id=$1
    local quantity=$2
    local expected_reserved=$3
    local expected_available=$4
    
    echo -e "${BLUE}Liberando $quantity unidades de stock reservado...${NC}"
    
    response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/inventory/items/$item_id/release" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -d "{
            \"quantity\": $quantity
        }" 2>&1)
    
    if [ -z "$response" ]; then
        echo -e "${RED}✗ Error: La respuesta de curl está vacía${NC}"
        return 1
    fi
    
    http_code=$(echo "$response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    body=$(echo "$response" | sed '/HTTP_CODE:/d')
    
    if [ -z "$http_code" ] || ! [[ "$http_code" =~ ^[0-9]+$ ]]; then
        echo -e "${RED}✗ Error: No se pudo obtener código HTTP${NC}"
        return 1
    fi
    
    if [ "$http_code" -eq 200 ]; then
        total_quantity=$(echo "$body" | grep -o '"quantity":[0-9]*' | cut -d':' -f2)
        reserved=$(echo "$body" | grep -o '"reserved":[0-9]*' | cut -d':' -f2)
        available=$(echo "$body" | grep -o '"available":[0-9]*' | cut -d':' -f2)
        
        total_quantity=${total_quantity:-0}
        reserved=${reserved:-0}
        available=${available:-0}
        
        echo -e "${GREEN}✓ Stock liberado exitosamente${NC}"
        echo -e "   Cantidad total: $total_quantity"
        echo -e "   Reservado: $reserved"
        echo -e "   Disponible: $available"
        
        # Verificar que los valores sean correctos
        if [ -n "$reserved" ] && [ -n "$expected_reserved" ] && [ -n "$available" ] && [ -n "$expected_available" ] && \
           [ "$reserved" -eq "$expected_reserved" ] && [ "$available" -eq "$expected_available" ]; then
            echo -e "${GREEN}✓ Valores verificados correctamente${NC}"
            return 0
        else
            echo -e "${RED}✗ ERROR: Valores incorrectos${NC}"
            echo -e "   Esperado: Reservado=$expected_reserved, Disponible=$expected_available"
            echo -e "   Obtenido: Reservado=$reserved, Disponible=$available"
            return 1
        fi
    else
        echo -e "${RED}✗ Error al liberar stock (HTTP $http_code): $body${NC}"
        return 1
    fi
}

# Crear item de prueba
echo -e "${YELLOW}1. Creando item de prueba...${NC}"
ITEM_ID=$(create_item "TEST-RELEASE-001" "Item Test Release Stock" 100 2>&1 | grep -E "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$" | head -1)

if [ -z "$ITEM_ID" ]; then
    echo -e "${RED}No se pudo crear el item. Abortando.${NC}"
    exit 1
fi

echo -e "${GREEN}Item ID: $ITEM_ID${NC}"
echo ""

# Obtener estado inicial
echo -e "${YELLOW}2. Estado inicial del item:${NC}"
initial_status=$(get_item_status "$ITEM_ID" "query")
if [ "$initial_status" != "ERROR" ]; then
    initial_quantity=$(echo "$initial_status" | cut -d'|' -f1)
    initial_reserved=$(echo "$initial_status" | cut -d'|' -f2)
    initial_available=$(echo "$initial_status" | cut -d'|' -f3)
    echo -e "   Cantidad total: $initial_quantity"
    echo -e "   Reservado: $initial_reserved"
    echo -e "   Disponible: $initial_available"
else
    echo -e "${YELLOW}⚠ No se pudo obtener estado inicial desde Query Service${NC}"
    initial_quantity=100
    initial_reserved=0
    initial_available=100
fi
echo ""

# Test 1: Reservar stock inicial
echo -e "${YELLOW}3. Test 1: Reservar 60 unidades${NC}"
if ! reserve_stock "$ITEM_ID" 60; then
    echo -e "${RED}Test 1 falló${NC}"
    exit 1
fi
sleep 2
echo ""

# Verificar estado después de reservar
echo -e "${YELLOW}4. Estado después de reservar:${NC}"
after_reserve_status=$(get_item_status "$ITEM_ID" "query")
if [ "$after_reserve_status" != "ERROR" ]; then
    after_reserve_quantity=$(echo "$after_reserve_status" | cut -d'|' -f1)
    after_reserve_reserved=$(echo "$after_reserve_status" | cut -d'|' -f2)
    after_reserve_available=$(echo "$after_reserve_status" | cut -d'|' -f3)
    echo -e "   Cantidad total: $after_reserve_quantity"
    echo -e "   Reservado: $after_reserve_reserved"
    echo -e "   Disponible: $after_reserve_available"
    
    if [ "$after_reserve_reserved" -eq 60 ] && [ "$after_reserve_available" -eq 40 ]; then
        echo -e "${GREEN}✓ Estado verificado correctamente${NC}"
    else
        echo -e "${YELLOW}⚠ Estado no coincide exactamente (puede ser por timing)${NC}"
    fi
else
    echo -e "${YELLOW}⚠ No se pudo obtener estado desde Query Service${NC}"
fi
echo ""

# Test 2: Liberar parte del stock reservado
echo -e "${YELLOW}5. Test 2: Liberar 30 unidades de las 60 reservadas${NC}"
if ! release_stock "$ITEM_ID" 30 30 70; then
    echo -e "${RED}Test 2 falló${NC}"
    exit 1
fi
sleep 2
echo ""

# Verificar estado después de liberar
echo -e "${YELLOW}6. Estado después de liberar:${NC}"
after_release_status=$(get_item_status "$ITEM_ID" "query")
if [ "$after_release_status" != "ERROR" ]; then
    after_release_quantity=$(echo "$after_release_status" | cut -d'|' -f1)
    after_release_reserved=$(echo "$after_release_status" | cut -d'|' -f2)
    after_release_available=$(echo "$after_release_status" | cut -d'|' -f3)
    echo -e "   Cantidad total: $after_release_quantity"
    echo -e "   Reservado: $after_release_reserved"
    echo -e "   Disponible: $after_release_available"
    
    if [ "$after_release_reserved" -eq 30 ] && [ "$after_release_available" -eq 70 ]; then
        echo -e "${GREEN}✓ Estado verificado correctamente${NC}"
    else
        echo -e "${YELLOW}⚠ Estado no coincide exactamente (puede ser por timing)${NC}"
    fi
else
    echo -e "${YELLOW}⚠ No se pudo obtener estado desde Query Service${NC}"
fi
echo ""

# Test 3: Liberar todo el stock restante
echo -e "${YELLOW}7. Test 3: Liberar las 30 unidades restantes${NC}"
if ! release_stock "$ITEM_ID" 30 0 100; then
    echo -e "${RED}Test 3 falló${NC}"
    exit 1
fi
sleep 2
echo ""

# Verificar estado final
echo -e "${YELLOW}8. Estado final:${NC}"
final_status=$(get_item_status "$ITEM_ID" "query")
if [ "$final_status" != "ERROR" ]; then
    final_quantity=$(echo "$final_status" | cut -d'|' -f1)
    final_reserved=$(echo "$final_status" | cut -d'|' -f2)
    final_available=$(echo "$final_status" | cut -d'|' -f3)
    echo -e "   Cantidad total: $final_quantity"
    echo -e "   Reservado: $final_reserved"
    echo -e "   Disponible: $final_available"
    
    if [ "$final_reserved" -eq 0 ] && [ "$final_available" -eq 100 ]; then
        echo -e "${GREEN}✓ Estado final verificado correctamente${NC}"
    else
        echo -e "${YELLOW}⚠ Estado final no coincide exactamente (puede ser por timing)${NC}"
    fi
else
    echo -e "${YELLOW}⚠ No se pudo obtener estado final desde Query Service${NC}"
fi
echo ""

# Test 4: Intentar liberar más de lo reservado (debe fallar)
echo -e "${YELLOW}9. Test 4: Intentar liberar más de lo reservado (debe fallar)${NC}"
response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/inventory/items/$ITEM_ID/release" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -d '{"quantity": 10}' 2>&1)

http_code=$(echo "$response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
body=$(echo "$response" | sed '/HTTP_CODE:/d')

if [ "$http_code" -eq 400 ]; then
    echo -e "${GREEN}✓ Correctamente rechazado (HTTP 400)${NC}"
    echo -e "   Respuesta: $body"
else
    echo -e "${RED}✗ ERROR: Debería haber retornado 400, pero retornó $http_code${NC}"
    echo -e "   Respuesta: $body"
    exit 1
fi
echo ""

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Todos los tests de liberación pasaron${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${BLUE}Resumen:${NC}"
echo "- Item creado: $ITEM_ID"
echo "- Reservas: 1 (60 unidades)"
echo "- Liberaciones: 2 (30 + 30 unidades)"
echo "- Validación de error: 1 (intentar liberar más de lo reservado)"
echo ""

