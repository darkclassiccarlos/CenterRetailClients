# 🚀 Manual de Ejecución: Sistema de Gestión de Inventario (CQRS + EDA)

Este proyecto implementa una arquitectura distribuida basada en **CQRS (Command Query Responsibility Segregation)** y **Event-Driven Architecture (EDA)** para la gestión de inventario.

## 📋 Tabla de Contenidos

1. [Requisitos Previos](#-requisitos-previos)
2. [Ejecución Automática (Recomendado)](#-ejecución-automática-recomendado)
3. [Ejecución Manual (Pasos Detallados)](#️-ejecución-manual-pasos-detallados)
4. [Verificación de Servicios](#-verificación-de-servicios)
5. [Acceso a los Servicios](#-acceso-a-los-servicios)
6. [Detener los Servicios](#-detener-los-servicios)
7. [Solución de Problemas](#-solución-de-problemas)
8. [Información para Evaluadores](#-información-para-evaluadores)
9. [Pruebas de Endpoints](#-pruebas-de-endpoints)

---

## ⚙️ Requisitos Previos

Asegúrese de tener instalados los siguientes componentes antes de ejecutar el proyecto:

### 1. Docker & Docker Compose

- **Docker Desktop** (Windows/macOS) o **Docker Engine** (Linux)
- **Docker Compose** versión 2.0 o superior

**Verificación:**
```bash
docker --version
docker compose version
# o
docker-compose --version
```

**Instalación:**
- **Windows/macOS:** https://www.docker.com/products/docker-desktop
- **Linux:** Siga la documentación oficial de Docker

### 2. Go (Golang)

- **Versión requerida**: Go 1.20 o superior

**Verificación:**
```bash
go version
```

**Instalación:**
- Descargar desde: https://golang.org/dl/
- Configurar variables de entorno según su sistema operativo:
  - **Windows:** Agregar `C:\Program Files\Go\bin` al PATH
  - **Linux/macOS:** Agregar a `~/.bashrc` o `~/.zshrc`:
    ```bash
    export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin
    export GOPATH=$HOME/go
    ```

### 3. Git (Opcional)

- Para clonar el repositorio si es necesario

**Verificación:**
```bash
git --version
```

### 4. jq (Opcional - para pruebas)

- Para parsear JSON en los scripts de prueba

**Instalación:**
- **macOS:** `brew install jq`
- **Linux:** `apt-get install jq` o `yum install jq`

---

## ⚡ Ejecución Automática (Recomendado)

Para una experiencia óptima, ejecute el script auto-contenido en la raíz del proyecto:

### Windows

```bash
run.bat
```

O haga **doble clic** en `run.bat`

### macOS/Linux

```bash
./run.sh
```

O:

```bash
bash run.sh
```

### ¿Qué hace el script automáticamente?

El script realizará los siguientes pasos en orden:

1. ✅ **Verificación de prerrequisitos** (Docker, Docker Compose, Go)
2. ✅ **Levantará Kafka y Redis** usando `docker-compose`
3. ✅ **Compilará y ejecutará el Command Service** (Go) en puerto 8080
4. ✅ **Compilará y ejecutará el Query Service** (Go) en puerto 8081
5. ✅ **Compilará y ejecutará el Listener Service** (Go)
6. ✅ **Compilará y ejecutará el Dashboard Server** (Go) en puerto 8000
7. ✅ **Abrirá el Dashboard de Pruebas** (`index.html`) en su navegador (`localhost:8000`)

**Nota:** Los servicios Go se ejecutan en ventanas/terminales separadas para facilitar el monitoreo de logs.

### Proceso de Inicio Detallado

#### Paso 1: Verificación de Prerrequisitos

El script verifica automáticamente que Docker, Docker Compose y Go estén instalados y disponibles. Si falta algún prerrequisito, el script mostrará un error y se detendrá.

#### Paso 2: Levantar Docker Compose

1. **Kafka** (`docker-components/kafka/docker-compose.yml`)
   - Zookeeper (puerto 2181)
   - Kafka (puertos 9092, 9093, 29093)
   - Kafdrop (puerto 9000) - Visualización web
   - Kafka Init - Crea topics automáticamente

2. **Redis** (`docker-components/redis/docker-compose.yml`)
   - Redis (puerto 6379)
   - Persistencia habilitada

**Espera:** El script espera 10 segundos después de levantar Kafka y 5 segundos después de levantar Redis para asegurar que los servicios estén listos.

#### Paso 3: Levantar Servicios Go

Los servicios Go se ejecutan en ventanas/terminales separadas:

1. **Command Service** (puerto 8080)
   - Ubicación: `prototipes/command-service/cmd/api/main.go`
   - Ventana: "Command Service (8080)"
   - Inicializa base de datos SQLite si es necesario

2. **Query Service** (puerto 8081)
   - Ubicación: `prototipes/query-service/cmd/api/main.go`
   - Ventana: "Query Service (8081)"
   - Lee desde base de datos SQLite y Redis Cache

3. **Listener Service**
   - Ubicación: `prototipes/listener-service/cmd/listener/main.go`
   - Ventana: "Listener Service"
   - Consume eventos de Kafka y actualiza SQLite

**Espera:** El script espera 3 segundos entre cada servicio para asegurar que se inicien correctamente.

#### Paso 4: Levantar Dashboard Server

El Dashboard Server se ejecuta en una ventana/terminal separada:

- **Dashboard Server** (puerto 8000)
  - Ubicación: `html/server.go`
  - Ventana: "Dashboard Server (8000)"
  - Abre automáticamente el navegador en `http://localhost:8000/index.html`
  - Actúa como proxy reverso para resolver problemas de CORS

**Espera:** El script espera 5 segundos adicionales después de iniciar el Dashboard Server para asegurar que todos los servicios estén listos.

### Validaciones y Manejo de Errores

Los scripts incluyen validaciones en cada paso:

- ✅ Verificación de prerrequisitos (Docker, Docker Compose, Go)
- ✅ Verificación de existencia de archivos y directorios
- ✅ Verificación de éxito de comandos Docker Compose
- ✅ Manejo de errores con mensajes descriptivos
- ✅ Retorno de códigos de error apropiados
- ✅ Pausa en Windows para ver errores antes de cerrar

---

## 🛠️ Ejecución Manual (Pasos Detallados)

Si prefiere ejecutar los servicios manualmente o necesita entender el proceso paso a paso, siga estas instrucciones:

### Paso 1: Levantar Componentes Docker Compose

#### 1.1. Levantar Kafka

```bash
cd docker-components/kafka
docker compose up -d
# o
docker-compose up -d
```

**Verificación:**
```bash
docker ps
# Debería ver: zookeeper, kafka, kafdrop, kafka-init
```

**Esperar:** 10-15 segundos para que Kafka esté completamente listo

**Verificar Topics:**
```bash
docker exec -it kafka kafka-topics --list --bootstrap-server localhost:9092
```

**Topics esperados:**
- `inventory.items`
- `inventory.stock`
- `inventory.dlq`

**Acceso:**
- Kafdrop (Visualización): http://localhost:9000
- Kafka Broker: localhost:9092

#### 1.2. Levantar Redis

```bash
cd docker-components/redis
docker compose up -d
# o
docker-compose up -d
```

**Verificación:**
```bash
docker ps
# Debería ver: redis
```

**Esperar:** 5 segundos para que Redis esté completamente listo

**Verificar Redis:**
```bash
docker exec -it redis redis-cli ping
# Respuesta esperada: PONG
```

**Acceso:**
- Redis: localhost:6379

### Paso 2: Compilar y Ejecutar Servicios Go

#### 2.1. Command Service (Puerto 8080)

**Ubicación:** `prototipes/command-service`

```bash
cd prototipes/command-service
go mod download
go run cmd/api/main.go
```

**Verificación:**
```bash
curl http://localhost:8080/api/v1/health
# Respuesta esperada: {"status":"ok","service":"command-service"}
```

**Swagger:** http://localhost:8080/swagger/index.html

**Funcionalidades:**
- Operaciones de escritura (POST, PUT, DELETE)
- Publicación de eventos a Kafka
- Autenticación JWT

#### 2.2. Query Service (Puerto 8081)

**Ubicación:** `prototipes/query-service`

```bash
cd prototipes/query-service
go mod download
go run cmd/api/main.go
```

**Verificación:**
```bash
curl http://localhost:8081/api/v1/health
# Respuesta esperada: {"status":"ok","service":"query-service"}
```

**Swagger:** http://localhost:8081/swagger/index.html

**Funcionalidades:**
- Operaciones de lectura (GET)
- Cache con Redis (opcional)
- Lectura optimizada desde SQLite

#### 2.3. Listener Service

**Ubicación:** `prototipes/listener-service`

```bash
cd prototipes/listener-service
go mod download
go run cmd/listener/main.go
```

**Nota:** Este servicio no expone una API HTTP, solo consume eventos de Kafka y actualiza la base de datos SQLite.

**Verificación:**
- Revisar los logs del servicio para confirmar que está consumiendo eventos
- Verificar que el archivo `inventory.db` se esté actualizando
- Ver eventos en Kafdrop: http://localhost:9000

**Funcionalidades:**
- Consumo de eventos de Kafka
- Actualización de SQLite (Single Writer)
- Optimistic Locking para control de concurrencia

### Paso 3: Levantar Dashboard Server

**Ubicación:** `html`

```bash
cd html
go mod download
go run server.go
```

**Acceso:**
- Dashboard: http://localhost:8000/index.html

**Nota:** El servidor actúa como proxy reverso para resolver problemas de CORS y redirige las peticiones a los servicios backend:
- `GET /api/v1/inventory/*` → Query Service (8081)
- `POST/PUT/DELETE /api/v1/inventory/*` → Command Service (8080)

---

## ✅ Verificación de Servicios

### Verificar Servicios Docker

```bash
docker ps
```

**Debería mostrar:**
- `zookeeper` (puerto 2181)
- `kafka` (puertos 9092, 9093, 29093)
- `kafdrop` (puerto 9000)
- `kafka-init` (temporal, crea topics)
- `redis` (puerto 6379)

### Verificar Servicios Go

#### Command Service
```bash
curl http://localhost:8080/api/v1/health
```

**Respuesta esperada:**
```json
{"status":"ok","service":"command-service"}
```

#### Query Service
```bash
curl http://localhost:8081/api/v1/health
```

**Respuesta esperada:**
```json
{"status":"ok","service":"query-service"}
```

#### Dashboard Server
```bash
curl http://localhost:8000/index.html
```

**Respuesta esperada:** HTML del dashboard

### Verificar Kafka

```bash
# Ver topics creados
docker exec -it kafka kafka-topics --list --bootstrap-server localhost:9092
```

**Topics esperados:**
- `inventory.items`
- `inventory.stock`
- `inventory.dlq`

**Ver mensajes en un topic:**
```bash
docker exec -it kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic inventory.items --from-beginning
```

### Verificar Redis

```bash
# Conectar a Redis
docker exec -it redis redis-cli ping
```

**Respuesta esperada:** `PONG`

**Ver datos en Redis (si está habilitado):**
```bash
docker exec -it redis redis-cli
> KEYS *
> GET [key]
```

### Verificar Base de Datos SQLite

```bash
# Verificar que el archivo existe
ls -la prototipes/listener-service/inventory.db

# Ver contenido (requiere sqlite3)
sqlite3 prototipes/listener-service/inventory.db "SELECT * FROM inventory_items;"
```

---

## 🌐 Acceso a los Servicios

Una vez que todos los servicios estén corriendo, estarán disponibles en:

| Servicio | URL | Descripción |
|----------|-----|-------------|
| **Dashboard** | http://localhost:8000/index.html | Dashboard de pruebas con interfaz web |
| **Command Service** | http://localhost:8080 | API de escritura (POST, PUT, DELETE) |
| **Command Service Swagger** | http://localhost:8080/swagger/index.html | Documentación interactiva de la API |
| **Query Service** | http://localhost:8081 | API de lectura (GET) |
| **Query Service Swagger** | http://localhost:8081/swagger/index.html | Documentación interactiva de la API |
| **Listener Service Swagger** | http://localhost:8082/swagger/index.html | Documentación del Listener Service |
| **Kafdrop** | http://localhost:9000 | Visualización web de Kafka (topics, mensajes) |
| **Redis** | localhost:6379 | Cache distribuido (opcional) |

### Credenciales de Autenticación

**Usuario:** `admin`  
**Contraseña:** `admin123`

**Nota:** El token JWT tiene una duración de 10 minutos. El dashboard se autentica automáticamente y se re-autentica cuando el token está por expirar.

---

## 🛑 Detener los Servicios

### Opción 1: Detener Manualmente (Recomendado)

Los scripts no incluyen una función de detención automática. Debe detener los servicios manualmente:

1. **Cerrar ventanas/terminales de servicios Go:**
   - Command Service
   - Query Service
   - Listener Service
   - Dashboard Server

2. **Detener Docker Compose:**

**Windows:**
```bash
cd docker-components\kafka
docker-compose down

cd ..\redis
docker-compose down
```

**macOS/Linux:**
```bash
cd docker-components/kafka
docker compose down

cd ../redis
docker compose down
```

### Opción 2: Detener Todo Docker

```bash
docker stop $(docker ps -q)
```

**⚠️ Advertencia:** Esto detendrá todos los contenedores Docker en ejecución.

### Opción 3: Detener Servicios Específicos

```bash
# Detener solo Kafka
cd docker-components/kafka
docker compose down

# Detener solo Redis
cd docker-components/redis
docker compose down
```

---

## 🔧 Solución de Problemas

### Error: "Docker no está instalado"

**Solución:**
- **Windows/macOS:** Instale Docker Desktop desde https://www.docker.com/products/docker-desktop
- **Linux:** Instale Docker Engine siguiendo la documentación oficial
- Verifique que Docker esté corriendo: `docker ps`

### Error: "Docker Compose no está instalado"

**Solución:**
- **Windows/macOS:** Docker Compose viene incluido con Docker Desktop
- **Linux:** Instale Docker Compose siguiendo la documentación oficial
- Verifique la versión: `docker compose version` o `docker-compose --version`

### Error: "Go no está instalado"

**Solución:**
- Descargue e instale Go desde https://golang.org/dl/
- Configure las variables de entorno:
  - **Windows:** Agregar `C:\Program Files\Go\bin` al PATH
  - **Linux/macOS:** Agregar a `~/.bashrc` o `~/.zshrc`:
    ```bash
    export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin
    export GOPATH=$HOME/go
    ```
- Verifique la instalación: `go version`

### Error: "No se pudo levantar Kafka/Redis"

**Posibles causas:**
1. Docker no está corriendo
2. Puertos en uso (2181, 9092, 6379)
3. Permisos insuficientes

**Solución:**
```bash
# Verificar que Docker esté corriendo
docker ps

# Verificar puertos en uso
# Windows:
netstat -ano | findstr :9092
# Linux/macOS:
lsof -i :9092

# Ver logs de Docker Compose
cd docker-components/kafka
docker compose logs

cd ../redis
docker compose logs
```

### Error: "No se pudo iniciar [Servicio Go]"

**Posibles causas:**
1. Go no está instalado
2. Dependencias no instaladas
3. Puerto en uso

**Solución:**
```bash
# Verificar Go
go version

# Instalar dependencias
cd prototipes/[service-name]
go mod download

# Verificar puerto en uso
# Windows:
netstat -ano | findstr :8080
# Linux/macOS:
lsof -i :8080
```

### Error: "CORS" en el Dashboard

**Solución:**
- Asegúrese de usar el Dashboard Server (puerto 8000), no abra el HTML directamente
- El Dashboard Server incluye proxy CORS automático
- Acceda a: http://localhost:8000/index.html (no `file://`)

### Error: "Token expirado" o "No autorizado"

**Solución:**
- El dashboard se autentica automáticamente
- Si el token expira, el dashboard se re-autentica automáticamente
- Verifique que el Command Service esté corriendo en el puerto 8080
- Verifique las credenciales: usuario `admin`, contraseña `admin123`

### Los servicios no se inician en ventanas separadas

**Windows:**
- Verifique que `cmd` esté disponible
- Los servicios se ejecutan en ventanas CMD separadas

**macOS:**
- Verifique que Terminal.app esté disponible
- Los servicios se ejecutan en terminales separadas

**Linux:**
- Verifique que tenga un terminal gráfico instalado (gnome-terminal, xterm, etc.)
- Si no hay terminal gráfica, los servicios se ejecutan en background
- Los logs estarán en `/tmp/[service-name].log`

### Error: "Topics no creados en Kafka"

**Solución:**
- El servicio `kafka-init` crea los topics automáticamente
- Verifique los logs: `docker logs kafka-init`
- Cree los topics manualmente si es necesario:
  ```bash
  docker exec -it kafka kafka-topics --create --bootstrap-server localhost:9092 --topic inventory.items --partitions 3 --replication-factor 1
  docker exec -it kafka kafka-topics --create --bootstrap-server localhost:9092 --topic inventory.stock --partitions 3 --replication-factor 1
  docker exec -it kafka kafka-topics --create --bootstrap-server localhost:9092 --topic inventory.dlq --partitions 1 --replication-factor 1
  ```

---

## 👨‍💼 Información para Evaluadores

### Proceso de Evaluación Recomendado

1. **Ejecutar el script automático:**
   ```bash
   # Windows
   run.bat
   
   # macOS/Linux
   ./run.sh
   ```

2. **Verificar que todos los servicios estén corriendo:**
   ```bash
   # Servicios Docker
   docker ps
   
   # Servicios Go (verificar en ventanas/terminales separadas)
   # - Command Service (puerto 8080)
   # - Query Service (puerto 8081)
   # - Listener Service
   # - Dashboard Server (puerto 8000)
   ```

3. **Acceder al Dashboard:**
   - URL: http://localhost:8000/index.html
   - El dashboard debería autenticarse automáticamente
   - Verificar estado de servicios en el dashboard

4. **Probar funcionalidades:**
   - Crear items de inventario
   - Consultar stock
   - Reservar/liberar stock
   - Ver eventos en Kafdrop

### Verificación de Arquitectura

#### CQRS (Command Query Responsibility Segregation)

- **Command Service (8080):** Solo operaciones de escritura (POST, PUT, DELETE)
- **Query Service (8081):** Solo operaciones de lectura (GET)

#### Event-Driven Architecture (EDA)

- **Event Broker:** Kafka (puerto 9092)
- **Event Producer:** Command Service publica eventos
- **Event Consumer:** Listener Service consume eventos
- **Visualización:** Kafdrop (puerto 9000)

#### Single Writer Principle

- **Listener Service:** Único escritor de la base de datos SQLite
- **Optimistic Locking:** Control de concurrencia mediante campo `version`

### Endpoints Principales

#### Command Service (http://localhost:8080)

- `POST /api/v1/auth/login` - Autenticación (usuario: admin, password: admin123)
- `POST /api/v1/inventory/items` - Crear item
- `PUT /api/v1/inventory/items/:id` - Actualizar item
- `DELETE /api/v1/inventory/items/:id` - Eliminar item
- `POST /api/v1/inventory/items/:id/reserve` - Reservar stock
- `POST /api/v1/inventory/items/:id/release` - Liberar stock
- `POST /api/v1/inventory/items/:id/adjust` - Ajustar stock

#### Query Service (http://localhost:8081)

- `GET /api/v1/inventory/items` - Listar items (paginado)
- `GET /api/v1/inventory/items/:id` - Obtener item por ID
- `GET /api/v1/inventory/items/sku/:sku` - Obtener item por SKU
- `GET /api/v1/inventory/items/:id/stock` - Obtener estado de stock

### Flujo de Operación

1. **Cliente** → POST /api/v1/inventory/items (Command Service)
2. **Command Service** → Valida y publica evento en Kafka
3. **Command Service** → Retorna 202 Accepted (asíncrono)
4. **Listener Service** → Consume evento de Kafka
5. **Listener Service** → Actualiza SQLite (Single Writer)
6. **Cliente** → GET /api/v1/inventory/items (Query Service)
7. **Query Service** → Lee desde Redis Cache o Read Model
8. **Query Service** → Retorna datos (ultra baja latencia)

### Verificación de Componentes

#### Docker Compose

```bash
# Verificar estado
docker ps

# Ver logs
cd docker-components/kafka
docker compose logs

cd docker-components/redis
docker compose logs
```

#### Servicios Go

```bash
# Verificar que estén corriendo
curl http://localhost:8080/api/v1/health
curl http://localhost:8081/api/v1/health

# Ver logs en las ventanas/terminales de cada servicio
```

#### Kafka

```bash
# Ver topics
docker exec -it kafka kafka-topics --list --bootstrap-server localhost:9092

# Ver mensajes en un topic
docker exec -it kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic inventory.items --from-beginning
```

#### Base de Datos SQLite

```bash
# Verificar que el archivo existe
ls -la prototipes/listener-service/inventory.db

# Ver contenido (requiere sqlite3)
sqlite3 prototipes/listener-service/inventory.db "SELECT * FROM inventory_items;"
```

### Pruebas Rápidas

#### 1. Crear un Item

```bash
# Autenticarse primero
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'

# Guardar el token de la respuesta

# Crear item
curl -X POST http://localhost:8080/api/v1/inventory/items \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer [TOKEN]" \
  -d '{
    "sku": "TEST-001",
    "name": "Producto de Prueba",
    "description": "Descripción del producto",
    "quantity": 100
  }'
```

#### 2. Consultar Stock

```bash
# Consultar por SKU
curl http://localhost:8081/api/v1/inventory/items/sku/TEST-001 \
  -H "Authorization: Bearer [TOKEN]"
```

#### 3. Ver Eventos en Kafdrop

- Abrir: http://localhost:9000
- Seleccionar topic: `inventory.items` o `inventory.stock`
- Ver mensajes en tiempo real

---

## 🧪 Pruebas de Endpoints

### Script de Pruebas Automatizado

El proyecto incluye un script bash que prueba todos los endpoints disponibles en el Dashboard HTML:

**Ubicación:** `html/test-endpoints.sh`

**Ejecutar:**
```bash
cd html
./test-endpoints.sh
```

**Requisitos:**
- Servicios corriendo (Command Service, Query Service, Dashboard Server)
- `jq` instalado (para parsear JSON)

**Casos de uso probados:**
1. Autenticación (POST /api/v1/auth/login)
2. Listar Items (GET /api/v1/inventory/items)
3. Crear Item (POST /api/v1/inventory/items)
4. Buscar por SKU (GET /api/v1/inventory/items/sku/:sku)
5. Obtener por ID (GET /api/v1/inventory/items/:id)
6. Actualizar Item (PUT /api/v1/inventory/items/:id)
7. Reservar Stock (POST /api/v1/inventory/items/:id/reserve)
8. Liberar Stock (POST /api/v1/inventory/items/:id/release)
9. Ajustar Stock (POST /api/v1/inventory/items/:id/adjust)
10. Eliminar Item (DELETE /api/v1/inventory/items/:id)

**Documentación:** Ver `html/TEST_ENDPOINTS.md` para más detalles.

### Tests Unitarios del Proxy

El proyecto incluye tests unitarios para el servidor proxy del Dashboard:

**Ubicación:** `html/server_test.go`

**Ejecutar:**
```bash
cd html
go test -v
```

**Documentación:** Ver `html/TESTING.md` para más detalles.

---

## 📚 Documentación Adicional

- **Arquitectura:** Ver `docs/ENTREGABLE.MD`
- **Dashboard:** Ver `html/README.md`
- **Command Service:** Ver `prototipes/command-service/README.md`
- **Query Service:** Ver `prototipes/query-service/README.md`
- **Listener Service:** Ver `prototipes/listener-service/README.md`
- **Configuración Kafka:** Ver `prototipes/CONFIGURACION_KAFKA.md`
- **Pruebas de Endpoints:** Ver `html/TEST_ENDPOINTS.md`
- **Tests Unitarios:** Ver `html/TESTING.md`

---

## 🎯 Resumen Rápido

### Inicio Rápido (1 comando)

```bash
# Windows
run.bat

# macOS/Linux
./run.sh
```

### Verificación Rápida

```bash
# Servicios Docker
docker ps

# Servicios Go
curl http://localhost:8080/api/v1/health
curl http://localhost:8081/api/v1/health

# Dashboard
open http://localhost:8000/index.html
```

### Detener Todo

1. Cerrar ventanas/terminales de servicios Go
2. `cd docker-components/kafka && docker compose down`
3. `cd docker-components/redis && docker compose down`

---

## 📞 Soporte

Si encuentra problemas durante la ejecución:

1. Revise la sección [Solución de Problemas](#-solución-de-problemas)
2. Verifique los logs de cada servicio
3. Verifique que todos los prerrequisitos estén instalados
4. Consulte la documentación adicional en `docs/` y `html/`

---

**Última actualización:** 2025  
**Versión del proyecto:** 1.0
