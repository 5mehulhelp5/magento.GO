package apitest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"magento.GO/api"
	"magento.GO/core/registry"
)

func TestMockRoute_Health(t *testing.T) {
	registry.GlobalRegistry.UnlockForTesting(registry.KeyRegistryRoutes)
	defer registry.GlobalRegistry.Lock(registry.KeyRegistryRoutes)

	api.RegisterGET("/mock/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status": "ok",
			"mock":   true,
		})
	})

	e := echo.New()
	api.ApplyRoutes(e, nil)

	req := httptest.NewRequest(http.MethodGet, "/mock/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /mock/health status = %d, want 200", rec.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("status = %v, want ok", resp["status"])
	}
	if resp["mock"] != true {
		t.Errorf("mock = %v, want true", resp["mock"])
	}
}

func TestMockRoute_Products(t *testing.T) {
	registry.GlobalRegistry.UnlockForTesting(registry.KeyRegistryRoutes)
	defer registry.GlobalRegistry.Lock(registry.KeyRegistryRoutes)

	mockProducts := []map[string]interface{}{
		{"id": 1, "sku": "MOCK-SKU-1", "name": "Mock Product 1"},
		{"id": 2, "sku": "MOCK-SKU-2", "name": "Mock Product 2"},
	}
	api.RegisterGET("/mock/products", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"products": mockProducts,
			"count":    len(mockProducts),
		})
	})

	e := echo.New()
	api.ApplyRoutes(e, nil)

	req := httptest.NewRequest(http.MethodGet, "/mock/products", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /mock/products status = %d, want 200", rec.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	products, ok := resp["products"].([]interface{})
	if !ok || len(products) != 2 {
		t.Errorf("products = %v, want 2 items", resp["products"])
	}
	if resp["count"] != float64(2) {
		t.Errorf("count = %v, want 2", resp["count"])
	}
}

func TestMockRoute_NotFound(t *testing.T) {
	e := echo.New()
	api.ApplyRoutes(e, nil)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent/route", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /nonexistent/route status = %d, want 404", rec.Code)
	}
}
