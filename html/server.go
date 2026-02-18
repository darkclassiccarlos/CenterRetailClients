package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	CommandServiceURL = "http://localhost:8080"
	QueryServiceURL   = "http://localhost:8081"
)

func main() {
	// Configuración de flags
	port := flag.String("port", "8000", "Puerto del servidor HTTP")
	dir := flag.String("dir", ".", "Directorio a servir (por defecto: directorio actual)")
	flag.Parse()

	// Obtener el directorio absoluto
	absDir, err := filepath.Abs(*dir)
	if err != nil {
		log.Fatalf("Error al obtener ruta absoluta: %v", err)
	}

	// Verificar que el directorio existe
	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		log.Fatalf("El directorio %s no existe", absDir)
	}

	// Crear el file server
	fileServer := http.FileServer(http.Dir(absDir))

	// Crear los proxies
	commandProxy := createProxy(CommandServiceURL)
	queryProxy := createProxy(QueryServiceURL)

	// Crear el mux router
	mux := http.NewServeMux()

	// Proxy para health checks específicos
	mux.HandleFunc("/api/v1/health/command", func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = "/api/v1/health"
		commandProxy.ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/v1/health/query", func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = "/api/v1/health"
		queryProxy.ServeHTTP(w, r)
	})

	// Proxy para todas las peticiones /api/v1/
	mux.HandleFunc("/api/v1/", func(w http.ResponseWriter, r *http.Request) {
		// Determinar a qué servicio redirigir basado en la ruta y método
		path := r.URL.Path
		method := r.Method
		queryParams := r.URL.RawQuery

		// Health checks específicos ya están manejados arriba
		// Aquí solo manejamos el health check genérico
		if path == "/api/v1/health" {
			commandProxy.ServeHTTP(w, r)
			return
		}

		// Rutas de consulta (GET) van a Query Service (8081)
		if method == "GET" && strings.HasPrefix(path, "/api/v1/inventory/items") {
			// Todas las consultas de inventario van a Query Service:
			// - GET /api/v1/inventory/items (con o sin query params como ?page=1&page_size=100)
			// - GET /api/v1/inventory/items/:id
			// - GET /api/v1/inventory/items/sku/:sku
			// - GET /api/v1/inventory/items/:id/stock
			// El proxy preserva automáticamente los query params
			log.Printf("🔍 [Proxy] GET %s?%s -> Query Service (8081)", path, queryParams)
			queryProxy.ServeHTTP(w, r)
			return
		}

		// Rutas de autenticación van a ambos servicios (pero por defecto Command Service)
		// El dashboard puede autenticarse con cualquiera de los dos
		if strings.HasPrefix(path, "/api/v1/auth/") {
			log.Printf("🔐 [Proxy] %s %s -> Command Service (8080)", method, path)
			commandProxy.ServeHTTP(w, r)
			return
		}

		// Todas las demás rutas (POST, PUT, DELETE, PATCH) van a Command Service (8080)
		// Esto incluye:
		// - POST /api/v1/inventory/items (crear)
		// - PUT /api/v1/inventory/items/:id (actualizar)
		// - DELETE /api/v1/inventory/items/:id (eliminar)
		// - POST /api/v1/inventory/items/:id/reserve (reservar stock)
		// - POST /api/v1/inventory/items/:id/release (liberar stock)
		// - POST /api/v1/inventory/items/:id/adjust (ajustar stock)
		log.Printf("✏️  [Proxy] %s %s -> Command Service (8080)", method, path)
		commandProxy.ServeHTTP(w, r)
	})

	// Servir archivos estáticos para todo lo demás
	mux.Handle("/", fileServer)

	// Handler con CORS habilitado
	handler := corsMiddleware(mux)

	// Crear el servidor HTTP
	server := &http.Server{
		Addr:         ":" + *port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Mensaje de inicio
	fmt.Println("🚀 Servidor HTTP local iniciado con proxy CORS")
	fmt.Printf("📁 Directorio: %s\n", absDir)
	fmt.Printf("🌐 URL: http://localhost:%s\n", *port)
	fmt.Printf("📄 Abre: http://localhost:%s/index.html\n", *port)
	fmt.Printf("🔗 Command Service Proxy: http://localhost:%s/command-api/\n", *port)
	fmt.Printf("🔗 Query Service Proxy: http://localhost:%s/query-api/\n", *port)
	fmt.Println("⚠️  Presiona Ctrl+C para detener el servidor")
	fmt.Println()

	// Iniciar el servidor
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}
}

// createProxy crea un proxy reverso para un servicio backend
func createProxy(targetURL string) *httputil.ReverseProxy {
	target, err := url.Parse(targetURL)
	if err != nil {
		log.Fatalf("Error al parsear URL del servicio: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// Modificar la respuesta para agregar headers CORS
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		// Preservar la ruta y query originales antes de que el director las modifique
		originalPath := req.URL.Path
		originalRawQuery := req.URL.RawQuery
		originalRawPath := req.URL.RawPath

		// Llamar al director original (esto configura Scheme, Host, etc.)
		originalDirector(req)

		// Restaurar la ruta original - esto es crítico para rutas con parámetros dinámicos
		req.URL.Path = originalPath
		req.URL.RawQuery = originalRawQuery
		if originalRawPath != "" {
			req.URL.RawPath = originalRawPath
		}

		// Configurar el host y scheme
		req.Host = target.Host
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host

		// Log para depuración
		log.Printf("🔗 [Proxy Director] %s %s?%s -> %s://%s%s?%s",
			req.Method, originalPath, originalRawQuery,
			req.URL.Scheme, req.URL.Host, req.URL.Path, req.URL.RawQuery)
	}

	// Modificar la respuesta
	proxy.ModifyResponse = func(resp *http.Response) error {
		// Agregar headers CORS a la respuesta
		resp.Header.Set("Access-Control-Allow-Origin", "*")
		resp.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		resp.Header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept, X-Request-ID")
		resp.Header.Set("Access-Control-Allow-Credentials", "true")
		return nil
	}

	return proxy
}

// corsMiddleware configura los headers CORS para permitir todas las peticiones
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Configurar headers CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept, X-Request-ID")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "3600")

		// Manejar preflight requests (OPTIONS)
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Continuar con el siguiente handler
		next.ServeHTTP(w, r)
	})
}
