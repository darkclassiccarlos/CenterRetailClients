#!/bin/bash

# Script de pruebas de casos de uso para todos los endpoints del Dashboard HTML
# Autor: Sistema de Gestión de Inventario (CQRS + EDA)
# Fecha: 2025

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuración
PROXY_URL="http://localhost:8000"
COMMAND_SERVICE_URL="http://localhost:8080"
QUERY_SERVICE_URL="http://localhost:8081"
USE_PROXY=true

# Variables globales
AUTH_TOKEN=""
CREATED_ITEM_ID=""
CREATED_ITEM_SKU="TEST-SKU-$(date +%s)"

# Funciones de utilidad
print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ ERROR: $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_section() {
    echo ""
    echo -e "${YELLOW}=======================================================${NC}"
    echo -e "${YELLOW}$1${NC}"
    echo -e "${YELLOW}=======================================================${NC}"
    echo ""
}

# Variable global para almacenar el body de la última respuesta
LAST_RESPONSE_BODY=""

# Función para hacer peticiones HTTP
make_request() {
    local method=$1
    local url=$2
    local data=$3
    local description=$4
    
    print_info "Probando: $description"
    print_info "URL: $url"
    print_info "Método: $method"
    
    if [ -n "$data" ]; then
        print_info "Body: $data"
    fi
    
    # Construir comando curl
    local curl_cmd="curl -s -w '\nHTTP_CODE:%{http_code}' -X $method"
    
    # Agregar headers
    curl_cmd="$curl_cmd -H 'Content-Type: application/json'"
    curl_cmd="$curl_cmd -H 'accept: application/json'"
    
    # Agregar token si existe
    if [ -n "$AUTH_TOKEN" ]; then
        curl_cmd="$curl_cmd -H 'Authorization: Bearer $AUTH_TOKEN'"
    fi
    
    # Agregar URL
    curl_cmd="$curl_cmd '$url'"
    
    # Agregar body si existe
    if [ -n "$data" ]; then
        curl_cmd="$curl_cmd -d '$data'"
    fi
    
    # Ejecutar y capturar respuesta
    local response=$(eval $curl_cmd 2>&1)
    
    # Extraer código HTTP (compatible con BSD y GNU grep)
    local http_code=$(echo "$response" | tail -1 | sed -n 's/.*HTTP_CODE:\([0-9]*\).*/\1/p')
    
    # Extraer body (todo excepto la última línea con HTTP_CODE)
    local body=$(echo "$response" | sed '$d')
    
    # Guardar body en variable global
    LAST_RESPONSE_BODY="$body"
    
    # Verificar que http_code sea un número válido
    if [ -z "$http_code" ] || ! [[ "$http_code" =~ ^[0-9]+$ ]]; then
        print_error "No se pudo extraer el código HTTP de la respuesta"
        echo "Respuesta completa: $response"
        return 1
    fi
    
    # Mostrar resultado
    if [ "$http_code" -ge 200 ] && [ "$http_code" -lt 300 ]; then
        print_success "HTTP $http_code - $description"
        if [ -n "$body" ]; then
            echo "$body" | jq '.' 2>/dev/null || echo "$body"
        fi
        return 0
    else
        print_error "HTTP $http_code - $description"
        if [ -n "$body" ]; then
            echo "$body" | jq '.' 2>/dev/null || echo "$body"
        fi
        return 1
    fi
}

# Verificar que los servicios estén corriendo
check_services() {
    print_section "Verificando Servicios"
    
    print_info "Verificando Command Service (8080)..."
    if curl -s -f "$COMMAND_SERVICE_URL/api/v1/health" > /dev/null 2>&1; then
        print_success "Command Service está corriendo"
    else
        print_error "Command Service no está disponible en $COMMAND_SERVICE_URL"
        exit 1
    fi
    
    print_info "Verificando Query Service (8081)..."
    if curl -s -f "$QUERY_SERVICE_URL/api/v1/health" > /dev/null 2>&1; then
        print_success "Query Service está corriendo"
    else
        print_error "Query Service no está disponible en $QUERY_SERVICE_URL"
        exit 1
    fi
    
    if [ "$USE_PROXY" = true ]; then
        print_info "Verificando Proxy Server (8000)..."
        if curl -s -f "$PROXY_URL/index.html" > /dev/null 2>&1; then
            print_success "Proxy Server está corriendo"
        else
            print_error "Proxy Server no está disponible en $PROXY_URL"
            exit 1
        fi
    fi
}

# Test 1: Autenticación
test_login() {
    print_section "Test 1: Autenticación (POST /api/v1/auth/login)"
    
    local url
    if [ "$USE_PROXY" = true ]; then
        url="$PROXY_URL/api/v1/auth/login"
    else
        url="$COMMAND_SERVICE_URL/api/v1/auth/login"
    fi
    
    local data='{"username":"admin","password":"admin123"}'
    make_request "POST" "$url" "$data" "Autenticación con Command Service"
    local exit_code=$?
    
    if [ $exit_code -eq 0 ]; then
        # Extraer token de la respuesta JSON (usando variable global)
        AUTH_TOKEN=$(echo "$LAST_RESPONSE_BODY" | jq -r '.token' 2>/dev/null)
        if [ -n "$AUTH_TOKEN" ] && [ "$AUTH_TOKEN" != "null" ]; then
            print_success "Token obtenido: ${AUTH_TOKEN:0:20}..."
            return 0
        else
            print_error "No se pudo extraer el token de la respuesta"
            return 1
        fi
    else
        return 1
    fi
}

# Test 2: Listar Items (GET /api/v1/inventory/items)
test_list_items() {
    print_section "Test 2: Listar Items (GET /api/v1/inventory/items)"
    
    local url
    if [ "$USE_PROXY" = true ]; then
        url="$PROXY_URL/api/v1/inventory/items?page=1&page_size=100"
    else
        url="$QUERY_SERVICE_URL/api/v1/inventory/items?page=1&page_size=100"
    fi
    
    make_request "GET" "$url" "" "Listar items con paginación"
    return $?
}

# Test 3: Buscar Item por SKU (GET /api/v1/inventory/items/sku/:sku)
test_search_by_sku() {
    print_section "Test 3: Buscar Item por SKU (GET /api/v1/inventory/items/sku/:sku)"
    
    if [ -z "$CREATED_ITEM_SKU" ]; then
        print_error "No hay SKU disponible para buscar. Ejecuta primero test_create_item."
        return 1
    fi
    
    local url
    if [ "$USE_PROXY" = true ]; then
        url="$PROXY_URL/api/v1/inventory/items/sku/$CREATED_ITEM_SKU"
    else
        url="$QUERY_SERVICE_URL/api/v1/inventory/items/sku/$CREATED_ITEM_SKU"
    fi
    
    make_request "GET" "$url" "" "Buscar item por SKU: $CREATED_ITEM_SKU"
    return $?
}

# Test 4: Obtener Item por ID (GET /api/v1/inventory/items/:id)
test_get_item_by_id() {
    print_section "Test 4: Obtener Item por ID (GET /api/v1/inventory/items/:id)"
    
    if [ -z "$CREATED_ITEM_ID" ]; then
        print_error "No hay ID disponible para buscar. Ejecuta primero test_create_item."
        return 1
    fi
    
    local url
    if [ "$USE_PROXY" = true ]; then
        url="$PROXY_URL/api/v1/inventory/items/$CREATED_ITEM_ID"
    else
        url="$QUERY_SERVICE_URL/api/v1/inventory/items/$CREATED_ITEM_ID"
    fi
    
    make_request "GET" "$url" "" "Obtener item por ID: $CREATED_ITEM_ID"
    return $?
}

# Test 5: Crear Item (POST /api/v1/inventory/items)
test_create_item() {
    print_section "Test 5: Crear Item (POST /api/v1/inventory/items)"
    
    local url
    if [ "$USE_PROXY" = true ]; then
        url="$PROXY_URL/api/v1/inventory/items"
    else
        url="$COMMAND_SERVICE_URL/api/v1/inventory/items"
    fi
    
    local data="{\"sku\":\"$CREATED_ITEM_SKU\",\"name\":\"Item de Prueba $(date +%s)\",\"description\":\"Descripción del item de prueba\",\"quantity\":100}"
    make_request "POST" "$url" "$data" "Crear nuevo item"
    local exit_code=$?
    
    if [ $exit_code -eq 0 ]; then
        # Extraer ID de la respuesta JSON (usando variable global)
        CREATED_ITEM_ID=$(echo "$LAST_RESPONSE_BODY" | jq -r '.id' 2>/dev/null)
        if [ -n "$CREATED_ITEM_ID" ] && [ "$CREATED_ITEM_ID" != "null" ]; then
            print_success "Item creado con ID: $CREATED_ITEM_ID"
            return 0
        else
            print_error "No se pudo extraer el ID de la respuesta"
            return 1
        fi
    else
        return 1
    fi
}

# Test 6: Actualizar Item (PUT /api/v1/inventory/items/:id)
test_update_item() {
    print_section "Test 6: Actualizar Item (PUT /api/v1/inventory/items/:id)"
    
    if [ -z "$CREATED_ITEM_ID" ]; then
        print_error "No hay ID disponible para actualizar. Ejecuta primero test_create_item."
        return 1
    fi
    
    local url
    if [ "$USE_PROXY" = true ]; then
        url="$PROXY_URL/api/v1/inventory/items/$CREATED_ITEM_ID"
    else
        url="$COMMAND_SERVICE_URL/api/v1/inventory/items/$CREATED_ITEM_ID"
    fi
    
    local data="{\"name\":\"Item Actualizado $(date +%s)\",\"description\":\"Descripción actualizada\"}"
    make_request "PUT" "$url" "$data" "Actualizar item: $CREATED_ITEM_ID"
    return $?
}

# Test 7: Reservar Stock (POST /api/v1/inventory/items/:id/reserve)
test_reserve_stock() {
    print_section "Test 7: Reservar Stock (POST /api/v1/inventory/items/:id/reserve)"
    
    if [ -z "$CREATED_ITEM_ID" ]; then
        print_error "No hay ID disponible para reservar stock. Ejecuta primero test_create_item."
        return 1
    fi
    
    local url
    if [ "$USE_PROXY" = true ]; then
        url="$PROXY_URL/api/v1/inventory/items/$CREATED_ITEM_ID/reserve"
    else
        url="$COMMAND_SERVICE_URL/api/v1/inventory/items/$CREATED_ITEM_ID/reserve"
    fi
    
    local data='{"quantity":5}'
    make_request "POST" "$url" "$data" "Reservar 5 unidades de stock"
    return $?
}

# Test 8: Liberar Stock (POST /api/v1/inventory/items/:id/release)
test_release_stock() {
    print_section "Test 8: Liberar Stock (POST /api/v1/inventory/items/:id/release)"
    
    if [ -z "$CREATED_ITEM_ID" ]; then
        print_error "No hay ID disponible para liberar stock. Ejecuta primero test_create_item."
        return 1
    fi
    
    local url
    if [ "$USE_PROXY" = true ]; then
        url="$PROXY_URL/api/v1/inventory/items/$CREATED_ITEM_ID/release"
    else
        url="$COMMAND_SERVICE_URL/api/v1/inventory/items/$CREATED_ITEM_ID/release"
    fi
    
    local data='{"quantity":2}'
    make_request "POST" "$url" "$data" "Liberar 2 unidades de stock"
    return $?
}

# Test 9: Ajustar Stock (POST /api/v1/inventory/items/:id/adjust)
test_adjust_stock() {
    print_section "Test 9: Ajustar Stock (POST /api/v1/inventory/items/:id/adjust)"
    
    if [ -z "$CREATED_ITEM_ID" ]; then
        print_error "No hay ID disponible para ajustar stock. Ejecuta primero test_create_item."
        return 1
    fi
    
    local url
    if [ "$USE_PROXY" = true ]; then
        url="$PROXY_URL/api/v1/inventory/items/$CREATED_ITEM_ID/adjust"
    else
        url="$COMMAND_SERVICE_URL/api/v1/inventory/items/$CREATED_ITEM_ID/adjust"
    fi
    
    local data='{"quantity":10}'
    make_request "POST" "$url" "$data" "Aumentar stock en 10 unidades"
    return $?
}

# Test 10: Eliminar Item (DELETE /api/v1/inventory/items/:id)
test_delete_item() {
    print_section "Test 10: Eliminar Item (DELETE /api/v1/inventory/items/:id)"
    
    if [ -z "$CREATED_ITEM_ID" ]; then
        print_error "No hay ID disponible para eliminar. Ejecuta primero test_create_item."
        return 1
    fi
    
    local url
    if [ "$USE_PROXY" = true ]; then
        url="$PROXY_URL/api/v1/inventory/items/$CREATED_ITEM_ID"
    else
        url="$COMMAND_SERVICE_URL/api/v1/inventory/items/$CREATED_ITEM_ID"
    fi
    
    make_request "DELETE" "$url" "" "Eliminar item: $CREATED_ITEM_ID"
    return $?
}

# Función principal
main() {
    echo "======================================================="
    echo "🧪 Pruebas de Casos de Uso - Dashboard HTML"
    echo "Sistema de Gestión de Inventario (CQRS + EDA)"
    echo "======================================================="
    echo ""
    
    # Verificar que jq esté instalado
    if ! command -v jq &> /dev/null; then
        print_error "jq no está instalado. Instálalo con: brew install jq (macOS) o apt-get install jq (Linux)"
        exit 1
    fi
    
    # Verificar servicios
    check_services
    
    # Contador de tests
    local passed=0
    local failed=0
    
    # Ejecutar tests en orden
    print_section "Iniciando Pruebas"
    
    # Test 1: Autenticación (requerido para los demás tests)
    if test_login; then
        ((passed++))
    else
        ((failed++))
        print_error "La autenticación falló. No se pueden ejecutar los demás tests."
        exit 1
    fi
    
    # Test 2: Listar Items
    if test_list_items; then
        ((passed++))
    else
        ((failed++))
    fi
    
    # Test 5: Crear Item (debe ejecutarse antes de los tests que requieren un ID)
    if test_create_item; then
        ((passed++))
    else
        ((failed++))
    fi
    
    # Test 3: Buscar por SKU
    if test_search_by_sku; then
        ((passed++))
    else
        ((failed++))
    fi
    
    # Test 4: Obtener por ID
    if test_get_item_by_id; then
        ((passed++))
    else
        ((failed++))
    fi
    
    # Test 6: Actualizar Item
    if test_update_item; then
        ((passed++))
    else
        ((failed++))
    fi
    
    # Test 7: Reservar Stock
    if test_reserve_stock; then
        ((passed++))
    else
        ((failed++))
    fi
    
    # Test 8: Liberar Stock
    if test_release_stock; then
        ((passed++))
    else
        ((failed++))
    fi
    
    # Test 9: Ajustar Stock
    if test_adjust_stock; then
        ((passed++))
    else
        ((failed++))
    fi
    
    # Test 10: Eliminar Item (al final para limpiar)
    if test_delete_item; then
        ((passed++))
    else
        ((failed++))
    fi
    
    # Resumen
    print_section "Resumen de Pruebas"
    echo "Total de tests: $((passed + failed))"
    print_success "Tests pasados: $passed"
    if [ $failed -gt 0 ]; then
        print_error "Tests fallidos: $failed"
    else
        print_success "Tests fallidos: $failed"
    fi
    
    echo ""
    if [ $failed -eq 0 ]; then
        print_success "¡Todos los tests pasaron exitosamente! 🎉"
        exit 0
    else
        print_error "Algunos tests fallaron. Revisa los logs arriba."
        exit 1
    fi
}

# Ejecutar función principal
main

