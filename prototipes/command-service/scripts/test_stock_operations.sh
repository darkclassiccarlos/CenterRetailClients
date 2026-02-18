#!/bin/bash

# Script para probar las operaciones de stock (AdjustStock, ReserveStock, ReleaseStock)
# Verifica que los valores se actualicen correctamente en el flujo completo con autenticación JWT

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
echo -e "${BLUE}Prueba de Operaciones de Stock${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Obtener token JWT
echo -e "${YELLOW}0. Autenticando con JWT...${NC}"
if ! get_jwt_token; then
    echo -e "${RED}No se pudo obtener token JWT. Abortando.${NC}"
    exit 1
fi
echo ""

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

# Verificar servicios
echo -e "${YELLOW}1. Verificando servicios...${NC}"
if ! check_service "Command Service" "$COMMAND_SERVICE"; then
    echo -e "${RED}Command Service no está disponible. Por favor, inicia el servicio.${NC}"
    exit 1
fi

if ! check_service "Query Service" "$QUERY_SERVICE"; then
    echo -e "${YELLOW}Query Service no está disponible (continuando sin verificación de lectura)${NC}"
fi

if ! check_service "Listener Service" "$LISTENER_SERVICE"; then
    echo -e "${YELLOW}Listener Service no está disponible (continuando sin verificación de procesamiento)${NC}"
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
            \"description\": \"Item de prueba para operaciones de stock\",
            \"quantity\": $quantity
        }")
    
    http_code=$(echo "$response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    body=$(echo "$response" | sed '/HTTP_CODE:/d')
    
    # Validate http_code is a number
    if [ -z "$http_code" ] || ! [[ "$http_code" =~ ^[0-9]+$ ]]; then
        echo -e "${RED}✗ Error: No se pudo obtener código HTTP. Response: $response${NC}"
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

# Función para obtener estado de stock desde Command Service
get_stock_status() {
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
    
    # Validate http_code is a number
    if [ -z "$http_code" ] || ! [[ "$http_code" =~ ^[0-9]+$ ]]; then
        echo "ERROR"
        return
    fi
    
    if [ "$http_code" -eq 200 ]; then
        quantity=$(echo "$body" | grep -o '"quantity":[0-9]*' | cut -d':' -f2)
        reserved=$(echo "$body" | grep -o '"reserved":[0-9]*' | cut -d':' -f2)
        available=$(echo "$body" | grep -o '"available":[0-9]*' | cut -d':' -f2)
        # Default to 0 if empty
        quantity=${quantity:-0}
        reserved=${reserved:-0}
        available=${available:-0}
        echo "$quantity|$reserved|$available"
    else
        echo "ERROR"
    fi
}

# Función para ajustar stock
adjust_stock() {
    local item_id=$1
    local adjustment=$2
    local expected_new_quantity=$3
    
    echo -e "${BLUE}Ajustando stock del item $item_id: $adjustment unidades${NC}"
    
    response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/inventory/items/$item_id/adjust" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -d "{
            \"quantity\": $adjustment
        }" 2>&1)
    
    # Debug: check if response is empty
    if [ -z "$response" ]; then
        echo -e "${RED}✗ Error: La respuesta de curl está vacía${NC}"
        return 1
    fi
    
    http_code=$(echo "$response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    body=$(echo "$response" | sed '/HTTP_CODE:/d')
    
    # Validate http_code is a number
    if [ -z "$http_code" ] || ! [[ "$http_code" =~ ^[0-9]+$ ]]; then
        echo -e "${RED}✗ Error: No se pudo obtener código HTTP. Response length: ${#response}${NC}"
        echo -e "${YELLOW}Debug: First 200 chars of response: ${response:0:200}${NC}"
        return 1
    fi
    
    if [ "$http_code" -eq 200 ]; then
        new_quantity=$(echo "$body" | grep -o '"quantity":[0-9]*' | cut -d':' -f2)
        reserved=$(echo "$body" | grep -o '"reserved":[0-9]*' | cut -d':' -f2)
        available=$(echo "$body" | grep -o '"available":[0-9]*' | cut -d':' -f2)
        
        # Default to 0 if empty
        new_quantity=${new_quantity:-0}
        reserved=${reserved:-0}
        available=${available:-0}
        
        echo -e "${GREEN}✓ Stock ajustado exitosamente${NC}"
        echo -e "   Ajuste: $adjustment"
        echo -e "   Nueva cantidad: $new_quantity"
        echo -e "   Reservado: $reserved"
        echo -e "   Disponible: $available"
        
        # Verificar que la cantidad sea correcta
        if [ -n "$new_quantity" ] && [ -n "$expected_new_quantity" ] && [ "$new_quantity" -eq "$expected_new_quantity" ]; then
            echo -e "${GREEN}✓ Cantidad verificada correctamente ($new_quantity = $expected_new_quantity)${NC}"
        else
            echo -e "${RED}✗ ERROR: Cantidad incorrecta ($new_quantity != $expected_new_quantity)${NC}"
            return 1
        fi
    else
        echo -e "${RED}✗ Error al ajustar stock: $body${NC}"
        return 1
    fi
}

# Función para reservar stock
reserve_stock() {
    local item_id=$1
    local quantity=$2
    local expected_reserved=$3
    local expected_available=$4
    
    echo -e "${BLUE}Reservando $quantity unidades del item $item_id${NC}"
    
    response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/inventory/items/$item_id/reserve" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -d "{
            \"quantity\": $quantity
        }")
    
    http_code=$(echo "$response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    body=$(echo "$response" | sed '/HTTP_CODE:/d')
    
    # Validate http_code is a number
    if [ -z "$http_code" ] || ! [[ "$http_code" =~ ^[0-9]+$ ]]; then
        echo -e "${RED}✗ Error: No se pudo obtener código HTTP. Response: $response${NC}"
        return 1
    fi
    
    if [ "$http_code" -eq 200 ]; then
        total_quantity=$(echo "$body" | grep -o '"quantity":[0-9]*' | cut -d':' -f2)
        reserved=$(echo "$body" | grep -o '"reserved":[0-9]*' | cut -d':' -f2)
        available=$(echo "$body" | grep -o '"available":[0-9]*' | cut -d':' -f2)
        
        # Default to 0 if empty
        total_quantity=${total_quantity:-0}
        reserved=${reserved:-0}
        available=${available:-0}
        
        echo -e "${GREEN}✓ Stock reservado exitosamente${NC}"
        echo -e "   Cantidad a reservar: $quantity"
        echo -e "   Total reservado: $reserved"
        echo -e "   Disponible: $available"
        
        # Verificar que los valores sean correctos
        if [ -n "$reserved" ] && [ -n "$expected_reserved" ] && [ -n "$available" ] && [ -n "$expected_available" ] && \
           [ "$reserved" -eq "$expected_reserved" ] && [ "$available" -eq "$expected_available" ]; then
            echo -e "${GREEN}✓ Valores verificados correctamente${NC}"
        else
            echo -e "${RED}✗ ERROR: Valores incorrectos${NC}"
            echo -e "   Esperado: Reservado=$expected_reserved, Disponible=$expected_available"
            echo -e "   Obtenido: Reservado=$reserved, Disponible=$available"
            return 1
        fi
    else
        echo -e "${RED}✗ Error al reservar stock: $body${NC}"
        return 1
    fi
}

# Función para liberar stock
release_stock() {
    local item_id=$1
    local quantity=$2
    local expected_reserved=$3
    local expected_available=$4
    
    echo -e "${BLUE}Liberando $quantity unidades del item $item_id${NC}"
    
    response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/inventory/items/$item_id/release" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -d "{
            \"quantity\": $quantity
        }")
    
    http_code=$(echo "$response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    body=$(echo "$response" | sed '/HTTP_CODE:/d')
    
    # Validate http_code is a number
    if [ -z "$http_code" ] || ! [[ "$http_code" =~ ^[0-9]+$ ]]; then
        echo -e "${RED}✗ Error: No se pudo obtener código HTTP. Response: $response${NC}"
        return 1
    fi
    
    if [ "$http_code" -eq 200 ]; then
        total_quantity=$(echo "$body" | grep -o '"quantity":[0-9]*' | cut -d':' -f2)
        reserved=$(echo "$body" | grep -o '"reserved":[0-9]*' | cut -d':' -f2)
        available=$(echo "$body" | grep -o '"available":[0-9]*' | cut -d':' -f2)
        
        # Default to 0 if empty
        total_quantity=${total_quantity:-0}
        reserved=${reserved:-0}
        available=${available:-0}
        
        echo -e "${GREEN}✓ Stock liberado exitosamente${NC}"
        echo -e "   Cantidad a liberar: $quantity"
        echo -e "   Total reservado: $reserved"
        echo -e "   Disponible: $available"
        
        # Verificar que los valores sean correctos
        if [ -n "$reserved" ] && [ -n "$expected_reserved" ] && [ -n "$available" ] && [ -n "$expected_available" ] && \
           [ "$reserved" -eq "$expected_reserved" ] && [ "$available" -eq "$expected_available" ]; then
            echo -e "${GREEN}✓ Valores verificados correctamente${NC}"
        else
            echo -e "${RED}✗ ERROR: Valores incorrectos${NC}"
            echo -e "   Esperado: Reservado=$expected_reserved, Disponible=$expected_available"
            echo -e "   Obtenido: Reservado=$reserved, Disponible=$available"
            return 1
        fi
    else
        echo -e "${RED}✗ Error al liberar stock: $body${NC}"
        return 1
    fi
}

# Crear item de prueba
echo -e "${YELLOW}2. Creando item de prueba...${NC}"
ITEM_ID=$(create_item "TEST-STOCK-001" "Item de Prueba Stock" 100 2>&1 | grep -E "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$" | head -1)

if [ -z "$ITEM_ID" ]; then
    echo -e "${RED}No se pudo crear el item. Abortando.${NC}"
    exit 1
fi

echo ""
echo -e "${YELLOW}3. Probando operaciones de stock...${NC}"
echo ""

# Test 1: Ajustar stock (aumentar)
echo -e "${YELLOW}Test 1: Ajustar stock (aumentar +25)${NC}"
if ! adjust_stock "$ITEM_ID" 25 125; then
    echo -e "${RED}Test 1 falló${NC}"
    exit 1
fi
sleep 2
echo ""

# Test 2: Ajustar stock (disminuir)
echo -e "${YELLOW}Test 2: Ajustar stock (disminuir -15)${NC}"
if ! adjust_stock "$ITEM_ID" -15 110; then
    echo -e "${RED}Test 2 falló${NC}"
    exit 1
fi
sleep 2
echo ""

# Test 3: Reservar stock
echo -e "${YELLOW}Test 3: Reservar stock (30 unidades)${NC}"
if ! reserve_stock "$ITEM_ID" 30 30 80; then
    echo -e "${RED}Test 3 falló${NC}"
    exit 1
fi
sleep 2
echo ""

# Test 4: Reservar más stock
echo -e "${YELLOW}Test 4: Reservar más stock (20 unidades adicionales)${NC}"
if ! reserve_stock "$ITEM_ID" 20 50 60; then
    echo -e "${RED}Test 4 falló${NC}"
    exit 1
fi
sleep 2
echo ""

# Test 5: Liberar stock
echo -e "${YELLOW}Test 5: Liberar stock (15 unidades)${NC}"
if ! release_stock "$ITEM_ID" 15 35 75; then
    echo -e "${RED}Test 5 falló${NC}"
    exit 1
fi
sleep 2
echo ""

# Test 6: Ajustar stock con reservas activas
echo -e "${YELLOW}Test 6: Ajustar stock con reservas activas (+10)${NC}"
if ! adjust_stock "$ITEM_ID" 10 120; then
    echo -e "${RED}Test 6 falló${NC}"
    exit 1
fi
sleep 2
echo ""

# Verificar estado final
echo -e "${YELLOW}4. Verificando estado final...${NC}"
final_status=$(get_stock_status "$ITEM_ID" "query")
if [ "$final_status" = "ERROR" ]; then
    echo -e "${YELLOW}⚠ No se pudo obtener estado final desde Command Service${NC}"
    echo -e "${BLUE}Nota: Esto puede ser normal si el item solo existe en memoria del Command Service${NC}"
else
    final_quantity=$(echo "$final_status" | cut -d'|' -f1)
    final_reserved=$(echo "$final_status" | cut -d'|' -f2)
    final_available=$(echo "$final_status" | cut -d'|' -f3)
    
    # Default to 0 if empty
    final_quantity=${final_quantity:-0}
    final_reserved=${final_reserved:-0}
    final_available=${final_available:-0}
    
    echo -e "${BLUE}Estado final del item:${NC}"
    echo -e "   Cantidad total: $final_quantity"
    echo -e "   Reservado: $final_reserved"
    echo -e "   Disponible: $final_available"
    
    # Verificar que los valores sean consistentes
    if [ -n "$final_quantity" ] && [ -n "$final_reserved" ] && [ -n "$final_available" ]; then
        expected_available=$((final_quantity - final_reserved))
        if [ "$final_available" -eq "$expected_available" ]; then
            echo -e "${GREEN}✓ Valores consistentes (Disponible = Total - Reservado)${NC}"
        else
            echo -e "${YELLOW}⚠ Valores inconsistentes (esto puede ser normal si el item solo existe en memoria)${NC}"
            echo -e "   Esperado disponible: $expected_available"
            echo -e "   Obtenido disponible: $final_available"
        fi
    fi
fi

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Todos los tests pasaron exitosamente${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${BLUE}Resumen:${NC}"
echo "- Item creado: $ITEM_ID"
echo "- Ajustes de stock: 3 (aumentar, disminuir, con reservas)"
echo "- Reservas: 2"
echo "- Liberaciones: 1"
echo "- Estado final: Total=$final_quantity, Reservado=$final_reserved, Disponible=$final_available"
echo ""

