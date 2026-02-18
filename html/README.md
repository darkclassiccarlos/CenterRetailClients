# Servidor HTTP Local para Dashboard

Servidor HTTP simple en Go para servir el dashboard HTML con soporte CORS habilitado.

## 🚀 Uso Rápido

### Opción 1: Ejecutar directamente (recomendado)

```bash
cd html
go run server.go
```

### Opción 2: Compilar y ejecutar

```bash
cd html
go build -o dashboard-server server.go
./dashboard-server
```

### Opción 3: Con puerto personalizado

```bash
cd html
go run server.go -port 8080
```

## 📋 Opciones Disponibles

- `-port`: Puerto del servidor HTTP (por defecto: 8000)
- `-dir`: Directorio a servir (por defecto: directorio actual)

## 🌐 Acceso

Una vez iniciado el servidor, abre en tu navegador:

```
http://localhost:8000/index.html
```

## ✅ Características

- ✅ **CORS habilitado**: Permite todas las peticiones desde el navegador
- ✅ **Proxy reverso**: Actúa como proxy para los servicios de backend (Command Service y Query Service)
- ✅ **Soporte para archivos estáticos**: Sirve todos los archivos del directorio
- ✅ **Configuración flexible**: Puerto y directorio configurables
- ✅ **Mensajes informativos**: Muestra la URL y el directorio al iniciar
- ✅ **Enrutamiento inteligente**: Redirige automáticamente las peticiones a los servicios correctos

## 🔄 Cómo Funciona el Proxy

El servidor actúa como **proxy reverso** para resolver problemas de CORS:

1. **Sirve archivos estáticos**: El HTML se sirve desde el directorio local
2. **Proxy para Command Service**: Todas las peticiones a `/api/v1/` (excepto GET a inventory/items) se redirigen a `http://localhost:8080`
3. **Proxy para Query Service**: Las peticiones GET a `/api/v1/inventory/items` se redirigen a `http://localhost:8081`
4. **Headers CORS**: Todas las respuestas incluyen headers CORS necesarios

### Enrutamiento Automático

- **Command Service** (`http://localhost:8080`):
  - `/api/v1/auth/*` - Autenticación
  - `/api/v1/inventory/*` (POST, PUT, DELETE) - Operaciones de escritura
  - `/api/v1/health` - Health check

- **Query Service** (`http://localhost:8081`):
  - `/api/v1/inventory/items` (GET) - Consultas de lectura
  - `/api/v1/health` - Health check

## 🔧 Solución de Problemas

### Error: "go: command not found"
- Asegúrate de tener Go instalado (versión 1.20 o superior)
- Verifica la instalación: `go version`

### Error: "port already in use"
- Usa un puerto diferente: `go run server.go -port 8080`
- O detén el proceso que está usando el puerto 8000

### El dashboard aún muestra errores de CORS
- Asegúrate de abrir `http://localhost:8000/index.html` (no `file://`)
- Verifica que el servidor esté corriendo
- Revisa la consola del navegador para más detalles

## 📝 Notas

- El servidor sirve archivos estáticos desde el directorio actual
- CORS está configurado para permitir todas las peticiones (`Access-Control-Allow-Origin: *`)
- El servidor se detiene con `Ctrl+C`

