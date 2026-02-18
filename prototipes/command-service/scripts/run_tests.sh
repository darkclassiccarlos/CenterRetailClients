#!/bin/bash

# Script para ejecutar pruebas unitarias y generar reportes
# Uso: ./scripts/run_tests.sh

set -e

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Directorio base
BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$BASE_DIR"

# Directorio de resultados
RESULTS_DIR="test-results"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
TEST_OUTPUT_DIR="$RESULTS_DIR/$TIMESTAMP"

# Crear directorios
mkdir -p "$TEST_OUTPUT_DIR/coverage"
mkdir -p "$TEST_OUTPUT_DIR/handlers"
mkdir -p "$TEST_OUTPUT_DIR/domain"
mkdir -p "$TEST_OUTPUT_DIR/events"
mkdir -p "$TEST_OUTPUT_DIR/repository"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Ejecutando Pruebas Unitarias${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# Función para ejecutar pruebas de un paquete
run_package_tests() {
    local package=$1
    local output_file=$2
    local coverage_file=$3
    
    echo -e "${YELLOW}Ejecutando pruebas para: $package${NC}"
    
    go test -v -coverprofile="$coverage_file" -covermode=atomic \
        "./internal/$package" \
        > "$output_file" 2>&1
    
    local exit_code=$?
    
    if [ $exit_code -eq 0 ]; then
        echo -e "${GREEN}✓ Pruebas de $package completadas exitosamente${NC}"
        
        # Generar reporte de cobertura en HTML
        if [ -f "$coverage_file" ]; then
            go tool cover -html="$coverage_file" -o "${coverage_file%.out}.html"
        fi
    else
        echo -e "${RED}✗ Pruebas de $package fallaron${NC}"
    fi
    
    return $exit_code
}

# Contadores
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Ejecutar pruebas por paquete
echo -e "${YELLOW}1. Pruebas de Domain (inventory.go)${NC}"
if run_package_tests "domain" \
    "$TEST_OUTPUT_DIR/domain/test_output.txt" \
    "$TEST_OUTPUT_DIR/coverage/domain.out"; then
    ((PASSED_TESTS++))
else
    ((FAILED_TESTS++))
fi
((TOTAL_TESTS++))

echo ""
echo -e "${YELLOW}2. Pruebas de Events (kafka_publisher.go)${NC}"
if run_package_tests "events" \
    "$TEST_OUTPUT_DIR/events/test_output.txt" \
    "$TEST_OUTPUT_DIR/coverage/events.out"; then
    ((PASSED_TESTS++))
else
    ((FAILED_TESTS++))
fi
((TOTAL_TESTS++))

echo ""
echo -e "${YELLOW}3. Pruebas de Handlers (inventory_handler.go)${NC}"
if run_package_tests "handlers" \
    "$TEST_OUTPUT_DIR/handlers/test_output.txt" \
    "$TEST_OUTPUT_DIR/coverage/handlers.out"; then
    ((PASSED_TESTS++))
else
    ((FAILED_TESTS++))
fi
((TOTAL_TESTS++))

echo ""
echo -e "${YELLOW}4. Pruebas de Repository (inventory_repository.go)${NC}"
if run_package_tests "repository" \
    "$TEST_OUTPUT_DIR/repository/test_output.txt" \
    "$TEST_OUTPUT_DIR/coverage/repository.out"; then
    ((PASSED_TESTS++))
else
    ((FAILED_TESTS++))
fi
((TOTAL_TESTS++))

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Resumen de Pruebas${NC}"
echo -e "${GREEN}========================================${NC}"
echo "Total de paquetes: $TOTAL_TESTS"
echo -e "${GREEN}Exitosos: $PASSED_TESTS${NC}"
echo -e "${RED}Fallidos: $FAILED_TESTS${NC}"
echo ""

# Generar cobertura total
echo -e "${YELLOW}Generando reporte de cobertura total...${NC}"
go test -coverprofile="$TEST_OUTPUT_DIR/coverage/total.out" -covermode=atomic ./...
go tool cover -html="$TEST_OUTPUT_DIR/coverage/total.out" -o "$TEST_OUTPUT_DIR/coverage/total.html"

# Mostrar porcentaje de cobertura
if [ -f "$TEST_OUTPUT_DIR/coverage/total.out" ]; then
    COVERAGE=$(go tool cover -func="$TEST_OUTPUT_DIR/coverage/total.out" | grep total | awk '{print $3}')
    echo -e "${GREEN}Cobertura total: $COVERAGE${NC}"
fi

# Generar reporte resumen
cat > "$TEST_OUTPUT_DIR/summary.txt" << EOF
========================================
Resumen de Pruebas Unitarias
========================================
Fecha: $(date)
Timestamp: $TIMESTAMP

Total de paquetes: $TOTAL_TESTS
Exitosos: $PASSED_TESTS
Fallidos: $FAILED_TESTS

Cobertura total: $COVERAGE

Archivos generados:
- test-results/$TIMESTAMP/coverage/total.html (Reporte de cobertura HTML)
- test-results/$TIMESTAMP/coverage/total.out (Datos de cobertura)
- test-results/$TIMESTAMP/*/test_output.txt (Output de pruebas por paquete)
EOF

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Resultados guardados en: $TEST_OUTPUT_DIR${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Para ver el reporte de cobertura HTML:"
echo "  open $TEST_OUTPUT_DIR/coverage/total.html"
echo ""
echo "Para ver el resumen:"
echo "  cat $TEST_OUTPUT_DIR/summary.txt"
echo ""

# Exit con código de error si hay pruebas fallidas
if [ $FAILED_TESTS -gt 0 ]; then
    exit 1
fi

exit 0

