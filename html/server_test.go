package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"strings"
	"testing"
)

// TestProxyRouting_GET_InventoryItems verifica que las peticiones GET a /api/v1/inventory/items
// se redirijan al Query Service (8081)
func TestProxyRouting_GET_InventoryItems(t *testing.T) {
	// Crear un servidor mock para Query Service
	queryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/inventory/items" {
			t.Errorf("Expected path /api/v1/inventory/items, got %s", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("Expected method GET, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"items":[]}`))
	}))
	defer queryServer.Close()

	// Crear un servidor mock para Command Service (no debería ser llamado)
	commandServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Command Service should not be called for GET requests")
	}))
	defer commandServer.Close()

	// Crear el proxy con los servidores mock
	queryProxy := createProxy(queryServer.URL)
	commandProxy := createProxy(commandServer.URL)

	// Crear el handler del proxy
	handler := createProxyHandler(queryProxy, commandProxy)

	// Crear la petición
	req := httptest.NewRequest("GET", "/api/v1/inventory/items?page=1&page_size=100", nil)
	w := httptest.NewRecorder()

	// Ejecutar la petición
	handler.ServeHTTP(w, req)

	// Verificar la respuesta
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestProxyRouting_GET_InventoryItemsByID verifica que las peticiones GET a /api/v1/inventory/items/:id
// se redirijan al Query Service (8081)
func TestProxyRouting_GET_InventoryItemsByID(t *testing.T) {
	itemID := "b0805017-7a73-4909-96bd-e027fa4bbf0b"
	expectedPath := "/api/v1/inventory/items/" + itemID

	queryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"` + itemID + `"}`))
	}))
	defer queryServer.Close()

	commandServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Command Service should not be called for GET requests")
	}))
	defer commandServer.Close()

	queryProxy := createProxy(queryServer.URL)
	commandProxy := createProxy(commandServer.URL)
	handler := createProxyHandler(queryProxy, commandProxy)

	req := httptest.NewRequest("GET", expectedPath, nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestProxyRouting_GET_InventoryItemsBySKU verifica que las peticiones GET a /api/v1/inventory/items/sku/:sku
// se redirijan al Query Service (8081)
func TestProxyRouting_GET_InventoryItemsBySKU(t *testing.T) {
	sku := "SKU-2025"
	expectedPath := "/api/v1/inventory/items/sku/" + sku

	queryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"sku":"` + sku + `"}`))
	}))
	defer queryServer.Close()

	commandServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Command Service should not be called for GET requests")
	}))
	defer commandServer.Close()

	queryProxy := createProxy(queryServer.URL)
	commandProxy := createProxy(commandServer.URL)
	handler := createProxyHandler(queryProxy, commandProxy)

	req := httptest.NewRequest("GET", expectedPath, nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestProxyRouting_POST_CreateItem verifica que las peticiones POST a /api/v1/inventory/items
// se redirijan al Command Service (8080)
func TestProxyRouting_POST_CreateItem(t *testing.T) {
	commandServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/inventory/items" {
			t.Errorf("Expected path /api/v1/inventory/items, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Expected method POST, got %s", r.Method)
		}
		// Verificar el body
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "sku") {
			t.Error("Expected body to contain 'sku'")
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"test-id"}`))
	}))
	defer commandServer.Close()

	queryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Query Service should not be called for POST requests")
	}))
	defer queryServer.Close()

	queryProxy := createProxy(queryServer.URL)
	commandProxy := createProxy(commandServer.URL)
	handler := createProxyHandler(queryProxy, commandProxy)

	body := `{"sku":"SKU-001","name":"Test Item","quantity":100}`
	req := httptest.NewRequest("POST", "/api/v1/inventory/items", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}
}

// TestProxyRouting_POST_ReserveStock verifica que las peticiones POST a /api/v1/inventory/items/:id/reserve
// se redirijan al Command Service (8080) y preserven la ruta completa
func TestProxyRouting_POST_ReserveStock(t *testing.T) {
	itemID := "b0805017-7a73-4909-96bd-e027fa4bbf0b"
	expectedPath := "/api/v1/inventory/items/" + itemID + "/reserve"

	commandServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Expected method POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"` + itemID + `","reserved":2}`))
	}))
	defer commandServer.Close()

	queryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Query Service should not be called for POST requests")
	}))
	defer queryServer.Close()

	queryProxy := createProxy(queryServer.URL)
	commandProxy := createProxy(commandServer.URL)
	handler := createProxyHandler(queryProxy, commandProxy)

	body := `{"quantity":2}`
	req := httptest.NewRequest("POST", expectedPath, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestProxyRouting_POST_ReleaseStock verifica que las peticiones POST a /api/v1/inventory/items/:id/release
// se redirijan al Command Service (8080) y preserven la ruta completa
func TestProxyRouting_POST_ReleaseStock(t *testing.T) {
	itemID := "b0805017-7a73-4909-96bd-e027fa4bbf0b"
	expectedPath := "/api/v1/inventory/items/" + itemID + "/release"

	commandServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"` + itemID + `","reserved":0}`))
	}))
	defer commandServer.Close()

	queryProxy := createProxy("http://localhost:8081")
	commandProxy := createProxy(commandServer.URL)
	handler := createProxyHandler(queryProxy, commandProxy)

	body := `{"quantity":2}`
	req := httptest.NewRequest("POST", expectedPath, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestProxyRouting_POST_AdjustStock verifica que las peticiones POST a /api/v1/inventory/items/:id/adjust
// se redirijan al Command Service (8080) y preserven la ruta completa
func TestProxyRouting_POST_AdjustStock(t *testing.T) {
	itemID := "b0805017-7a73-4909-96bd-e027fa4bbf0b"
	expectedPath := "/api/v1/inventory/items/" + itemID + "/adjust"

	commandServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"` + itemID + `","quantity":102}`))
	}))
	defer commandServer.Close()

	queryProxy := createProxy("http://localhost:8081")
	commandProxy := createProxy(commandServer.URL)
	handler := createProxyHandler(queryProxy, commandProxy)

	body := `{"quantity":2}`
	req := httptest.NewRequest("POST", expectedPath, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestProxyRouting_PUT_UpdateItem verifica que las peticiones PUT a /api/v1/inventory/items/:id
// se redirijan al Command Service (8080)
func TestProxyRouting_PUT_UpdateItem(t *testing.T) {
	itemID := "b0805017-7a73-4909-96bd-e027fa4bbf0b"
	expectedPath := "/api/v1/inventory/items/" + itemID

	commandServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}
		if r.Method != "PUT" {
			t.Errorf("Expected method PUT, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"` + itemID + `"}`))
	}))
	defer commandServer.Close()

	queryProxy := createProxy("http://localhost:8081")
	commandProxy := createProxy(commandServer.URL)
	handler := createProxyHandler(queryProxy, commandProxy)

	body := `{"name":"Updated Name"}`
	req := httptest.NewRequest("PUT", expectedPath, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestProxyRouting_DELETE_DeleteItem verifica que las peticiones DELETE a /api/v1/inventory/items/:id
// se redirijan al Command Service (8080)
func TestProxyRouting_DELETE_DeleteItem(t *testing.T) {
	itemID := "b0805017-7a73-4909-96bd-e027fa4bbf0b"
	expectedPath := "/api/v1/inventory/items/" + itemID

	commandServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}
		if r.Method != "DELETE" {
			t.Errorf("Expected method DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"item deleted successfully"}`))
	}))
	defer commandServer.Close()

	queryProxy := createProxy("http://localhost:8081")
	commandProxy := createProxy(commandServer.URL)
	handler := createProxyHandler(queryProxy, commandProxy)

	req := httptest.NewRequest("DELETE", expectedPath, nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestProxyRouting_POST_Login verifica que las peticiones POST a /api/v1/auth/login
// se redirijan al Command Service (8080)
func TestProxyRouting_POST_Login(t *testing.T) {
	commandServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/login" {
			t.Errorf("Expected path /api/v1/auth/login, got %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("Expected method POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"token":"test-token","expires_in":600}`))
	}))
	defer commandServer.Close()

	queryProxy := createProxy("http://localhost:8081")
	commandProxy := createProxy(commandServer.URL)
	handler := createProxyHandler(queryProxy, commandProxy)

	body := `{"username":"admin","password":"admin123"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestProxyRouting_QueryParamsPreservation verifica que los query params se preserven correctamente
func TestProxyRouting_QueryParamsPreservation(t *testing.T) {
	queryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "page=1&page_size=100" {
			t.Errorf("Expected query params 'page=1&page_size=100', got '%s'", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"items":[]}`))
	}))
	defer queryServer.Close()

	queryProxy := createProxy(queryServer.URL)
	commandProxy := createProxy("http://localhost:8080")
	handler := createProxyHandler(queryProxy, commandProxy)

	req := httptest.NewRequest("GET", "/api/v1/inventory/items?page=1&page_size=100", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestCORSHeaders verifica que los headers CORS se agreguen correctamente
func TestCORSHeaders(t *testing.T) {
	queryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"items":[]}`))
	}))
	defer queryServer.Close()

	queryProxy := createProxy(queryServer.URL)
	commandProxy := createProxy("http://localhost:8080")
	handler := createProxyHandler(queryProxy, commandProxy)

	req := httptest.NewRequest("GET", "/api/v1/inventory/items", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Verificar headers CORS
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Expected Access-Control-Allow-Origin: *, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Expected Access-Control-Allow-Methods header")
	}
}

// TestCORS_PreflightRequest verifica que las peticiones OPTIONS (preflight) se manejen correctamente
func TestCORS_PreflightRequest(t *testing.T) {
	queryProxy := createProxy("http://localhost:8081")
	commandProxy := createProxy("http://localhost:8080")
	handler := createProxyHandler(queryProxy, commandProxy)

	req := httptest.NewRequest("OPTIONS", "/api/v1/inventory/items", nil)
	req.Header.Set("Origin", "http://localhost:8000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for OPTIONS request, got %d", w.Code)
	}

	// Verificar headers CORS
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Expected Access-Control-Allow-Origin: *, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

// createProxyHandler crea un handler de prueba que simula el comportamiento del servidor
func createProxyHandler(queryProxy, commandProxy *httputil.ReverseProxy) http.Handler {
	mux := http.NewServeMux()

	// Health checks específicos
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
		path := r.URL.Path
		method := r.Method

		// Health check genérico
		if path == "/api/v1/health" {
			commandProxy.ServeHTTP(w, r)
			return
		}

		// Rutas de consulta (GET) van a Query Service
		if method == "GET" && strings.HasPrefix(path, "/api/v1/inventory/items") {
			queryProxy.ServeHTTP(w, r)
			return
		}

		// Rutas de autenticación van a Command Service
		if strings.HasPrefix(path, "/api/v1/auth/") {
			commandProxy.ServeHTTP(w, r)
			return
		}

		// Todas las demás rutas van a Command Service
		commandProxy.ServeHTTP(w, r)
	})

	return corsMiddleware(mux)
}
