#!/bin/bash

# Helper script para autenticación JWT
# Proporciona funciones para obtener y usar tokens JWT

# URLs de los servicios
COMMAND_SERVICE="${COMMAND_SERVICE:-http://localhost:8080}"

# Credenciales por defecto
DEFAULT_USERNAME="${JWT_USERNAME:-admin}"
DEFAULT_PASSWORD="${JWT_PASSWORD:-admin123}"

# Variable global para almacenar el token
JWT_TOKEN=""

# Función para obtener token JWT
get_jwt_token() {
    local username="${1:-$DEFAULT_USERNAME}"
    local password="${2:-$DEFAULT_PASSWORD}"
    
    echo "Obteniendo token JWT para usuario: $username" >&2
    
    response=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$COMMAND_SERVICE/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d "{
            \"username\": \"$username\",
            \"password\": \"$password\"
        }" 2>&1)
    
    http_code=$(echo "$response" | grep "HTTP_CODE:" | cut -d':' -f2 | tr -d '\r\n')
    body=$(echo "$response" | sed '/HTTP_CODE:/d')
    
    if [ -z "$http_code" ] || ! [[ "$http_code" =~ ^[0-9]+$ ]]; then
        echo "ERROR: No se pudo obtener código HTTP" >&2
        echo "Response: $response" >&2
        return 1
    fi
    
    if [ "$http_code" -eq 200 ]; then
        token=$(echo "$body" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
        if [ -n "$token" ]; then
            JWT_TOKEN="$token"
            echo "$token"
            return 0
        else
            echo "ERROR: No se pudo extraer token de la respuesta" >&2
            echo "Response body: $body" >&2
            return 1
        fi
    else
        echo "ERROR: Login falló con código HTTP $http_code" >&2
        echo "Response: $body" >&2
        return 1
    fi
}

# Función para obtener el token almacenado
get_stored_token() {
    echo "$JWT_TOKEN"
}

# Función para establecer el token
set_token() {
    JWT_TOKEN="$1"
}

# Función para verificar si el token está configurado
is_token_set() {
    [ -n "$JWT_TOKEN" ]
}

# Función para hacer login y almacenar el token
login() {
    local username="${1:-$DEFAULT_USERNAME}"
    local password="${2:-$DEFAULT_PASSWORD}"
    
    token=$(get_jwt_token "$username" "$password")
    if [ $? -eq 0 ] && [ -n "$token" ]; then
        set_token "$token"
        echo "✓ Login exitoso, token obtenido" >&2
        return 0
    else
        echo "✗ Error al hacer login" >&2
        return 1
    fi
}

