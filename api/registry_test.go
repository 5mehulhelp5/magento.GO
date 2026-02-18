package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestRegistry_Register_Apply(t *testing.T) {
	RegisterGET("/test/registry/check", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	e := echo.New()
	ApplyRoutes(e, nil)

	req := httptest.NewRequest(http.MethodGet, "/test/registry/check", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
