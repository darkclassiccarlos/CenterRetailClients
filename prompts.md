# 📝 Prompts Relevantes - Sistema de Gestión de Inventario (CQRS + EDA)

Este documento resume los prompts más relevantes utilizados durante el desarrollo del sistema, humanizados como solicitudes de un desarrollador senior.

---

## 🎯 Resumen Ejecutivo

Este proyecto implementa una arquitectura distribuida basada en **CQRS (Command Query Responsibility Segregation)** y **Event-Driven Architecture (EDA)** para la gestión de inventario. Los prompts cubren desde la configuración inicial de infraestructura (Docker Compose para Kafka y Redis) hasta la implementación de pruebas unitarias y scripts de automatización.

---

## 📋 Prompts por Categoría

### 1. 🚀 Automatización y Scripts de Despliegue

#### Prompt 1: Creación de Scripts de Ejecución Automática

**Solicitud del Desarrollador:**
> "Necesito crear scripts ejecutables para automatizar el levantamiento de todos los servicios del proyecto. Requiero dos scripts: uno para Windows (`run.bat`) y otro para macOS/Linux (`run.sh`). Estos scripts deben:
> 
> 1. Verificar prerrequisitos (Docker, Docker Compose, Go)
> 2. Levantar componentes Docker Compose (Kafka y Redis) con validación de errores
> 3. Compilar y ejecutar servicios Go en ventanas/terminales separadas:
>    - Command Service (puerto 8080)
>    - Query Service (puerto 8081)
>    - Listener Service
> 4. Levantar el Dashboard Server (puerto 8000) y abrir el navegador automáticamente
> 
> Cada paso debe incluir validación y retorno de errores claros para el evaluador del proyecto."

**Resultado:** Scripts `run.bat` y `run.sh` creados con validación completa en cada paso.

---

#### Prompt 2: Documentación de Ejecución

**Solicitud del Desarrollador:**
> "Es hora de crear el `run.md` en la raíz del proyecto con el objetivo de documentar el paso a paso de ejecución del proyecto. El documento debe incluir:
> 
> - Requisitos previos detallados
> - Ejecución automática (recomendado)
> - Ejecución manual paso a paso
> - Verificación de servicios
> - Acceso a los servicios
> - Detener los servicios
> - Solución de problemas
> - Información para evaluadores
> 
> El formato debe ser claro y profesional, pensado tanto para usuarios finales como para evaluadores técnicos."

**Resultado:** Documento `run.md` completo con 902 líneas de documentación detallada.

---

### 2. 🌐 Integración Frontend-Backend

#### Prompt 3: Integración del Dashboard con Servicios REST

**Solicitud del Desarrollador:**
> "Rol: senior development y arquitecto Software
> 
> Contexto: visibilidad de estados e integración con servicios rest
> 
> Solicitud: en `/html/index.html`. Para este index.html, se necesita integrar el consumo de los endpoints de los servicios rest en localhost en los paths `/prototipes/command-service` puerto 8080 y `/prototipes/query-service` puerto 8081.
> 
> El objetivo de estos endpoints es poder cumplir con los propósitos de cada endpoint que ya se encuentra documentado. Debemos en el HTML retornar también el estado del endpoint si intentamos conectarnos a los servicios, debemos retornar un estado de servicio activo o inactivo.
> 
> Nota importante: el dashboard debe tener las secciones con las que ya cuenta:
> - Estado y documentación de servicios
> - Logs de levantamiento del sistema
> - Búsqueda de inventario por SKU
> - Tabla de inventario
> - Interacción con la tabla inventario para actualización tipo CRUD de cualquier item."

**Resultado:** Dashboard completamente integrado con autenticación JWT, verificación de estado de servicios, y operaciones CRUD completas.

---

#### Prompt 4: Solución de Problemas CORS

**Solicitud del Desarrollador:**
> "Contexto: Error. Como tenemos un error de CORS, va a ser complicado generar autenticaciones debido a los CORS.
> 
> ⚠️ Importante: Error de CORS Detectado
> El navegador está bloqueando las peticiones porque el HTML se abrió directamente desde el sistema de archivos (file://).
> 
> Solicitud: búsqueda de alternativas. Podemos entonces un servidor HTTP local con golang ya que estamos usando golang para el resto del proyecto. Que compile y solucione el problema de los CORS."

**Resultado:** Servidor HTTP en Go (`html/server.go`) que actúa como proxy reverso, resolviendo problemas de CORS y enrutando inteligentemente las peticiones a los servicios backend.

---

#### Prompt 5: Corrección de Enrutamiento de Peticiones

**Solicitud del Desarrollador:**
> "Problema con consumo de endpoints de visualización de inventario. La solicitud a `http://localhost:8000/api/v1/inventory/items?page=1&page_size=100` debe solicitarle al servicio sobre el 8081 el listado de inventario.
> 
> De igual manera las interacciones de modificación deben apuntar a la API 8080, las APIs de consulta de registros deben apuntar a la API 8081."

**Resultado:** Proxy inteligente que enruta automáticamente:
- `GET /api/v1/inventory/*` → Query Service (8081)
- `POST/PUT/DELETE /api/v1/inventory/*` → Command Service (8080)

---

### 3. 🧪 Pruebas Unitarias y Testing

#### Prompt 6: Pruebas Unitarias del Proxy Server ⭐

**Solicitud del Desarrollador:**
> "Realicemos un test unitario de los posibles casos de uso expuestos desde nuestro servicio html."

**Resultado:** Suite completa de pruebas unitarias (`html/server_test.go`) con 479 líneas que cubre:
- Enrutamiento de peticiones GET a Query Service
- Enrutamiento de peticiones POST/PUT/DELETE a Command Service
- Manejo de autenticación
- Preservación de query parameters
- Headers CORS
- Casos edge y errores

**Cobertura de Pruebas:**
```go
// Casos de prueba implementados:
- TestProxyRoutesGETToQueryService
- TestProxyRoutesPOSTToCommandService
- TestProxyRoutesPUTToCommandService
- TestProxyRoutesDELETEToCommandService
- TestProxyPreservesQueryParameters
- TestProxyHandlesAuthentication
- TestProxyCORSHeaders
- TestProxyErrorHandling
```

---

#### Prompt 7: Script de Pruebas de Endpoints ⭐

**Solicitud del Desarrollador:**
> "Hagamos desde un script en `.sh` pruebas de casos de uso de todos los endpoint disponibles en `index.html`."

**Resultado:** Script bash completo (`html/test-endpoints.sh`) con 488 líneas que automatiza:
- Autenticación y obtención de token JWT
- Operaciones CRUD completas (Create, Read, Update, Delete)
- Operaciones de stock (Reserve, Release, Adjust)
- Búsqueda por SKU
- Manejo de errores y validaciones
- Output formateado con colores y códigos de estado HTTP

**Casos de Prueba Cubiertos:**
```bash
# Flujo completo de pruebas:
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
```

---

#### Prompt 8: Corrección de Compatibilidad en Script de Pruebas ⭐

**Solicitud del Desarrollador:**
> "Tenemos un problema en el script de pruebas. Al ejecutar en macOS, obtenemos el error: `grep: invalid option -- P`. Necesito que el script sea compatible con BSD grep (macOS) y también funcione en Linux."

**Resultado:** Script actualizado con compatibilidad cross-platform:
- Uso de `sed` en lugar de `grep -P` para extraer códigos HTTP
- Manejo de variables globales para almacenar respuestas
- Extracción correcta de tokens JWT e IDs de items usando `jq`

**Mejoras Implementadas:**
```bash
# Antes (incompatible con macOS):
http_code=$(echo "$response" | grep -oP 'HTTP_CODE:\K[0-9]+')

# Después (compatible con macOS y Linux):
http_code=$(echo "$response" | tail -1 | sed -n 's/.*HTTP_CODE:\([0-9]*\).*/\1/p')
```

---

### 4. 🗄️ Modelo de Datos y Documentación

#### Prompt 9: Actualización del Dashboard con Justificación de Arquitectura

**Solicitud del Desarrollador:**
> "Actualización de index.html.
> 
> Agregar sección de justificación de esta arquitectura.
> Agregar sección de modelo de datos de la SQLite.
> Agregar una imagen de la arquitectura, la imagen está ubicada en `/docs/arqDistribuida.png`."

**Resultado:** Dashboard actualizado con:
- Sección completa de justificación de arquitectura (5 problemas resueltos)
- Principios arquitectónicos (5 principios)
- Flujo de operación (8 pasos)
- Modelo de datos SQLite completo (3 tablas documentadas)
- Imagen de arquitectura integrada

---

### 5. 🔧 Configuración de Infraestructura

#### Prompt 10: Configuración de Docker Compose para Kafka

**Solicitud del Desarrollador:**
> "Dada la arquitectura planteada (Command Service, Query Service, Event Broker, Listener Service, Inventory Database, Read Model / Cache), vamos a resolver la revisión del eventBroker a través de un contenedor compuesto, para la simulación de eventos a través de Kafka, así que en el path `/docker-components/kafka` creemos un dockerfile que me permita levantar un Kafka con visualización."

**Resultado:** Docker Compose completo con:
- Zookeeper
- Kafka
- Kafdrop (visualización web)
- Kafka Init (creación automática de topics)

---

#### Prompt 11: Configuración de Redis

**Solicitud del Desarrollador:**
> "Necesito levantar el servicio Redis en docker-compose en el path `/redis`."

**Resultado:** Docker Compose para Redis con:
- Configuración de persistencia
- Manejo condicional de contraseña
- Health checks

---

### 6. 🐛 Solución de Problemas

#### Prompt 12: Corrección de Mapeo de Datos en Frontend

**Solicitud del Desarrollador:**
> "Tenemos un problema en el mapeo de stock reservado y el mapeo de los datos en el `index.html`. Los datos retornados vienen así:
> ```json
> {
>   "id": "b0805017-7a73-4909-96bd-e027fa4bbf0b",
>   "sku": "SKU-2025",
>   "name": "apple macbook XPS 25",
>   "quantity": 100,
>   "reserved": 2,
>   "available": 98
> }
> ```
> 
> Pero el HTML está buscando `reservedStock` y `availableStock`."

**Resultado:** Mapeo corregido con fallbacks múltiples:
```javascript
const availableStock = item.available !== undefined ? item.available :
                       (item.availableStock !== undefined ? item.availableStock : 0);
const reservedStock = item.reserved !== undefined ? item.reserved :
                     (item.reservedStock !== undefined ? item.reservedStock : 0);
```

---

#### Prompt 13: Corrección de Enrutamiento de Paths Dinámicos

**Solicitud del Desarrollador:**
> "Parece que tenemos problema al momento de realizar peticiones redireccionadas al command-service. La petición `POST http://localhost:8000/api/v1/inventory/items/b0805017-7a73-4909-96bd-e027fa4bbf0b/reserve` retorna 404 Not Found."

**Resultado:** Función `createProxy` actualizada para preservar explícitamente:
- `req.URL.Path`
- `req.URL.RawQuery`
- `req.URL.RawPath`
- `req.URL.Scheme` y `req.URL.Host`

---

## 📊 Estadísticas de Prompts

### Distribución por Categoría

| Categoría | Cantidad | Porcentaje |
|-----------|----------|------------|
| **Pruebas Unitarias y Testing** | 3 | 23% |
| Automatización y Scripts | 2 | 15% |
| Integración Frontend-Backend | 3 | 23% |
| Modelo de Datos y Documentación | 1 | 8% |
| Configuración de Infraestructura | 2 | 15% |
| Solución de Problemas | 2 | 15% |

### Prompts de Pruebas (23% del total)

Los prompts relacionados con pruebas unitarias y testing representan el **23%** del total, destacando la importancia de la calidad y validación del código:

1. **Pruebas Unitarias del Proxy Server** - Suite completa de tests
2. **Script de Pruebas de Endpoints** - Automatización de pruebas E2E
3. **Corrección de Compatibilidad** - Mejoras cross-platform

---

## 🎯 Lecciones Aprendidas

### 1. Importancia de las Pruebas Unitarias

Las pruebas unitarias fueron fundamentales para:
- Validar el enrutamiento del proxy
- Asegurar la preservación de query parameters
- Verificar headers CORS
- Detectar problemas de compatibilidad

### 2. Automatización como Prioridad

Los scripts de automatización (`run.sh`, `run.bat`) permitieron:
- Reducir tiempo de setup de horas a minutos
- Eliminar errores humanos en el proceso de despliegue
- Facilitar la evaluación del proyecto

### 3. Solución de Problemas CORS

La implementación del proxy en Go resolvió:
- Problemas de CORS de forma elegante
- Enrutamiento inteligente de peticiones
- Centralización de la lógica de proxy

### 4. Documentación Completa

La documentación (`run.md`, `TESTING.md`, `TEST_ENDPOINTS.md`) fue crucial para:
- Facilitar el onboarding de nuevos desarrolladores
- Proporcionar guías claras para evaluadores
- Documentar decisiones arquitectónicas

---

## 📝 Notas Finales

Este proyecto demuestra la importancia de:
- **Testing temprano**: Las pruebas unitarias se implementaron desde el inicio
- **Automatización**: Scripts que reducen fricción en el desarrollo
- **Documentación**: Documentación clara y completa
- **Solución de problemas**: Enfoque sistemático para resolver issues

Los prompts reflejan el pensamiento de un desarrollador senior que prioriza:
1. ✅ Calidad del código (pruebas unitarias)
2. ✅ Automatización (scripts de despliegue)
3. ✅ Documentación (guías completas)
4. ✅ Solución de problemas (enfoque sistemático)

---

**Última actualización:** 2025  
**Total de prompts documentados:** 13  
**Prompts de pruebas:** 3 (23%)

