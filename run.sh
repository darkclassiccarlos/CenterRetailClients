#!/bin/bash

# Script ejecutable para macOS/Linux - Levanta todos los servicios del proyecto
# Autor: Sistema de Gestión de Inventario (CQRS + EDA)
# Fecha: 2024

# No usar set -e porque algunos comandos pueden fallar pero no son críticos

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Función para imprimir mensajes
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
    echo "======================================================="
    echo "$1"
    echo "======================================================="
    echo ""
}

# Función para verificar comandos
check_command() {
    if ! command -v $1 &> /dev/null; then
        print_error "$1 no está instalado o no está en el PATH"
        return 1
    fi
    return 0
}

# Función para esperar
wait_seconds() {
    print_info "Esperando $1 segundos..."
    sleep $1
}

# Inicio del script
print_section "Sistema de Gestión de Inventario - Inicio de Servicios"
echo "Arquitectura: CQRS + Event-Driven (EDA)"
echo ""

# Verificar prerrequisitos
print_section "[1/4] Verificando prerrequisitos..."

# Verificar Docker
if ! check_command docker; then
    print_error "Docker no está instalado"
    echo "Por favor instala Docker Desktop desde: https://www.docker.com/products/docker-desktop"
    exit 1
fi
print_success "Docker encontrado: $(docker --version)"

# Verificar Docker Compose
if command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE_CMD="docker-compose"
    print_success "Docker Compose encontrado: $(docker-compose --version)"
elif docker compose version &> /dev/null; then
    DOCKER_COMPOSE_CMD="docker compose"
    print_success "Docker Compose encontrado: $(docker compose version)"
else
    print_error "Docker Compose no está instalado"
    echo "Por favor instala Docker Compose"
    exit 1
fi

# Verificar Go
if ! check_command go; then
    print_error "Go no está instalado"
    echo "Por favor instala Go desde: https://golang.org/dl/"
    exit 1
fi
print_success "Go encontrado: $(go version)"
echo ""

# Paso 1: Levantar componentes Docker Compose
print_section "[2/4] 1. Levantar Docker Compose (Kafka, Redis)"

# Levantar Kafka
print_info "Levantando Kafka..."
cd docker-components/kafka || {
    print_error "No se pudo acceder a docker-components/kafka"
    exit 1
}

if [ ! -f "docker-compose.yml" ]; then
    print_error "docker-compose.yml no encontrado en docker-components/kafka"
    cd ../..
    exit 1
fi

if ! $DOCKER_COMPOSE_CMD up -d; then
    print_error "No se pudo levantar Kafka"
    cd ../..
    exit 1
fi
print_success "Kafka levantado correctamente"
cd ../.. || exit 1

# Esperar a que Kafka esté listo
wait_seconds 10

# Levantar Redis
print_info "Levantando Redis..."
cd docker-components/redis || {
    print_error "No se pudo acceder a docker-components/redis"
    exit 1
}

if [ ! -f "docker-compose.yml" ]; then
    print_error "docker-compose.yml no encontrado en docker-components/redis"
    cd ../..
    exit 1
fi

if ! $DOCKER_COMPOSE_CMD up -d; then
    print_error "No se pudo levantar Redis"
    cd ../..
    exit 1
fi
print_success "Redis levantado correctamente"
cd ../.. || exit 1

# Esperar a que Redis esté listo
wait_seconds 5
echo ""

# Paso 2: Levantar servicios Go
print_section "[3/4] 2. Compilando y Ejecutando Servicios Go"
echo "Los logs aparecerán en terminales separadas"
echo ""

# Verificar que los directorios existen
if [ ! -d "prototipes/command-service" ]; then
    print_error "prototipes/command-service no encontrado"
    exit 1
fi

if [ ! -d "prototipes/query-service" ]; then
    print_error "prototipes/query-service no encontrado"
    exit 1
fi

if [ ! -d "prototipes/listener-service" ]; then
    print_error "prototipes/listener-service no encontrado"
    exit 1
fi

# Obtener la ruta absoluta del proyecto
PROJECT_ROOT=$(cd "$(dirname "$0")" && pwd)

# Función para iniciar servicio Go en background
start_go_service() {
    local service_name=$1
    local service_path=$2
    local main_file=$3
    local port=$4
    
    print_info "Iniciando $service_name (puerto $port)..."
    
    # Obtener la ruta absoluta del servicio
    local full_service_path="$PROJECT_ROOT/$service_path"
    
    # Crear un script temporal para ejecutar el servicio
    local temp_script=$(mktemp)
    cat > "$temp_script" << EOF
#!/bin/bash
cd "$full_service_path"
echo "[$service_name] Iniciando en puerto $port..."
echo "Directorio: \$(pwd)"
go run $main_file
EOF
    chmod +x "$temp_script"
    
    # Ejecutar en una nueva terminal (macOS/Linux)
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        osascript -e "tell app \"Terminal\" to do script \"$temp_script\"" 2>/dev/null || {
            # Fallback: ejecutar en background
            nohup bash "$temp_script" > "/tmp/${service_name// /_}.log" 2>&1 &
            print_info "Servicio ejecutándose en background. Logs en: /tmp/${service_name// /_}.log"
        }
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        # Linux - intentar con diferentes terminales
        if command -v gnome-terminal &> /dev/null; then
            gnome-terminal -- bash -c "$temp_script; exec bash" 2>/dev/null || {
                nohup bash "$temp_script" > "/tmp/${service_name// /_}.log" 2>&1 &
                print_info "Servicio ejecutándose en background. Logs en: /tmp/${service_name// /_}.log"
            }
        elif command -v xterm &> /dev/null; then
            xterm -e "$temp_script" 2>/dev/null &
        else
            # Si no hay terminal gráfica, ejecutar en background
            nohup bash "$temp_script" > "/tmp/${service_name// /_}.log" 2>&1 &
            print_info "Servicio ejecutándose en background. Logs en: /tmp/${service_name// /_}.log"
        fi
    else
        # Fallback: ejecutar en background
        nohup bash "$temp_script" > "/tmp/${service_name// /_}.log" 2>&1 &
        print_info "Servicio ejecutándose en background. Logs en: /tmp/${service_name// /_}.log"
    fi
    
    print_success "$service_name iniciado"
    wait_seconds 3
}

# Iniciar Command Service (puerto 8080)
start_go_service "Command Service" "prototipes/command-service" "cmd/api/main.go" "8080"

# Iniciar Query Service (puerto 8081)
start_go_service "Query Service" "prototipes/query-service" "cmd/api/main.go" "8081"

# Iniciar Listener Service
start_go_service "Listener Service" "prototipes/listener-service" "cmd/listener/main.go" "N/A"

echo ""

# Paso 3: Levantar servicio Go Dashboard
print_section "[4/4] 3. Abriendo Dashboard de Pruebas (localhost:8000)"

# Verificar que el directorio existe
if [ ! -d "html" ]; then
    print_error "Directorio html no encontrado"
    exit 1
fi

if [ ! -f "html/server.go" ]; then
    print_error "server.go no encontrado en html"
    exit 1
fi

# Iniciar Dashboard Server
print_info "Iniciando Dashboard Server (puerto 8000)..."

# Crear script temporal para el dashboard
dashboard_script=$(mktemp)
cat > "$dashboard_script" << EOF
#!/bin/bash
cd "$PROJECT_ROOT/html"
echo "[Dashboard Server] Iniciando en puerto 8000..."
echo "Directorio: \$(pwd)"
go run server.go
EOF
chmod +x "$dashboard_script"

# Ejecutar en una nueva terminal (macOS/Linux)
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    osascript -e "tell app \"Terminal\" to do script \"$dashboard_script\"" 2>/dev/null || {
        nohup bash "$dashboard_script" > "/tmp/dashboard-server.log" 2>&1 &
        print_info "Dashboard ejecutándose en background. Logs en: /tmp/dashboard-server.log"
    }
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    # Linux
    if command -v gnome-terminal &> /dev/null; then
        gnome-terminal -- bash -c "$dashboard_script; exec bash" 2>/dev/null || {
            nohup bash "$dashboard_script" > "/tmp/dashboard-server.log" 2>&1 &
            print_info "Dashboard ejecutándose en background. Logs en: /tmp/dashboard-server.log"
        }
    elif command -v xterm &> /dev/null; then
        xterm -e "$dashboard_script" 2>/dev/null &
    else
        nohup bash "$dashboard_script" > "/tmp/dashboard-server.log" 2>&1 &
        print_info "Dashboard ejecutándose en background. Logs en: /tmp/dashboard-server.log"
    fi
else
    nohup bash "$dashboard_script" > "/tmp/dashboard-server.log" 2>&1 &
    print_info "Dashboard ejecutándose en background. Logs en: /tmp/dashboard-server.log"
fi

print_success "Dashboard Server iniciado"
wait_seconds 3

# Esperar un poco más para que todos los servicios estén listos
wait_seconds 5

# Abrir el navegador
print_info "Abriendo navegador en http://localhost:8000/index.html..."
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    open http://localhost:8000/index.html
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    # Linux
    if command -v xdg-open &> /dev/null; then
        xdg-open http://localhost:8000/index.html
    elif command -v firefox &> /dev/null; then
        firefox http://localhost:8000/index.html &
    elif command -v google-chrome &> /dev/null; then
        google-chrome http://localhost:8000/index.html &
    else
        print_info "Por favor abre manualmente: http://localhost:8000/index.html"
    fi
else
    print_info "Por favor abre manualmente: http://localhost:8000/index.html"
fi

echo ""
print_section "✅ Todos los servicios han sido iniciados correctamente"
echo ""
echo "Servicios disponibles:"
echo "  - Command Service: http://localhost:8080"
echo "  - Query Service: http://localhost:8081"
echo "  - Dashboard: http://localhost:8000/index.html"
echo "  - Kafka (Kafdrop): http://localhost:9000"
echo ""
echo "Para detener los servicios:"
echo "  1. Cierra las terminales de los servicios Go (Command, Query, Listener, Dashboard)"
echo "  2. Ejecuta: cd docker-components/kafka && $DOCKER_COMPOSE_CMD down"
echo "  3. Ejecuta: cd docker-components/redis && $DOCKER_COMPOSE_CMD down"
echo ""
echo "NOTA: Los servicios Go se ejecutan en terminales separadas."
echo "      Revisa las terminales para ver los logs de cada servicio."
echo ""
print_info "Presiona Ctrl+C para salir (los servicios seguirán corriendo)"
echo ""

