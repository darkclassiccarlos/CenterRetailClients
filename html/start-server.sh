#!/bin/bash

# Script para iniciar el servidor HTTP local en Go

echo "🚀 Iniciando servidor HTTP local para el dashboard..."
echo ""

# Verificar que Go esté instalado
if ! command -v go &> /dev/null; then
    echo "❌ Error: Go no está instalado"
    echo "Por favor instala Go desde: https://golang.org/dl/"
    exit 1
fi

# Verificar que estamos en el directorio correcto
if [ ! -f "server.go" ]; then
    echo "❌ Error: server.go no encontrado"
    echo "Por favor ejecuta este script desde el directorio html/"
    exit 1
fi

# Iniciar el servidor
echo "✅ Go encontrado: $(go version)"
echo ""
echo "Iniciando servidor en http://localhost:8000"
echo "Presiona Ctrl+C para detener el servidor"
echo ""

go run server.go

