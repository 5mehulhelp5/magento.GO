package apitest

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/gorm"

	stockApi "magento.GO/api/stock"
	entity "magento.GO/model/entity"
	productEntity "magento.GO/model/entity/product"
)

const (
	testUser = "admin"
	testPass = "secret"
)

func stockTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("stock_api_test_%s_%d.db", t.Name(), time.Now().UnixNano()))
	t.Cleanup(func() { os.Remove(tmpFile) })
	db, err := gorm.Open(sqlite.Open(tmpFile), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA busy_timeout=5000")
	if err := db.AutoMigrate(
		&productEntity.Product{},
		&productEntity.StockItem{},
		&entity.EavAttribute{},
		&entity.OauthToken{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_stock_unq ON cataloginventory_stock_item (product_id, stock_id)")
	return db
}

func stockTestServer(t *testing.T, db *gorm.DB) *echo.Echo {
	t.Helper()
	e := echo.New()
	apiGroup := e.Group("/api")
	apiGroup.Use(middleware.BasicAuth(func(user, pass string, c echo.Context) (bool, error) {
		return user == testUser && pass == testPass, nil
	}))
	stockApi.RegisterStockRoutes(apiGroup, db)
	return e
}

func basicAuth(user, pass string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))
}

func doStockRequest(e *echo.Echo, body interface{}, auth string) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/stock/import", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// ---------- Auth tests ----------

func TestStockAPI_NoAuth_Returns401(t *testing.T) {
	db := stockTestDB(t)
	e := stockTestServer(t, db)

	body := map[string]interface{}{
		"items": []map[string]interface{}{
			{"sku": "X", "qty": 1},
		},
	}
	rec := doStockRequest(e, body, "")

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestStockAPI_WrongCredentials_Returns401(t *testing.T) {
	db := stockTestDB(t)
	e := stockTestServer(t, db)

	body := map[string]interface{}{
		"items": []map[string]interface{}{
			{"sku": "X", "qty": 1},
		},
	}
	rec := doStockRequest(e, body, basicAuth("wrong", "creds"))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestStockAPI_ValidAuth_Returns200(t *testing.T) {
	db := stockTestDB(t)
	p := productEntity.Product{SKU: "AUTH-OK", AttributeSetID: 4, TypeID: "simple"}
	db.Create(&p)

	e := stockTestServer(t, db)

	body := map[string]interface{}{
		"items": []map[string]interface{}{
			{"sku": "AUTH-OK", "qty": 10, "is_in_stock": 1},
		},
	}
	rec := doStockRequest(e, body, basicAuth(testUser, testPass))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["imported"] != float64(1) {
		t.Errorf("imported = %v, want 1", resp["imported"])
	}
}

// ---------- Validation tests ----------

func TestStockAPI_EmptyItems_Returns400(t *testing.T) {
	db := stockTestDB(t)
	e := stockTestServer(t, db)

	body := map[string]interface{}{"items": []map[string]interface{}{}}
	rec := doStockRequest(e, body, basicAuth(testUser, testPass))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestStockAPI_MissingItems_Returns400(t *testing.T) {
	db := stockTestDB(t)
	e := stockTestServer(t, db)

	rec := doStockRequest(e, map[string]interface{}{}, basicAuth(testUser, testPass))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestStockAPI_InvalidJSON_Returns400(t *testing.T) {
	db := stockTestDB(t)
	e := stockTestServer(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/stock/import", bytes.NewReader([]byte("{invalid")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", basicAuth(testUser, testPass))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// ---------- Functional tests ----------

func TestStockAPI_ImportAndUpsert(t *testing.T) {
	db := stockTestDB(t)
	p := productEntity.Product{SKU: "API-UPD", AttributeSetID: 4, TypeID: "simple"}
	db.Create(&p)
	e := stockTestServer(t, db)

	// First import
	body1 := map[string]interface{}{
		"items": []map[string]interface{}{
			{"sku": "API-UPD", "qty": 100, "is_in_stock": 1},
		},
	}
	rec1 := doStockRequest(e, body1, basicAuth(testUser, testPass))
	if rec1.Code != http.StatusOK {
		t.Fatalf("first import status = %d", rec1.Code)
	}

	// Second import updates
	body2 := map[string]interface{}{
		"items": []map[string]interface{}{
			{"sku": "API-UPD", "qty": 0, "is_in_stock": 0},
		},
	}
	rec2 := doStockRequest(e, body2, basicAuth(testUser, testPass))
	if rec2.Code != http.StatusOK {
		t.Fatalf("second import status = %d", rec2.Code)
	}

	var item productEntity.StockItem
	db.Where("product_id = ?", p.EntityID).First(&item)
	if item.Qty != 0 {
		t.Errorf("qty = %f, want 0", item.Qty)
	}
	if item.IsInStock != 0 {
		t.Errorf("is_in_stock = %d, want 0", item.IsInStock)
	}
}

func TestStockAPI_UnknownSKU_Skipped(t *testing.T) {
	db := stockTestDB(t)
	p := productEntity.Product{SKU: "API-REAL", AttributeSetID: 4, TypeID: "simple"}
	db.Create(&p)
	e := stockTestServer(t, db)

	body := map[string]interface{}{
		"items": []map[string]interface{}{
			{"sku": "API-REAL", "qty": 5, "is_in_stock": 1},
			{"sku": "API-GHOST", "qty": 1, "is_in_stock": 1},
		},
	}
	rec := doStockRequest(e, body, basicAuth(testUser, testPass))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["imported"] != float64(1) {
		t.Errorf("imported = %v, want 1", resp["imported"])
	}
	if resp["skipped"] != float64(1) {
		t.Errorf("skipped = %v, want 1", resp["skipped"])
	}
	warnings := resp["warnings"].([]interface{})
	if len(warnings) != 1 {
		t.Errorf("warnings count = %d, want 1", len(warnings))
	}
}

func TestStockAPI_ResponseHasDuration(t *testing.T) {
	db := stockTestDB(t)
	p := productEntity.Product{SKU: "API-DUR", AttributeSetID: 4, TypeID: "simple"}
	db.Create(&p)
	e := stockTestServer(t, db)

	body := map[string]interface{}{
		"items": []map[string]interface{}{
			{"sku": "API-DUR", "qty": 1},
		},
	}
	rec := doStockRequest(e, body, basicAuth(testUser, testPass))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	if rec.Header().Get("X-Request-Duration-ms") == "" {
		t.Error("missing X-Request-Duration-ms header")
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["request_duration_ms"] == nil {
		t.Error("missing request_duration_ms in response body")
	}
}

// ---------- Token auth tests ----------

const testStaticKey = "my_static_api_key_123"

func tokenAuthServer(t *testing.T, db *gorm.DB) *echo.Echo {
	return tokenAuthServerWithKey(t, db, testStaticKey)
}

func tokenAuthServerWithKey(t *testing.T, db *gorm.DB, staticKey string) *echo.Echo {
	t.Helper()
	e := echo.New()
	apiGroup := e.Group("/api")
	apiGroup.Use(middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
		Validator: func(token string, c echo.Context) (bool, error) {
			if staticKey != "" && token == staticKey {
				return true, nil
			}
			var count int64
			db.Table("oauth_token").
				Where("token = ? AND type = 'access' AND revoked = 0", token).
				Count(&count)
			return count > 0, nil
		},
	}))
	stockApi.RegisterStockRoutes(apiGroup, db)
	return e
}

func seedToken(t *testing.T, db *gorm.DB, token string, revoked uint16) {
	t.Helper()
	tk := entity.OauthToken{
		Type:    "access",
		Token:   token,
		Secret:  "secret",
		Revoked: revoked,
	}
	if err := db.Create(&tk).Error; err != nil {
		t.Fatalf("seed token: %v", err)
	}
}

func TestStockAPI_TokenAuth_ValidToken(t *testing.T) {
	db := stockTestDB(t)
	seedToken(t, db, "valid_test_token_123", 0)
	p := productEntity.Product{SKU: "TK-OK", AttributeSetID: 4, TypeID: "simple"}
	db.Create(&p)

	e := tokenAuthServer(t, db)
	body := map[string]interface{}{
		"items": []map[string]interface{}{
			{"sku": "TK-OK", "qty": 10, "is_in_stock": 1},
		},
	}
	rec := doStockRequest(e, body, "Bearer valid_test_token_123")
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["imported"] != float64(1) {
		t.Errorf("imported = %v, want 1", resp["imported"])
	}
}

func TestStockAPI_TokenAuth_NoToken(t *testing.T) {
	db := stockTestDB(t)
	e := tokenAuthServer(t, db)

	body := map[string]interface{}{
		"items": []map[string]interface{}{{"sku": "X", "qty": 1}},
	}
	rec := doStockRequest(e, body, "")
	// Echo KeyAuth returns 400 (missing key) when no Authorization header is present
	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 400 or 401", rec.Code)
	}
}

func TestStockAPI_TokenAuth_InvalidToken(t *testing.T) {
	db := stockTestDB(t)
	seedToken(t, db, "real_token", 0)
	e := tokenAuthServer(t, db)

	body := map[string]interface{}{
		"items": []map[string]interface{}{{"sku": "X", "qty": 1}},
	}
	rec := doStockRequest(e, body, "Bearer wrong_token")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestStockAPI_TokenAuth_RevokedToken(t *testing.T) {
	db := stockTestDB(t)
	seedToken(t, db, "revoked_token_abc", 1)
	e := tokenAuthServer(t, db)

	body := map[string]interface{}{
		"items": []map[string]interface{}{{"sku": "X", "qty": 1}},
	}
	rec := doStockRequest(e, body, "Bearer revoked_token_abc")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestStockAPI_TokenAuth_StaticKey(t *testing.T) {
	db := stockTestDB(t)
	p := productEntity.Product{SKU: "TK-STATIC", AttributeSetID: 4, TypeID: "simple"}
	db.Create(&p)

	e := tokenAuthServer(t, db)
	body := map[string]interface{}{
		"items": []map[string]interface{}{
			{"sku": "TK-STATIC", "qty": 5, "is_in_stock": 1},
		},
	}
	// Uses the static key, no DB token needed
	rec := doStockRequest(e, body, "Bearer "+testStaticKey)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["imported"] != float64(1) {
		t.Errorf("imported = %v, want 1", resp["imported"])
	}
}

func TestStockAPI_TokenAuth_StaticKeyDoesNotBypassWhenEmpty(t *testing.T) {
	db := stockTestDB(t)
	// Server with no static key configured
	e := tokenAuthServerWithKey(t, db, "")

	body := map[string]interface{}{
		"items": []map[string]interface{}{{"sku": "X", "qty": 1}},
	}
	// Random token should fail since no static key and no DB token
	rec := doStockRequest(e, body, "Bearer random_thing")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}
