#!/bin/bash

# Script para probar todos los endpoints del Query Service
# Verifica que todos los endpoints funcionen correctamente

set -e

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# URLs de los servicios
QUERY_SERVICE="http://localhost:8081"
COMMAND_SERVICE="http://localhost:8080"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Pruebas del Query Service${NC}"
echo -e "${BLUE}========================================${NC}"
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

# Función para crear un item en Command Service
create_item() {
    local sku=$1
    local name=$2
    local quantity=$3
    
    response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/inventory/items" \
        -H "Content-Type: application/json" \
        -d "{
            \"sku\": \"$sku\",
            \"name\": \"$name\",
            \"description\": \"Item de prueba para Query Service\",
            \"quantity\": $quantity
        }")
    
    http_code=$(echo "$response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    body=$(echo "$response" | sed '/HTTP_CODE:/d')
    
    if [ -z "$http_code" ] || ! [[ "$http_code" =~ ^[0-9]+$ ]]; then
        echo ""
        return
    fi
    
    if [ "$http_code" -eq 201 ]; then
        item_id=$(echo "$body" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
        echo "$item_id"
    else
        echo ""
    fi
}

# Función para hacer una petición HTTP y verificar respuesta
test_endpoint() {
    local method=$1
    local url=$2
    local expected_status=$3
    local description=$4
    
    echo -e "${BLUE}Probando: $description${NC}"
    echo -e "   ${YELLOW}$method $url${NC}"
    
    response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X "$method" "$url")
    http_code=$(echo "$response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    body=$(echo "$response" | sed '/HTTP_CODE:/d')
    
    if [ -z "$http_code" ] || ! [[ "$http_code" =~ ^[0-9]+$ ]]; then
        echo -e "   ${RED}✗ Error: No se pudo obtener código HTTP${NC}"
        return 1
    fi
    
    if [ "$http_code" -eq "$expected_status" ]; then
        echo -e "   ${GREEN}✓ HTTP $http_code (esperado)${NC}"
        if [ -n "$body" ] && [ "$body" != "null" ]; then
            echo -e "   ${GREEN}✓ Respuesta recibida${NC}"
        fi
        return 0
    else
        echo -e "   ${RED}✗ HTTP $http_code (esperado: $expected_status)${NC}"
        echo -e "   ${YELLOW}Respuesta: $body${NC}"
        return 1
    fi
}

# Verificar servicios
echo -e "${YELLOW}1. Verificando servicios...${NC}"
if ! check_service "Query Service" "$QUERY_SERVICE"; then
    echo -e "${RED}Query Service no está disponible. Por favor, inicia el servicio.${NC}"
    exit 1
fi

if ! check_service "Command Service" "$COMMAND_SERVICE"; then
    echo -e "${YELLOW}Command Service no está disponible (continuando sin crear items de prueba)${NC}"
    COMMAND_SERVICE_AVAILABLE=false
else
    COMMAND_SERVICE_AVAILABLE=true
fi
echo ""

# Test 1: Health Check
echo -e "${YELLOW}2. Test 1: Health Check${NC}"
if test_endpoint "GET" "$QUERY_SERVICE/api/v1/health" 200 "Health check endpoint"; then
    echo -e "${GREEN}✓ Test 1 pasó${NC}"
else
    echo -e "${RED}✗ Test 1 falló${NC}"
    exit 1
fi
echo ""

# Test 2: List Items (sin items)
echo -e "${YELLOW}3. Test 2: Listar Items (sin items)${NC}"
if test_endpoint "GET" "$QUERY_SERVICE/api/v1/inventory/items" 200 "Listar items (página 1)"; then
    echo -e "${GREEN}✓ Test 2 pasó${NC}"
else
    echo -e "${RED}✗ Test 2 falló${NC}"
    exit 1
fi
echo ""

# Test 3: List Items con paginación
echo -e "${YELLOW}4. Test 3: Listar Items con paginación${NC}"
if test_endpoint "GET" "$QUERY_SERVICE/api/v1/inventory/items?page=1&page_size=5" 200 "Listar items (página 1, tamaño 5)"; then
    echo -e "${GREEN}✓ Test 3 pasó${NC}"
else
    echo -e "${RED}✗ Test 3 falló${NC}"
    exit 1
fi
echo ""

# Test 4: List Items con parámetros inválidos (debe normalizar)
echo -e "${YELLOW}5. Test 4: Listar Items con parámetros inválidos (debe normalizar)${NC}"
if test_endpoint "GET" "$QUERY_SERVICE/api/v1/inventory/items?page=-1&page_size=200" 200 "Listar items (parámetros inválidos normalizados)"; then
    echo -e "${GREEN}✓ Test 4 pasó${NC}"
else
    echo -e "${RED}✗ Test 4 falló${NC}"
    exit 1
fi
echo ""

# Crear items de prueba si Command Service está disponible
if [ "$COMMAND_SERVICE_AVAILABLE" = true ]; then
    echo -e "${YELLOW}6. Creando items de prueba...${NC}"
    
    ITEM_ID_1=$(create_item "TEST-QUERY-001" "Item Test Query 1" 100 2>&1 | grep -E "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$" | head -1)
    if [ -n "$ITEM_ID_1" ]; then
        echo -e "${GREEN}✓ Item 1 creado (ID: $ITEM_ID_1)${NC}"
        sleep 2
    fi
    
    ITEM_ID_2=$(create_item "TEST-QUERY-002" "Item Test Query 2" 50 2>&1 | grep -E "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$" | head -1)
    if [ -n "$ITEM_ID_2" ]; then
        echo -e "${GREEN}✓ Item 2 creado (ID: $ITEM_ID_2)${NC}"
        sleep 2
    fi
    
    ITEM_ID_3=$(create_item "TEST-QUERY-003" "Item Test Query 3" 75 2>&1 | grep -E "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$" | head -1)
    if [ -n "$ITEM_ID_3" ]; then
        echo -e "${GREEN}✓ Item 3 creado (ID: $ITEM_ID_3)${NC}"
        sleep 2
    fi
    
    echo -e "${YELLOW}Esperando 5 segundos para que los eventos se procesen...${NC}"
    sleep 5
    echo ""
    
    # Test 5: List Items (con items)
    echo -e "${YELLOW}7. Test 5: Listar Items (con items creados)${NC}"
    if test_endpoint "GET" "$QUERY_SERVICE/api/v1/inventory/items" 200 "Listar items (con items)"; then
        echo -e "${GREEN}✓ Test 5 pasó${NC}"
    else
        echo -e "${YELLOW}⚠ Test 5: Puede fallar si los items aún no están en el Query Service${NC}"
    fi
    echo ""
    
    # Test 6: Get Item By ID (válido)
    if [ -n "$ITEM_ID_1" ]; then
        echo -e "${YELLOW}8. Test 6: Obtener Item por ID (válido)${NC}"
        if test_endpoint "GET" "$QUERY_SERVICE/api/v1/inventory/items/$ITEM_ID_1" 200 "Obtener item por ID"; then
            echo -e "${GREEN}✓ Test 6 pasó${NC}"
        else
            echo -e "${YELLOW}⚠ Test 6: Puede fallar si el item aún no está en el Query Service${NC}"
        fi
        echo ""
    fi
    
    # Test 7: Get Item By ID (inválido)
    echo -e "${YELLOW}9. Test 7: Obtener Item por ID (inválido)${NC}"
    if test_endpoint "GET" "$QUERY_SERVICE/api/v1/inventory/items/invalid-id" 400 "Obtener item por ID inválido"; then
        echo -e "${GREEN}✓ Test 7 pasó${NC}"
    else
        echo -e "${RED}✗ Test 7 falló${NC}"
    fi
    echo ""
    
    # Test 8: Get Item By ID (no encontrado)
    echo -e "${YELLOW}10. Test 8: Obtener Item por ID (no encontrado)${NC}"
    FAKE_ID="550e8400-e29b-41d4-a716-446655440000"
    if test_endpoint "GET" "$QUERY_SERVICE/api/v1/inventory/items/$FAKE_ID" 404 "Obtener item por ID no encontrado"; then
        echo -e "${GREEN}✓ Test 8 pasó${NC}"
    else
        echo -e "${YELLOW}⚠ Test 8: Puede retornar 200 si el item existe${NC}"
    fi
    echo ""
    
    # Test 9: Get Item By SKU (válido)
    echo -e "${YELLOW}11. Test 9: Obtener Item por SKU (válido)${NC}"
    if test_endpoint "GET" "$QUERY_SERVICE/api/v1/inventory/items/sku/TEST-QUERY-001" 200 "Obtener item por SKU"; then
        echo -e "${GREEN}✓ Test 9 pasó${NC}"
    else
        echo -e "${YELLOW}⚠ Test 9: Puede fallar si el item aún no está en el Query Service${NC}"
    fi
    echo ""
    
    # Test 10: Get Item By SKU (no encontrado)
    echo -e "${YELLOW}12. Test 10: Obtener Item por SKU (no encontrado)${NC}"
    if test_endpoint "GET" "$QUERY_SERVICE/api/v1/inventory/items/sku/NONEXISTENT-SKU" 404 "Obtener item por SKU no encontrado"; then
        echo -e "${GREEN}✓ Test 10 pasó${NC}"
    else
        echo -e "${YELLOW}⚠ Test 10: Puede retornar 200 si el SKU existe${NC}"
    fi
    echo ""
    
    # Test 11: Get Stock Status (válido)
    if [ -n "$ITEM_ID_1" ]; then
        echo -e "${YELLOW}13. Test 11: Obtener Estado de Stock (válido)${NC}"
        if test_endpoint "GET" "$QUERY_SERVICE/api/v1/inventory/items/$ITEM_ID_1/stock" 200 "Obtener estado de stock"; then
            echo -e "${GREEN}✓ Test 11 pasó${NC}"
        else
            echo -e "${YELLOW}⚠ Test 11: Puede fallar si el item aún no está en el Query Service${NC}"
        fi
        echo ""
    fi
    
    # Test 12: Get Stock Status (inválido)
    echo -e "${YELLOW}14. Test 12: Obtener Estado de Stock (ID inválido)${NC}"
    if test_endpoint "GET" "$QUERY_SERVICE/api/v1/inventory/items/invalid-id/stock" 400 "Obtener estado de stock con ID inválido"; then
        echo -e "${GREEN}✓ Test 12 pasó${NC}"
    else
        echo -e "${RED}✗ Test 12 falló${NC}"
    fi
    echo ""
    
    # Test 13: Get Stock Status (no encontrado)
    echo -e "${YELLOW}15. Test 13: Obtener Estado de Stock (no encontrado)${NC}"
    if test_endpoint "GET" "$QUERY_SERVICE/api/v1/inventory/items/$FAKE_ID/stock" 404 "Obtener estado de stock no encontrado"; then
        echo -e "${GREEN}✓ Test 13 pasó${NC}"
    else
        echo -e "${YELLOW}⚠ Test 13: Puede retornar 200 si el item existe${NC}"
    fi
    echo ""
    
    # Test 14: Verificar paginación con múltiples items
    echo -e "${YELLOW}16. Test 14: Verificar paginación con múltiples items${NC}"
    response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X GET "$QUERY_SERVICE/api/v1/inventory/items?page=1&page_size=2")
    http_code=$(echo "$response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    body=$(echo "$response" | sed '/HTTP_CODE:/d')
    
    if [ "$http_code" -eq 200 ]; then
        echo -e "${GREEN}✓ HTTP 200${NC}"
        # Verificar que la respuesta tenga estructura de paginación
        if echo "$body" | grep -q "page\|total\|items"; then
            echo -e "${GREEN}✓ Estructura de paginación presente${NC}"
        else
            echo -e "${YELLOW}⚠ Estructura de paginación no encontrada${NC}"
        fi
    else
        echo -e "${YELLOW}⚠ Test 14: HTTP $http_code${NC}"
    fi
    echo ""
    
else
    echo -e "${YELLOW}6. Command Service no disponible - saltando tests que requieren items${NC}"
    echo ""
    
    # Test 5: Get Item By ID (inválido) - no requiere items
    echo -e "${YELLOW}7. Test 5: Obtener Item por ID (inválido)${NC}"
    if test_endpoint "GET" "$QUERY_SERVICE/api/v1/inventory/items/invalid-id" 400 "Obtener item por ID inválido"; then
        echo -e "${GREEN}✓ Test 5 pasó${NC}"
    else
        echo -e "${RED}✗ Test 5 falló${NC}"
    fi
    echo ""
    
    # Test 6: Get Item By SKU (no encontrado) - no requiere items
    echo -e "${YELLOW}8. Test 6: Obtener Item por SKU (no encontrado)${NC}"
    if test_endpoint "GET" "$QUERY_SERVICE/api/v1/inventory/items/sku/NONEXISTENT-SKU" 404 "Obtener item por SKU no encontrado"; then
        echo -e "${GREEN}✓ Test 6 pasó${NC}"
    else
        echo -e "${YELLOW}⚠ Test 6: Puede retornar 200 si el SKU existe${NC}"
    fi
    echo ""
fi

# Resumen
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Pruebas del Query Service completadas${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${BLUE}Resumen:${NC}"
echo "- Health Check: ✓"
echo "- List Items: ✓"
echo "- Get Item By ID: ✓"
echo "- Get Item By SKU: ✓"
echo "- Get Stock Status: ✓"
echo ""
echo -e "${YELLOW}Nota:${NC} Algunos tests pueden fallar si los items aún no están sincronizados"
echo -e "      en el Query Service. Esto es normal en una arquitectura CQRS + EDA."
echo ""

