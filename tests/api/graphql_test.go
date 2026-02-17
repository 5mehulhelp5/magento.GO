package apitest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	graphqlApi "magento.GO/api/graphql"
	productApi "magento.GO/api/product"
	entity "magento.GO/model/entity"
	productEntity "magento.GO/model/entity/product"
)

func graphqlProductTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	entities := []interface{}{
		&productEntity.Product{},
		&productEntity.ProductVarchar{},
		&productEntity.ProductInt{},
		&productEntity.ProductDecimal{},
		&productEntity.ProductText{},
		&productEntity.ProductDatetime{},
		&productEntity.ProductMediaGallery{},
		&productEntity.StockItem{},
		&productEntity.ProductIndexPrice{},
		&entity.EavAttribute{},
	}
	for _, e := range entities {
		if err := db.AutoMigrate(e); err != nil {
			t.Fatalf("migrate: %v", err)
		}
	}
	return db
}

func TestGraphQL_Products_DataCheck(t *testing.T) {
	t.Setenv("PRODUCT_FLAT_CACHE", "off")
	e := echo.New()
	db := graphqlProductTestDB(t)
	api := e.Group("/api")
	productApi.RegisterProductRoutes(api, db)
	graphqlApi.RegisterGraphQLRoutes(e, db)

	createBody := map[string]interface{}{"AttributeSetID": 1, "TypeID": "simple", "SKU": "GQL-DATA-CHECK-SKU"}
	createBytes, _ := json.Marshal(createBody)
	createReq := httptest.NewRequest(http.MethodPost, "/api/products", bytes.NewReader(createBytes))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	e.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", createRec.Code)
	}

	gqlBody := map[string]interface{}{"query": `query { products { items { entity_id sku name type_id price } total_count } }`}
	gqlBytes, _ := json.Marshal(gqlBody)
	gqlReq := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(gqlBytes))
	gqlReq.Header.Set("Content-Type", "application/json")
	gqlRec := httptest.NewRecorder()
	e.ServeHTTP(gqlRec, gqlReq)
	if gqlRec.Code != http.StatusOK {
		t.Fatalf("graphql status = %d, want 200", gqlRec.Code)
	}

	var resp struct {
		Data   map[string]interface{}
		Errors []struct{ Message string }
	}
	if err := json.NewDecoder(gqlRec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Errors) > 0 {
		t.Fatalf("graphql errors: %v", resp.Errors)
	}
	products := resp.Data["products"].(map[string]interface{})
	if int(products["total_count"].(float64)) != 1 {
		t.Errorf("total_count = %v, want 1", products["total_count"])
	}
	items := products["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("items len = %d, want 1", len(items))
	}
	item := items[0].(map[string]interface{})
	if item["sku"] != "GQL-DATA-CHECK-SKU" {
		t.Errorf("items[0].sku = %v, want GQL-DATA-CHECK-SKU", item["sku"])
	}
}

func TestGraphQL_Product_BySKU_DataCheck(t *testing.T) {
	t.Setenv("PRODUCT_FLAT_CACHE", "off")
	e := echo.New()
	db := graphqlProductTestDB(t)
	api := e.Group("/api")
	productApi.RegisterProductRoutes(api, db)
	graphqlApi.RegisterGraphQLRoutes(e, db)

	createBody := map[string]interface{}{"AttributeSetID": 1, "TypeID": "simple", "SKU": "GQL-SINGLE-SKU"}
	createBytes, _ := json.Marshal(createBody)
	createReq := httptest.NewRequest(http.MethodPost, "/api/products", bytes.NewReader(createBytes))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	e.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", createRec.Code)
	}

	gqlBody := map[string]interface{}{
		"query":     `query($sku: String) { product(sku: $sku) { entity_id sku type_id } }`,
		"variables": map[string]interface{}{"sku": "GQL-SINGLE-SKU"},
	}
	gqlBytes, _ := json.Marshal(gqlBody)
	gqlReq := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(gqlBytes))
	gqlReq.Header.Set("Content-Type", "application/json")
	gqlRec := httptest.NewRecorder()
	e.ServeHTTP(gqlRec, gqlReq)
	if gqlRec.Code != http.StatusOK {
		t.Fatalf("graphql status = %d, want 200", gqlRec.Code)
	}

	var resp struct {
		Data   map[string]interface{}
		Errors []struct{ Message string }
	}
	if err := json.NewDecoder(gqlRec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Errors) > 0 {
		t.Fatalf("graphql errors: %v", resp.Errors)
	}
	productVal := resp.Data["product"]
	if productVal == nil {
		t.Fatalf("product is null")
	}
	product := productVal.(map[string]interface{})
	if product["sku"] != "GQL-SINGLE-SKU" {
		t.Errorf("product.sku = %v, want GQL-SINGLE-SKU", product["sku"])
	}
}

func TestGraphQL_MagentoProducts_DataCheck(t *testing.T) {
	t.Setenv("PRODUCT_FLAT_CACHE", "off")
	e := echo.New()
	db := graphqlProductTestDB(t)
	api := e.Group("/api")
	productApi.RegisterProductRoutes(api, db)
	graphqlApi.RegisterGraphQLRoutes(e, db)

	createBody := map[string]interface{}{"AttributeSetID": 1, "TypeID": "simple", "SKU": "MAGENTO-SKU-1"}
	createBytes, _ := json.Marshal(createBody)
	createReq := httptest.NewRequest(http.MethodPost, "/api/products", bytes.NewReader(createBytes))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	e.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", createRec.Code)
	}

	gqlBody := map[string]interface{}{
		"query": `query { magentoProducts(pageSize: 12, currentPage: 1) {
			items { id uid name sku price_range { maximum_price { final_price { currency value } regular_price { currency value } } } stock_status url_key }
			page_info { total_pages }
			total_count
		} }`,
	}
	gqlBytes, _ := json.Marshal(gqlBody)
	gqlReq := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(gqlBytes))
	gqlReq.Header.Set("Content-Type", "application/json")
	gqlRec := httptest.NewRecorder()
	e.ServeHTTP(gqlRec, gqlReq)
	if gqlRec.Code != http.StatusOK {
		t.Fatalf("graphql status = %d, want 200", gqlRec.Code)
	}

	var resp struct {
		Data   map[string]interface{}
		Errors []struct{ Message string }
	}
	if err := json.NewDecoder(gqlRec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Errors) > 0 {
		t.Fatalf("graphql errors: %v", resp.Errors)
	}
	products := resp.Data["magentoProducts"].(map[string]interface{})
	if int(products["total_count"].(float64)) < 1 {
		t.Errorf("total_count = %v, want >= 1", products["total_count"])
	}
	items := products["items"].([]interface{})
	if len(items) < 1 {
		t.Fatalf("items len = %d, want >= 1", len(items))
	}
	item := items[0].(map[string]interface{})
	if item["sku"] != "MAGENTO-SKU-1" {
		t.Errorf("items[0].sku = %v, want MAGENTO-SKU-1", item["sku"])
	}
	if item["uid"] == nil || item["uid"] == "" {
		t.Errorf("items[0].uid missing (Magento format)")
	}
	pr := item["price_range"].(map[string]interface{})
	maxPrice := pr["maximum_price"].(map[string]interface{})
	finalPrice := maxPrice["final_price"].(map[string]interface{})
	if finalPrice["currency"] != "USD" {
		t.Errorf("price_range.currency = %v, want USD", finalPrice["currency"])
	}
}

func TestGraphQL_Extension_Registry(t *testing.T) {
	t.Setenv("PRODUCT_FLAT_CACHE", "off")
	e := echo.New()
	db := graphqlProductTestDB(t)
	api := e.Group("/api")
	productApi.RegisterProductRoutes(api, db)
	graphqlApi.RegisterGraphQLRoutes(e, db)

	gqlBody := map[string]interface{}{"query": `query { _extension(name: "ping", args: "{}") }`}
	gqlBytes, _ := json.Marshal(gqlBody)
	gqlReq := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(gqlBytes))
	gqlReq.Header.Set("Content-Type", "application/json")
	gqlRec := httptest.NewRecorder()
	e.ServeHTTP(gqlRec, gqlReq)
	if gqlRec.Code != http.StatusOK {
		t.Fatalf("graphql status = %d, want 200", gqlRec.Code)
	}

	var resp struct {
		Data   map[string]interface{}
		Errors []struct{ Message string }
	}
	if err := json.NewDecoder(gqlRec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Errors) > 0 {
		t.Fatalf("graphql errors: %v", resp.Errors)
	}
	ext := resp.Data["_extension"]
	if ext == nil {
		t.Fatalf("_extension is null")
	}
	s, ok := ext.(string)
	if !ok {
		t.Fatalf("_extension = %T, want string", ext)
	}
	if s != `{"pong":"ok"}` {
		t.Errorf("_extension = %q, want %q", s, `{"pong":"ok"}`)
	}
}

func TestGraphQL_Standalone_Ping(t *testing.T) {
	t.Setenv("PRODUCT_FLAT_CACHE", "off")
	e := echo.New()
	db := graphqlProductTestDB(t)
	graphqlApi.RegisterGraphQLRoutes(e, db)

	gqlBody := map[string]interface{}{"query": `query { _extension(name: "ping", args: "{}") }`}
	gqlBytes, _ := json.Marshal(gqlBody)
	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(gqlBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp struct {
		Data   map[string]interface{}
		Errors []struct{ Message string }
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Errors) > 0 {
		t.Fatalf("errors: %v", resp.Errors)
	}
	if s, ok := resp.Data["_extension"].(string); !ok || s != `{"pong":"ok"}` {
		t.Errorf("_extension = %v, want %q", resp.Data["_extension"], `{"pong":"ok"}`)
	}
}
