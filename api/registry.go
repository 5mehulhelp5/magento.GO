package api

import (
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"

	"magento.GO/core/registry"
)

// Route holds method, path, and handler.
type Route struct {
	Method  string
	Path    string
	Handler echo.HandlerFunc
}

var mu sync.Mutex

func getRoutes() []Route {
	if v, ok := registry.GlobalRegistry.GetGlobal(registry.KeyRegistryAPI); ok && v != nil {
		return v.([]Route)
	}
	return nil
}

// Register adds an HTTP route. Call from init() in custom packages. Panics if registry is locked.
func Register(method, path string, handler echo.HandlerFunc) {
	mu.Lock()
	defer mu.Unlock()
	if registry.GlobalRegistry.IsLocked(registry.KeyRegistryAPI) {
		panic("api/registry: locked (register only during init before Apply)")
	}
	list := getRoutes()
	list = append(list, Route{Method: method, Path: path, Handler: handler})
	registry.GlobalRegistry.SetGlobal(registry.KeyRegistryAPI, list)
}

// RegisterGET is shorthand for Register(http.MethodGet, path, handler).
func RegisterGET(path string, handler echo.HandlerFunc) {
	Register(http.MethodGet, path, handler)
}

// RegisterPOST is shorthand for Register(http.MethodPost, path, handler).
func RegisterPOST(path string, handler echo.HandlerFunc) {
	Register(http.MethodPost, path, handler)
}

// Apply adds all registered routes to the Echo instance. Locks the api registry (immutable after).
func Apply(e *echo.Echo) {
	list := getRoutes()
	for _, r := range list {
		switch r.Method {
		case http.MethodGet:
			e.GET(r.Path, r.Handler)
		case http.MethodPost:
			e.POST(r.Path, r.Handler)
		case http.MethodPut:
			e.PUT(r.Path, r.Handler)
		case http.MethodDelete:
			e.DELETE(r.Path, r.Handler)
		default:
			e.Add(r.Method, r.Path, r.Handler)
		}
	}
	registry.GlobalRegistry.Lock(registry.KeyRegistryAPI)
}
