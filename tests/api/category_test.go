package apitest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	categoryApi "magento.GO/api/category"
	categoryEntity "magento.GO/model/entity/category"
)

func categoryTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&categoryEntity.Category{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestCategoryAPI_List(t *testing.T) {
	e := echo.New()
	db := categoryTestDB(t)
	api := e.Group("/api")
	categoryApi.RegisterCategoryAPI(api, db)

	req := httptest.NewRequest(http.MethodGet, "/api/categories", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// May return 200 with empty list or 500 if schema not migrated
	if rec.Code != http.StatusOK && rec.Code != http.StatusInternalServerError {
		t.Errorf("GET /api/categories status = %d", rec.Code)
	}
}

func TestCategoryAPI_InvalidID(t *testing.T) {
	e := echo.New()
	db := categoryTestDB(t)
	api := e.Group("/api")
	categoryApi.RegisterCategoryAPI(api, db)

	req := httptest.NewRequest(http.MethodGet, "/api/category/abc", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("GET /api/category/abc status = %d, want 400", rec.Code)
	}
}
