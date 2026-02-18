package api

import (
	"sync"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"magento.GO/core/registry"
)

var mu sync.Mutex

// --- /api group modules (authenticated, DB-dependent) ---

// ModuleFunc registers routes on the /api group with DB access.
type ModuleFunc func(g *echo.Group, db *gorm.DB)

func getModules() []ModuleFunc {
	if v, ok := registry.GlobalRegistry.GetGlobal(registry.KeyRegistryAPI); ok && v != nil {
		return v.([]ModuleFunc)
	}
	return nil
}

// RegisterModule registers an API module. Call from init() in API packages.
func RegisterModule(fn ModuleFunc) {
	mu.Lock()
	defer mu.Unlock()
	if registry.GlobalRegistry.IsLocked(registry.KeyRegistryAPI) {
		panic("api/registry: API modules locked (register only during init)")
	}
	list := getModules()
	list = append(list, fn)
	registry.GlobalRegistry.SetGlobal(registry.KeyRegistryAPI, list)
}

// ApplyModules calls all registered /api modules. Locks the registry.
func ApplyModules(g *echo.Group, db *gorm.DB) {
	for _, fn := range getModules() {
		fn(g, db)
	}
	registry.GlobalRegistry.Lock(registry.KeyRegistryAPI)
}

// --- Root-level routes (public: health, custom, HTML, etc.) ---

// RouteFunc registers routes on the root Echo instance.
type RouteFunc func(e *echo.Echo, db *gorm.DB)

func getRoutes() []RouteFunc {
	if v, ok := registry.GlobalRegistry.GetGlobal(registry.KeyRegistryRoutes); ok && v != nil {
		return v.([]RouteFunc)
	}
	return nil
}

// RegisterRoute registers a root-level route module. Call from init().
func RegisterRoute(fn RouteFunc) {
	mu.Lock()
	defer mu.Unlock()
	if registry.GlobalRegistry.IsLocked(registry.KeyRegistryRoutes) {
		panic("api/registry: routes locked (register only during init)")
	}
	list := getRoutes()
	list = append(list, fn)
	registry.GlobalRegistry.SetGlobal(registry.KeyRegistryRoutes, list)
}

// RegisterGET is shorthand for registering a simple GET route on root.
func RegisterGET(path string, handler echo.HandlerFunc) {
	RegisterRoute(func(e *echo.Echo, _ *gorm.DB) {
		e.GET(path, handler)
	})
}

// RegisterPOST is shorthand for registering a simple POST route on root.
func RegisterPOST(path string, handler echo.HandlerFunc) {
	RegisterRoute(func(e *echo.Echo, _ *gorm.DB) {
		e.POST(path, handler)
	})
}

// RegisterHTMLModule registers an HTML route module (alias for RegisterRoute).
func RegisterHTMLModule(fn RouteFunc) {
	RegisterRoute(fn)
}

// ApplyRoutes calls all registered root-level routes. Locks the registry.
func ApplyRoutes(e *echo.Echo, db *gorm.DB) {
	for _, fn := range getRoutes() {
		fn(e, db)
	}
	registry.GlobalRegistry.Lock(registry.KeyRegistryRoutes)
}
