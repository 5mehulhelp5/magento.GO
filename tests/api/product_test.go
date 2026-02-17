package apitest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	productApi "magento.GO/api/product"
	productEntity "magento.GO/model/entity/product"
)

func productTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&productEntity.Product{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestProductAPI_List(t *testing.T) {
	e := echo.New()
	db := productTestDB(t)
	api := e.Group("/api")
	productApi.RegisterProductRoutes(api, db)

	req := httptest.NewRequest(http.MethodGet, "/api/products", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /api/products status = %d, want 200", rec.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["products"] == nil {
		t.Error("products key missing")
	}
}

func TestProductAPI_InvalidID(t *testing.T) {
	e := echo.New()
	db := productTestDB(t)
	api := e.Group("/api")
	productApi.RegisterProductRoutes(api, db)

	req := httptest.NewRequest(http.MethodGet, "/api/products/abc", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("GET /api/products/abc status = %d, want 400", rec.Code)
	}
}

func TestProductAPI_Create(t *testing.T) {
	e := echo.New()
	db := productTestDB(t)
	api := e.Group("/api")
	productApi.RegisterProductRoutes(api, db)

	body := map[string]interface{}{
		"AttributeSetID": 1,
		"TypeID":         "simple",
		"SKU":            "TEST-API-SKU",
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/products", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("POST /api/products status = %d, want 201", rec.Code)
	}
}

func TestProductAPI_CreateAndListAndGetByID_DataCheck(t *testing.T) {
	e := echo.New()
	db := productTestDB(t)
	api := e.Group("/api")
	productApi.RegisterProductRoutes(api, db)

	// Create product
	createBody := map[string]interface{}{
		"AttributeSetID": 1,
		"TypeID":         "simple",
		"SKU":            "DATA-CHECK-SKU",
	}
	createBytes, _ := json.Marshal(createBody)
	createReq := httptest.NewRequest(http.MethodPost, "/api/products", bytes.NewReader(createBytes))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	e.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", createRec.Code)
	}

	// List and check data
	listReq := httptest.NewRequest(http.MethodGet, "/api/products", nil)
	listRec := httptest.NewRecorder()
	e.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200", listRec.Code)
	}

	var listResp map[string]interface{}
	if err := json.NewDecoder(listRec.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	count, ok := listResp["count"].(float64)
	if !ok || count != 1 {
		t.Errorf("count = %v, want 1", listResp["count"])
	}
	products, ok := listResp["products"].([]interface{})
	if !ok || len(products) != 1 {
		t.Fatalf("products = %v, want len 1", listResp["products"])
	}
	prod := products[0].(map[string]interface{})
	if sku, _ := prod["SKU"].(string); sku != "DATA-CHECK-SKU" {
		t.Errorf("products[0].SKU = %v, want DATA-CHECK-SKU", prod["SKU"])
	}
	if typeID, _ := prod["TypeID"].(string); typeID != "simple" {
		t.Errorf("products[0].TypeID = %v, want simple", prod["TypeID"])
	}

	// Get by ID and check data
	entityID := int(prod["EntityID"].(float64))
	getReq := httptest.NewRequest(http.MethodGet, "/api/products/"+strconv.Itoa(entityID), nil)
	getRec := httptest.NewRecorder()
	e.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want 200", getRec.Code)
	}
	var getResp map[string]interface{}
	if err := json.NewDecoder(getRec.Body).Decode(&getResp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	gotProd := getResp["product"].(map[string]interface{})
	if sku, _ := gotProd["SKU"].(string); sku != "DATA-CHECK-SKU" {
		t.Errorf("product.SKU = %v, want DATA-CHECK-SKU", gotProd["SKU"])
	}
}
