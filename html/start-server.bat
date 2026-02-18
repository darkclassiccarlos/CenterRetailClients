@echo off
REM Script para iniciar el servidor HTTP local en Go (Windows)

echo 🚀 Iniciando servidor HTTP local para el dashboard...
echo.

REM Verificar que Go esté instalado
where go >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo ❌ Error: Go no está instalado
    echo Por favor instala Go desde: https://golang.org/dl/
    pause
    exit /b 1
)

REM Verificar que estamos en el directorio correcto
if not exist "server.go" (
    echo ❌ Error: server.go no encontrado
    echo Por favor ejecuta este script desde el directorio html/
    pause
    exit /b 1
)

REM Iniciar el servidor
echo ✅ Go encontrado
go version
echo.
echo Iniciando servidor en http://localhost:8000
echo Presiona Ctrl+C para detener el servidor
echo.

go run server.go

pause

