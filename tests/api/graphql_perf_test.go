package apitest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	graphqlApi "magento.GO/api/graphql"
	productApi "magento.GO/api/product"
	entity "magento.GO/model/entity"
	productEntity "magento.GO/model/entity/product"
)

func seedProductAttributes(t *testing.T, db *gorm.DB, count int) {
	for i := 1; i <= count; i++ {
		attr := &entity.EavAttribute{
			AttributeID:   uint16(i),
			EntityTypeID:  4,
			AttributeCode: fmt.Sprintf("attr_%03d", i),
			BackendType:   "varchar",
		}
		if err := db.Create(attr).Error; err != nil {
			t.Fatalf("create eav_attribute %d: %v", i, err)
		}
	}
}

func addAttributesPerProduct(t *testing.T, db *gorm.DB, attrCount int) {
	var products []productEntity.Product
	if err := db.Find(&products).Error; err != nil {
		t.Fatalf("find products: %v", err)
	}
	for _, p := range products {
		for attrID := uint16(1); attrID <= uint16(attrCount); attrID++ {
			v := &productEntity.ProductVarchar{
				AttributeID: attrID,
				StoreID:     0,
				EntityID:    p.EntityID,
				Value:       fmt.Sprintf("val_%d_%d", p.EntityID, attrID),
			}
			if err := db.Create(v).Error; err != nil {
				t.Fatalf("create varchar entity=%d attr=%d: %v", p.EntityID, attrID, err)
			}
		}
	}
}

// TestPerf_GraphQL_vs_API creates 100 products with 100 attributes each, measures REST API and GraphQL fetch times, outputs comparison.
func TestPerf_GraphQL_vs_API(t *testing.T) {
	t.Setenv("PRODUCT_FLAT_CACHE", "off")
	e := echo.New()
	db := graphqlProductTestDB(t)
	seedProductAttributes(t, db, 100)

	api := e.Group("/api")
	productApi.RegisterProductRoutes(api, db)
	graphqlApi.RegisterGraphQLRoutes(e, db)

	// 1. Create 100 products
	createStart := time.Now()
	for i := 0; i < 100; i++ {
		createBody := map[string]interface{}{
			"AttributeSetID": 1,
			"TypeID":         "simple",
			"SKU":            fmt.Sprintf("PERF-SKU-%03d", i+1),
		}
		createBytes, _ := json.Marshal(createBody)
		createReq := httptest.NewRequest(http.MethodPost, "/api/products", bytes.NewReader(createBytes))
		createReq.Header.Set("Content-Type", "application/json")
		createRec := httptest.NewRecorder()
		e.ServeHTTP(createRec, createReq)
		if createRec.Code != http.StatusCreated {
			t.Fatalf("create product %d: status = %d, want 201", i+1, createRec.Code)
		}
	}
	createDur := time.Since(createStart)

	// 1b. Add 100 attributes per product
	attrStart := time.Now()
	addAttributesPerProduct(t, db, 100)
	attrDur := time.Since(attrStart)

	// 2. REST API: GET /api/products/flat (all 100 with attributes) - run twice
	runRestAPI := func() time.Duration {
		start := time.Now()
		req := httptest.NewRequest(http.MethodGet, "/api/products/flat", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("REST API flat status = %d, want 200", rec.Code)
		}
		return time.Since(start)
	}
	apiDur1 := runRestAPI()
	apiDur2 := runRestAPI()

	// 3. GraphQL: fetch 1, 50, 100 products
	gqlQuery := `query($pageSize: Int) { products(pageSize: $pageSize) { items { entity_id sku name type_id price } total_count } }`
	runGql := func(pageSize int) time.Duration {
		body := map[string]interface{}{"query": gqlQuery, "variables": map[string]int{"pageSize": pageSize}}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		start := time.Now()
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("graphql pageSize=%d status = %d", pageSize, rec.Code)
		}
		return time.Since(start)
	}
	gqlDur1 := runGql(1)
	gqlDur50 := runGql(50)
	gqlDur100 := runGql(100)

	output := fmt.Sprintf(`=== GraphQL vs REST API Performance (100 products, 100 attrs each) ===
Create 100 products:   %v
Add 100 attrs each:    %v
REST API flat (100):   %v
REST API flat (100):   %v
GraphQL fetch 1:       %v
GraphQL fetch 50:      %v
GraphQL fetch 100:     %v
=====================================================`, createDur, attrDur, apiDur1, apiDur2, gqlDur1, gqlDur50, gqlDur100)
	t.Log(output)
	fmt.Println(output)
}

// TestGraphQL_Perf_100Products creates 100 products with 100 attributes each, fetches via GraphQL, measures and outputs speed.
func TestGraphQL_Perf_100Products(t *testing.T) {
	t.Setenv("PRODUCT_FLAT_CACHE", "off")
	e := echo.New()
	db := graphqlProductTestDB(t)
	seedProductAttributes(t, db, 100)

	api := e.Group("/api")
	productApi.RegisterProductRoutes(api, db)
	graphqlApi.RegisterGraphQLRoutes(e, db)

	// 1. Create 100 products
	createStart := time.Now()
	for i := 0; i < 100; i++ {
		createBody := map[string]interface{}{
			"AttributeSetID": 1,
			"TypeID":         "simple",
			"SKU":            fmt.Sprintf("PERF-SKU-%03d", i+1),
		}
		createBytes, _ := json.Marshal(createBody)
		createReq := httptest.NewRequest(http.MethodPost, "/api/products", bytes.NewReader(createBytes))
		createReq.Header.Set("Content-Type", "application/json")
		createRec := httptest.NewRecorder()
		e.ServeHTTP(createRec, createReq)
		if createRec.Code != http.StatusCreated {
			t.Fatalf("create product %d: status = %d, want 201", i+1, createRec.Code)
		}
	}
	createDur := time.Since(createStart)

	// 1b. Add 100 attributes per product
	addAttributesPerProduct(t, db, 100)

	// 2. Fetch via GraphQL: 1, 50, 100 products
	gqlQuery := `query($pageSize: Int) { products(pageSize: $pageSize) { items { entity_id sku name type_id price } total_count page_info { page_size current_page total_pages } } }`

	runGql := func(pageSize int) (dur time.Duration, totalCount int, items []interface{}) {
		body := map[string]interface{}{"query": gqlQuery, "variables": map[string]int{"pageSize": pageSize}}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		start := time.Now()
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		dur = time.Since(start)
		if rec.Code != http.StatusOK {
			t.Fatalf("graphql pageSize=%d status = %d", pageSize, rec.Code)
		}
		var r struct {
			Data   map[string]interface{}
			Errors []struct{ Message string }
		}
		if err := json.NewDecoder(rec.Body).Decode(&r); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(r.Errors) > 0 {
			t.Fatalf("graphql errors: %v", r.Errors)
		}
		p := r.Data["products"].(map[string]interface{})
		totalCount = int(p["total_count"].(float64))
		items = p["items"].([]interface{})
		return dur, totalCount, items
	}

	gqlDur1, tc1, items1 := runGql(1)
	gqlDur50, tc50, items50 := runGql(50)
	gqlDur100, tc100, items100 := runGql(100)

	if tc1 != 100 || len(items1) != 1 {
		t.Fatalf("fetch 1: total_count=%d items=%d, want total_count=100 items=1", tc1, len(items1))
	}
	if tc50 != 100 || len(items50) != 50 {
		t.Fatalf("fetch 50: total_count=%d items=%d, want total_count=100 items=50", tc50, len(items50))
	}
	if tc100 != 100 || len(items100) != 100 {
		t.Fatalf("fetch 100: total_count=%d items=%d, want total_count=100 items=100", tc100, len(items100))
	}

	output := fmt.Sprintf(`=== GraphQL Performance Test (100 products, 100 attrs each) ===
Create 100 products:  %v
GraphQL fetch 1:     %v
GraphQL fetch 50:    %v
GraphQL fetch 100:   %v
===============================================`, createDur, gqlDur1, gqlDur50, gqlDur100)
	t.Log(output)
	fmt.Println(output)
}
