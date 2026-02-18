@echo off
REM Script ejecutable para Windows - Levanta todos los servicios del proyecto
REM Autor: Sistema de Gestión de Inventario (CQRS + EDA)
REM Fecha: 2024

setlocal enabledelayedexpansion

echo =======================================================
echo Sistema de Gestión de Inventario - Inicio de Servicios
echo Arquitectura: CQRS + Event-Driven (EDA)
echo =======================================================
echo.

REM Verificar prerrequisitos
echo [1/4] Verificando prerrequisitos...
echo.

REM Verificar Docker
where docker >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo ❌ ERROR: Docker no está instalado o no está en el PATH
    echo Por favor instala Docker Desktop desde: https://www.docker.com/products/docker-desktop
    pause
    exit /b 1
)
echo ✅ Docker encontrado

REM Verificar Docker Compose
where docker-compose >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    REM Intentar con docker compose (v2)
    docker compose version >nul 2>nul
    if %ERRORLEVEL% NEQ 0 (
        echo ❌ ERROR: Docker Compose no está instalado
        echo Por favor instala Docker Compose
        pause
        exit /b 1
    )
    set DOCKER_COMPOSE_CMD=docker compose
) else (
    set DOCKER_COMPOSE_CMD=docker-compose
)
echo ✅ Docker Compose encontrado

REM Verificar Go
where go >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo ❌ ERROR: Go no está instalado o no está en el PATH
    echo Por favor instala Go desde: https://golang.org/dl/
    pause
    exit /b 1
)
echo ✅ Go encontrado: 
go version
echo.

REM Paso 1: Levantar componentes Docker Compose
echo =======================================================
echo [2/4] 1. Levantar Docker Compose (Kafka, Redis)
echo =======================================================
echo.

REM Obtener el directorio del script
set SCRIPT_DIR=%~dp0
cd /d %SCRIPT_DIR%

REM Levantar Kafka
echo Levantando Kafka...
cd docker-components\kafka
if not exist docker-compose.yml (
    echo ❌ ERROR: docker-compose.yml no encontrado en docker-components\kafka
    cd ..\..
    pause
    exit /b 1
)

%DOCKER_COMPOSE_CMD% up -d
if %ERRORLEVEL% NEQ 0 (
    echo ❌ ERROR: No se pudo levantar Kafka
    cd ..\..
    pause
    exit /b 1
)
echo ✅ Kafka levantado correctamente
cd ..\..

REM Esperar a que Kafka esté listo
echo Esperando a que Kafka esté listo (10 segundos)...
timeout /t 10 /nobreak >nul

REM Levantar Redis
echo Levantando Redis...
cd docker-components\redis
if not exist docker-compose.yml (
    echo ❌ ERROR: docker-compose.yml no encontrado en docker-components\redis
    cd ..\..
    pause
    exit /b 1
)

%DOCKER_COMPOSE_CMD% up -d
if %ERRORLEVEL% NEQ 0 (
    echo ❌ ERROR: No se pudo levantar Redis
    cd ..\..
    pause
    exit /b 1
)
echo ✅ Redis levantado correctamente
cd ..\..

REM Esperar a que Redis esté listo
echo Esperando a que Redis esté listo (5 segundos)...
timeout /t 5 /nobreak >nul
echo.

REM Paso 2: Levantar servicios Go
echo =======================================================
echo [3/4] 2. Compilando y Ejecutando Servicios Go
echo Los logs apareceran en ventanas separadas
echo =======================================================
echo.

REM Verificar que los directorios existen
if not exist "prototipes\command-service" (
    echo ❌ ERROR: prototipes\command-service no encontrado
    pause
    exit /b 1
)

if not exist "prototipes\query-service" (
    echo ❌ ERROR: prototipes\query-service no encontrado
    pause
    exit /b 1
)

if not exist "prototipes\listener-service" (
    echo ❌ ERROR: prototipes\listener-service no encontrado
    pause
    exit /b 1
)

REM Iniciar Command Service (puerto 8080)
echo Iniciando Command Service (puerto 8080)...
start "Command Service (8080)" cmd /k "cd /d %SCRIPT_DIR%prototipes\command-service && echo [Command Service] Iniciando en puerto 8080... && go run cmd\api\main.go"
echo ✅ Command Service iniciado (ventana separada)
timeout /t 3 /nobreak >nul

REM Iniciar Query Service (puerto 8081)
echo Iniciando Query Service (puerto 8081)...
start "Query Service (8081)" cmd /k "cd /d %SCRIPT_DIR%prototipes\query-service && echo [Query Service] Iniciando en puerto 8081... && go run cmd\api\main.go"
echo ✅ Query Service iniciado (ventana separada)
timeout /t 3 /nobreak >nul

REM Iniciar Listener Service
echo Iniciando Listener Service...
start "Listener Service" cmd /k "cd /d %SCRIPT_DIR%prototipes\listener-service && echo [Listener Service] Iniciando consumidor de eventos... && go run cmd\listener\main.go"
echo ✅ Listener Service iniciado (ventana separada)
timeout /t 3 /nobreak >nul
echo.

REM Paso 3: Levantar servicio Go Dashboard
echo =======================================================
echo [4/4] 3. Abriendo Dashboard de Pruebas (localhost:8000)
echo =======================================================
echo.

REM Verificar que el directorio existe
if not exist "html" (
    echo ❌ ERROR: Directorio html no encontrado
    pause
    exit /b 1
)

if not exist "html\server.go" (
    echo ❌ ERROR: server.go no encontrado en html
    pause
    exit /b 1
)

REM Iniciar Dashboard Server
echo Iniciando Dashboard Server (puerto 8000)...
start "Dashboard Server (8000)" cmd /k "cd /d %SCRIPT_DIR%html && echo [Dashboard Server] Iniciando en puerto 8000... && go run server.go"
echo ✅ Dashboard Server iniciado (ventana separada)
timeout /t 3 /nobreak >nul

REM Esperar un poco más para que todos los servicios estén listos
echo Esperando a que todos los servicios estén listos (5 segundos)...
timeout /t 5 /nobreak >nul

REM Abrir el navegador
echo Abriendo navegador en http://localhost:8000/index.html...
start http://localhost:8000/index.html

echo.
echo =======================================================
echo ✅ Todos los servicios han sido iniciados correctamente
echo =======================================================
echo.
echo Servicios disponibles:
echo - Command Service: http://localhost:8080
echo - Query Service: http://localhost:8081
echo - Dashboard: http://localhost:8000/index.html
echo - Kafka (Kafdrop): http://localhost:9000
echo.
echo Para detener los servicios:
echo 1. Cierra las ventanas de los servicios Go (Command, Query, Listener, Dashboard)
echo 2. Ejecuta: cd docker-components\kafka && %DOCKER_COMPOSE_CMD% down
echo 3. Ejecuta: cd docker-components\redis && %DOCKER_COMPOSE_CMD% down
echo.
echo NOTA: Los servicios Go se ejecutan en ventanas separadas.
echo       Revisa las ventanas para ver los logs de cada servicio.
echo.
echo Presiona cualquier tecla para continuar...
pause >nul

