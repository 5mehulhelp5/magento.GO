package apitest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	_ "magento.GO/custom"
	"magento.GO/api"
)

func TestCustomRoute_Ping(t *testing.T) {
	e := echo.New()
	api.Apply(e)

	req := httptest.NewRequest(http.MethodGet, "/custom/ping", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /custom/ping status = %d, want 200", rec.Code)
	}
	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["pong"] != "ok" {
		t.Errorf("pong = %q, want ok", resp["pong"])
	}
}
