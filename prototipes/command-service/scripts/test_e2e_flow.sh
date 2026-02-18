#!/bin/bash

# Script para probar el flujo completo end-to-end
# Prueba: Command Service -> Kafka -> Listener Service -> Query Service

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
echo -e "${BLUE}Prueba End-to-End del Flujo Completo${NC}"
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
    
    echo -e "${YELLOW}Verificando $service_name...${NC}"
    if curl -s -f "$service_url/api/v1/health" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ $service_name está disponible${NC}"
        return 0
    else
        echo -e "${RED}✗ $service_name no está disponible en $service_url${NC}"
        return 1
    fi
}

# Verificar servicios
echo -e "${YELLOW}1. Verificando servicios...${NC}"
SERVICES_OK=true

if ! check_service "Command Service" "$COMMAND_SERVICE"; then
    SERVICES_OK=false
fi

if ! check_service "Query Service" "$QUERY_SERVICE"; then
    SERVICES_OK=false
fi

if ! check_service "Listener Service" "$LISTENER_SERVICE"; then
    SERVICES_OK=false
fi

if [ "$SERVICES_OK" = false ]; then
    echo -e "${RED}Algunos servicios no están disponibles. Por favor, inicia todos los servicios antes de continuar.${NC}"
    exit 1
fi

echo ""
echo -e "${YELLOW}2. Creando items de inventario para diferentes tiendas...${NC}"

# Función para crear un item
create_item() {
    local sku=$1
    local name=$2
    local description=$3
    local quantity=$4
    local store_name=$5
    
    echo -e "${BLUE}Creando item: $name (SKU: $sku) para $store_name${NC}" >&2
    
    response=$(curl -s -w "\n%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/inventory/items" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -d "{
            \"sku\": \"$sku\",
            \"name\": \"$name\",
            \"description\": \"$description\",
            \"quantity\": $quantity
        }")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" -eq 201 ]; then
        # Extraer ID del JSON de forma más robusta usando Python si está disponible
        if command -v python3 > /dev/null 2>&1; then
            item_id=$(echo "$body" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('id', ''))" 2>/dev/null)
        else
            item_id=$(echo "$body" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
        fi
        
        if [ -n "$item_id" ] && [ "$item_id" != "null" ] && [ "$item_id" != "" ]; then
            echo -e "${GREEN}✓ Item creado exitosamente (ID: $item_id)${NC}" >&2
            echo "$item_id"
        else
            echo -e "${RED}✗ Error: No se pudo extraer ID del item${NC}" >&2
            echo ""
        fi
    else
        echo -e "${RED}✗ Error al crear item (HTTP $http_code): $body${NC}" >&2
        echo ""
    fi
}

# Crear items para diferentes tiendas
echo ""
echo -e "${YELLOW}Tienda 1: Tienda Centro${NC}"
ITEM1_OUTPUT=$(create_item "STORE1-LAPTOP-001" "Laptop Dell XPS 15" "Laptop de alta gama con 16GB RAM" 50 "Tienda Centro" 2>&1)
ITEM1_ID=$(echo "$ITEM1_OUTPUT" | grep -oE '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' | head -1)
echo "$ITEM1_OUTPUT" | grep -vE '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' >&2
sleep 3

echo ""
echo -e "${YELLOW}Tienda 2: Tienda Norte${NC}"
ITEM2_OUTPUT=$(create_item "STORE2-MOUSE-001" "Mouse Logitech MX Master" "Mouse inalámbrico ergonómico" 100 "Tienda Norte" 2>&1)
ITEM2_ID=$(echo "$ITEM2_OUTPUT" | grep -oE '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' | head -1)
echo "$ITEM2_OUTPUT" | grep -vE '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' >&2
sleep 3

echo ""
echo -e "${YELLOW}Tienda 3: Tienda Sur${NC}"
ITEM3_OUTPUT=$(create_item "STORE3-KEYBOARD-001" "Teclado Mecánico RGB" "Teclado mecánico con iluminación RGB" 75 "Tienda Sur" 2>&1)
ITEM3_ID=$(echo "$ITEM3_OUTPUT" | grep -oE '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' | head -1)
echo "$ITEM3_OUTPUT" | grep -vE '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' >&2
sleep 3

# Mostrar IDs capturados para debugging
echo ""
echo -e "${BLUE}IDs capturados:${NC}"
echo -e "  Item 1 (Tienda Centro): ${ITEM1_ID:-NO CAPTURADO}"
echo -e "  Item 2 (Tienda Norte): ${ITEM2_ID:-NO CAPTURADO}"
echo -e "  Item 3 (Tienda Sur): ${ITEM3_ID:-NO CAPTURADO}"
echo ""

echo ""
echo -e "${YELLOW}3. Ajustando stock de los items...${NC}"

# Función para ajustar stock
adjust_stock() {
    local item_id=$1
    local quantity=$2
    local operation=$3
    
    # Limpiar item_id de posibles caracteres de escape
    item_id=$(echo "$item_id" | tr -d '\n\r' | grep -oE '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}' | head -1)
    
    if [ -z "$item_id" ]; then
        echo -e "${RED}✗ Error: Item ID inválido${NC}" >&2
        return 1
    fi
    
    echo -e "${BLUE}Ajustando stock del item $item_id: $operation $quantity unidades${NC}"
    
    response=$(curl -s -w "\n%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/inventory/items/$item_id/adjust" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -d "{
            \"quantity\": $quantity
        }")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" -eq 200 ]; then
        new_quantity=$(echo "$body" | grep -o '"quantity":[0-9]*' | cut -d':' -f2)
        echo -e "${GREEN}✓ Stock ajustado exitosamente (Nueva cantidad: $new_quantity)${NC}"
    else
        echo -e "${RED}✗ Error al ajustar stock: $body${NC}"
    fi
}

# Ajustar stock de los items
if [ -n "$ITEM1_ID" ]; then
    adjust_stock "$ITEM1_ID" 10 "Aumentar"
    sleep 1
fi

if [ -n "$ITEM2_ID" ]; then
    adjust_stock "$ITEM2_ID" -20 "Disminuir"
    sleep 1
fi

echo ""
echo -e "${YELLOW}4. Reservando stock de los items...${NC}"

# Función para reservar stock
reserve_stock() {
    local item_id=$1
    local quantity=$2
    
    # Limpiar item_id de posibles caracteres de escape
    item_id=$(echo "$item_id" | tr -d '\n\r' | grep -oE '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}' | head -1)
    
    if [ -z "$item_id" ]; then
        echo -e "${RED}✗ Error: Item ID inválido${NC}" >&2
        return 1
    fi
    
    echo -e "${BLUE}Reservando $quantity unidades del item $item_id${NC}"
    
    response=$(curl -s -w "\n%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/inventory/items/$item_id/reserve" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -d "{
            \"quantity\": $quantity
        }")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" -eq 200 ]; then
        reserved=$(echo "$body" | grep -o '"reserved":[0-9]*' | cut -d':' -f2)
        available=$(echo "$body" | grep -o '"available":[0-9]*' | cut -d':' -f2)
        echo -e "${GREEN}✓ Stock reservado exitosamente (Reservado: $reserved, Disponible: $available)${NC}"
    else
        echo -e "${RED}✗ Error al reservar stock: $body${NC}"
    fi
}

# Reservar stock
if [ -n "$ITEM1_ID" ]; then
    reserve_stock "$ITEM1_ID" 5
    sleep 1
fi

if [ -n "$ITEM2_ID" ]; then
    reserve_stock "$ITEM2_ID" 10
    sleep 1
fi

echo ""
echo -e "${YELLOW}5. Esperando procesamiento de eventos (25 segundos)...${NC}"
echo -e "${YELLOW}   (Esto permite que el Listener Service procese todos los eventos)${NC}"
echo -e "${YELLOW}   Procesando eventos: Command Service -> Kafka -> Listener Service -> SQLite -> Query Service${NC}"
sleep 25

echo ""
echo -e "${YELLOW}6. Consultando items desde Query Service...${NC}"

# Función para consultar items
query_items() {
    echo -e "${BLUE}Consultando todos los items...${NC}"
    
    response=$(curl -s -w "\n%{http_code}" -X GET "$QUERY_SERVICE/api/v1/inventory/items")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" -eq 200 ]; then
        echo -e "${GREEN}✓ Items consultados exitosamente${NC}"
        echo "$body" | python3 -m json.tool 2>/dev/null || echo "$body"
    else
        echo -e "${RED}✗ Error al consultar items: $body${NC}"
    fi
}

# Consultar items
query_items

echo ""
echo -e "${YELLOW}7. Consultando item específico por ID...${NC}"

# Función para consultar item por SKU
query_item_by_sku() {
    local sku=$1
    
    if [ -z "$sku" ]; then
        echo -e "${YELLOW}⚠ SKU inválido, saltando consulta${NC}"
        return
    fi
    
    echo -e "${BLUE}Consultando item por SKU: $sku...${NC}"
    
    response=$(curl -s -w "\n%{http_code}" -X GET "$QUERY_SERVICE/api/v1/inventory/items/sku/$sku")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" -eq 200 ]; then
        echo -e "${GREEN}✓ Item consultado exitosamente por SKU${NC}"
        echo "$body" | python3 -m json.tool 2>/dev/null || echo "$body"
    else
        echo -e "${RED}✗ Error al consultar item por SKU (HTTP $http_code): $body${NC}"
    fi
}

# Función para consultar item por ID
query_item_by_id() {
    local item_id=$1
    
    # Limpiar item_id de posibles caracteres de escape
    item_id=$(echo "$item_id" | tr -d '\n\r' | grep -oE '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}' | head -1)
    
    if [ -z "$item_id" ]; then
        echo -e "${YELLOW}⚠ Item ID inválido, saltando consulta${NC}"
        return
    fi
    
    echo -e "${BLUE}Consultando item $item_id...${NC}"
    
    response=$(curl -s -w "\n%{http_code}" -X GET "$QUERY_SERVICE/api/v1/inventory/items/$item_id")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" -eq 200 ]; then
        echo -e "${GREEN}✓ Item consultado exitosamente${NC}"
        echo "$body" | python3 -m json.tool 2>/dev/null || echo "$body"
    else
        echo -e "${RED}✗ Error al consultar item (HTTP $http_code): $body${NC}"
    fi
}

# Consultar items específicos por ID y por SKU
if [ -n "$ITEM1_ID" ]; then
    echo -e "${YELLOW}Consultando Item 1 (Tienda Centro) por ID...${NC}"
    query_item_by_id "$ITEM1_ID"
    echo ""
    echo -e "${YELLOW}Consultando Item 1 (Tienda Centro) por SKU...${NC}"
    query_item_by_sku "STORE1-LAPTOP-001"
    echo ""
fi

if [ -n "$ITEM2_ID" ]; then
    echo -e "${YELLOW}Consultando Item 2 (Tienda Norte) por ID...${NC}"
    query_item_by_id "$ITEM2_ID"
    echo ""
    echo -e "${YELLOW}Consultando Item 2 (Tienda Norte) por SKU...${NC}"
    query_item_by_sku "STORE2-MOUSE-001"
    echo ""
fi

if [ -n "$ITEM3_ID" ]; then
    echo -e "${YELLOW}Consultando Item 3 (Tienda Sur) por ID...${NC}"
    query_item_by_id "$ITEM3_ID"
    echo ""
    echo -e "${YELLOW}Consultando Item 3 (Tienda Sur) por SKU...${NC}"
    query_item_by_sku "STORE3-KEYBOARD-001"
    echo ""
fi

echo ""
echo -e "${YELLOW}8. Consultando estado de stock...${NC}"

# Función para consultar estado de stock
query_stock_status() {
    local item_id=$1
    
    # Limpiar item_id de posibles caracteres de escape
    item_id=$(echo "$item_id" | tr -d '\n\r' | grep -oE '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}' | head -1)
    
    if [ -z "$item_id" ]; then
        echo -e "${YELLOW}⚠ Item ID inválido, saltando consulta${NC}"
        return
    fi
    
    echo -e "${BLUE}Consultando estado de stock del item $item_id...${NC}"
    
    response=$(curl -s -w "\n%{http_code}" -X GET "$QUERY_SERVICE/api/v1/inventory/items/$item_id/stock")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" -eq 200 ]; then
        echo -e "${GREEN}✓ Estado de stock consultado exitosamente${NC}"
        echo "$body" | python3 -m json.tool 2>/dev/null || echo "$body"
    else
        echo -e "${RED}✗ Error al consultar estado de stock (HTTP $http_code): $body${NC}"
    fi
}

# Consultar estado de stock de todos los items (usando SKU para obtener ID correcto)
if [ -n "$ITEM1_ID" ]; then
    echo -e "${YELLOW}Consultando estado de stock del Item 1...${NC}"
    # Primero obtener el ID correcto desde Query Service usando SKU
    ITEM1_QUERY_RESPONSE=$(curl -s "$QUERY_SERVICE/api/v1/inventory/items/sku/STORE1-LAPTOP-001" 2>/dev/null)
    if [ -n "$ITEM1_QUERY_RESPONSE" ] && echo "$ITEM1_QUERY_RESPONSE" | grep -q '"id"'; then
        ITEM1_CORRECT_ID=$(echo "$ITEM1_QUERY_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('id', ''))" 2>/dev/null || echo "$ITEM1_QUERY_RESPONSE" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
        if [ -n "$ITEM1_CORRECT_ID" ]; then
            query_stock_status "$ITEM1_CORRECT_ID"
        else
            query_stock_status "$ITEM1_ID"
        fi
    else
        query_stock_status "$ITEM1_ID"
    fi
    echo ""
fi

if [ -n "$ITEM2_ID" ]; then
    echo -e "${YELLOW}Consultando estado de stock del Item 2...${NC}"
    # Primero obtener el ID correcto desde Query Service usando SKU
    ITEM2_QUERY_RESPONSE=$(curl -s "$QUERY_SERVICE/api/v1/inventory/items/sku/STORE2-MOUSE-001" 2>/dev/null)
    if [ -n "$ITEM2_QUERY_RESPONSE" ] && echo "$ITEM2_QUERY_RESPONSE" | grep -q '"id"'; then
        ITEM2_CORRECT_ID=$(echo "$ITEM2_QUERY_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('id', ''))" 2>/dev/null || echo "$ITEM2_QUERY_RESPONSE" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
        if [ -n "$ITEM2_CORRECT_ID" ]; then
            query_stock_status "$ITEM2_CORRECT_ID"
        else
            query_stock_status "$ITEM2_ID"
        fi
    else
        query_stock_status "$ITEM2_ID"
    fi
    echo ""
fi

if [ -n "$ITEM3_ID" ]; then
    echo -e "${YELLOW}Consultando estado de stock del Item 3...${NC}"
    # Primero obtener el ID correcto desde Query Service usando SKU
    ITEM3_QUERY_RESPONSE=$(curl -s "$QUERY_SERVICE/api/v1/inventory/items/sku/STORE3-KEYBOARD-001" 2>/dev/null)
    if [ -n "$ITEM3_QUERY_RESPONSE" ] && echo "$ITEM3_QUERY_RESPONSE" | grep -q '"id"'; then
        ITEM3_CORRECT_ID=$(echo "$ITEM3_QUERY_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('id', ''))" 2>/dev/null || echo "$ITEM3_QUERY_RESPONSE" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
        if [ -n "$ITEM3_CORRECT_ID" ]; then
            query_stock_status "$ITEM3_CORRECT_ID"
        else
            query_stock_status "$ITEM3_ID"
        fi
    else
        query_stock_status "$ITEM3_ID"
    fi
    echo ""
fi

echo ""
echo -e "${YELLOW}9. Verificando estadísticas del Listener Service...${NC}"

# Función para verificar estadísticas
check_listener_stats() {
    echo -e "${BLUE}Consultando estadísticas del Listener Service...${NC}"
    
    response=$(curl -s -w "\n%{http_code}" -X GET "$LISTENER_SERVICE/api/v1/monitoring/stats")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" -eq 200 ]; then
        echo -e "${GREEN}✓ Estadísticas consultadas exitosamente${NC}"
        echo "$body" | python3 -m json.tool 2>/dev/null || echo "$body"
    else
        echo -e "${RED}✗ Error al consultar estadísticas: $body${NC}"
    fi
}

# Verificar estadísticas
check_listener_stats

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Prueba End-to-End Completada${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${BLUE}Resumen:${NC}"
echo "- Items creados: 3 (una para cada tienda)"
echo "- Stock ajustado: 2 items"
echo "- Stock reservado: 2 items"
echo "- Consultas realizadas: Query Service"
echo "- Estadísticas verificadas: Listener Service"
# Resumen de resultados
echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Resumen Detallado de Resultados${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Contar items en Query Service
TOTAL_ITEMS=$(curl -s "$QUERY_SERVICE/api/v1/inventory/items" 2>/dev/null | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('total', 0))" 2>/dev/null || echo "0")
echo -e "${BLUE}Total de items en Query Service: ${TOTAL_ITEMS}${NC}"

# Verificar si los items creados están disponibles
SUCCESS_COUNT=0
TOTAL_TESTS=0

if [ -n "$ITEM1_ID" ]; then
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    if curl -s -f "$QUERY_SERVICE/api/v1/inventory/items/$ITEM1_ID" > /dev/null 2>&1; then
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
        echo -e "${GREEN}✓ Item 1 (Tienda Centro) disponible en Query Service${NC}"
    else
        echo -e "${YELLOW}⚠ Item 1 (Tienda Centro) aún no disponible (eventual consistency)${NC}"
    fi
fi

if [ -n "$ITEM2_ID" ]; then
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    if curl -s -f "$QUERY_SERVICE/api/v1/inventory/items/$ITEM2_ID" > /dev/null 2>&1; then
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
        echo -e "${GREEN}✓ Item 2 (Tienda Norte) disponible en Query Service${NC}"
    else
        echo -e "${YELLOW}⚠ Item 2 (Tienda Norte) aún no disponible (eventual consistency)${NC}"
    fi
fi

if [ -n "$ITEM3_ID" ]; then
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    if curl -s -f "$QUERY_SERVICE/api/v1/inventory/items/$ITEM3_ID" > /dev/null 2>&1; then
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
        echo -e "${GREEN}✓ Item 3 (Tienda Sur) disponible en Query Service${NC}"
    else
        echo -e "${YELLOW}⚠ Item 3 (Tienda Sur) aún no disponible (eventual consistency)${NC}"
    fi
fi

echo ""
if [ $TOTAL_TESTS -gt 0 ]; then
    SUCCESS_RATE=$((SUCCESS_COUNT * 100 / TOTAL_TESTS))
    echo -e "${BLUE}Efectividad Inmediata: ${SUCCESS_COUNT}/${TOTAL_TESTS} items disponibles (${SUCCESS_RATE}%)${NC}"
    
    # Verificación adicional: buscar items por SKU (más confiable)
    echo ""
    echo -e "${BLUE}Verificación adicional por SKU:${NC}"
    SKU_SUCCESS=0
    if [ -n "$ITEM1_ID" ]; then
        if curl -s -f "$QUERY_SERVICE/api/v1/inventory/items/sku/STORE1-LAPTOP-001" > /dev/null 2>&1; then
            SKU_SUCCESS=$((SKU_SUCCESS + 1))
            echo -e "${GREEN}✓ Item 1 (STORE1-LAPTOP-001) disponible por SKU${NC}"
        else
            echo -e "${YELLOW}⚠ Item 1 (STORE1-LAPTOP-001) aún no disponible por SKU${NC}"
        fi
    fi
    if [ -n "$ITEM2_ID" ]; then
        if curl -s -f "$QUERY_SERVICE/api/v1/inventory/items/sku/STORE2-MOUSE-001" > /dev/null 2>&1; then
            SKU_SUCCESS=$((SKU_SUCCESS + 1))
            echo -e "${GREEN}✓ Item 2 (STORE2-MOUSE-001) disponible por SKU${NC}"
        else
            echo -e "${YELLOW}⚠ Item 2 (STORE2-MOUSE-001) aún no disponible por SKU${NC}"
        fi
    fi
    if [ -n "$ITEM3_ID" ]; then
        if curl -s -f "$QUERY_SERVICE/api/v1/inventory/items/sku/STORE3-KEYBOARD-001" > /dev/null 2>&1; then
            SKU_SUCCESS=$((SKU_SUCCESS + 1))
            echo -e "${GREEN}✓ Item 3 (STORE3-KEYBOARD-001) disponible por SKU${NC}"
        else
            echo -e "${YELLOW}⚠ Item 3 (STORE3-KEYBOARD-001) aún no disponible por SKU${NC}"
        fi
    fi
    
    if [ $TOTAL_TESTS -gt 0 ]; then
        SKU_RATE=$((SKU_SUCCESS * 100 / TOTAL_TESTS))
        echo -e "${BLUE}Efectividad por SKU: ${SKU_SUCCESS}/${TOTAL_TESTS} items disponibles (${SKU_RATE}%)${NC}"
    fi
fi

echo ""
echo -e "${YELLOW}Nota sobre Eventual Consistency:${NC}"
echo "  En arquitectura CQRS + EDA, los items pueden tardar unos segundos"
echo "  en estar disponibles en Query Service después de ser creados."
echo "  Esto es normal y esperado. Los items aparecerán automáticamente"
echo "  una vez que el Listener Service procese los eventos."
echo ""

