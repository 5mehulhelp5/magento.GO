package graphqltest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	graphqlApi "magento.GO/api/graphql"
)

func runQuery(t *testing.T, query string, variables map[string]interface{}) *httptest.ResponseRecorder {
	e := echo.New()
	graphqlApi.RegisterGraphQLRoutesWithSchema(e, NewMockSchema())

	body := map[string]interface{}{"query": query}
	if variables != nil {
		body["variables"] = variables
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Store", "1")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestExecuteQuery_Products(t *testing.T) {
	rec := runQuery(t, `query { products { items { sku name } total_count } }`, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
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
	products, ok := resp.Data["products"].(map[string]interface{})
	if !ok {
		t.Fatalf("data.products missing")
	}
	if int(products["total_count"].(float64)) != 1 {
		t.Errorf("total_count = %v, want 1", products["total_count"])
	}
	items := products["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("len(items) = %d", len(items))
	}
	item := items[0].(map[string]interface{})
	if item["sku"] != "MOCK-SKU-1" {
		t.Errorf("sku = %v", item["sku"])
	}
}

func TestExecuteQuery_Product(t *testing.T) {
	rec := runQuery(t, `query { product(sku: "x") { sku name } }`, map[string]interface{}{"sku": "x"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var resp struct {
		Data   map[string]interface{}
		Errors []struct{ Message string }
	}
	json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp.Errors) > 0 {
		t.Fatalf("errors: %v", resp.Errors)
	}
	product := resp.Data["product"].(map[string]interface{})
	if product["sku"] != "MOCK-SINGLE" {
		t.Errorf("sku = %v", product["sku"])
	}
}

func TestExecuteQuery_Categories(t *testing.T) {
	rec := runQuery(t, `{ categories { entity_id name } }`, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var resp struct {
		Data   map[string]interface{}
		Errors []struct{ Message string }
	}
	json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp.Errors) > 0 {
		t.Fatalf("errors: %v", resp.Errors)
	}
	cats := resp.Data["categories"].([]interface{})
	if len(cats) != 1 {
		t.Fatalf("len = %d", len(cats))
	}
	if cats[0].(map[string]interface{})["entity_id"] != "1" {
		t.Error("entity_id mismatch")
	}
}

func TestExecuteQuery_Category(t *testing.T) {
	rec := runQuery(t, `{ category(id: "1") { entity_id name } }`, map[string]interface{}{"id": "1"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var resp struct {
		Data   map[string]interface{}
		Errors []struct{ Message string }
	}
	json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp.Errors) > 0 {
		t.Fatalf("errors: %v", resp.Errors)
	}
	cat := resp.Data["category"].(map[string]interface{})
	if cat["entity_id"] != "1" {
		t.Errorf("entity_id = %v", cat["entity_id"])
	}
}

func TestExecuteQuery_CategoryTree(t *testing.T) {
	rec := runQuery(t, `{ categoryTree { entity_id name } }`, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var resp struct {
		Data   map[string]interface{}
		Errors []struct{ Message string }
	}
	json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp.Errors) > 0 {
		t.Fatalf("errors: %v", resp.Errors)
	}
	tree := resp.Data["categoryTree"].([]interface{})
	if len(tree) != 1 {
		t.Fatalf("len = %d", len(tree))
	}
}

func TestExecuteQuery_Search(t *testing.T) {
	rec := runQuery(t, `{ search(query: "test") { items { sku } total_count } }`, map[string]interface{}{"query": "test"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var resp struct {
		Data   map[string]interface{}
		Errors []struct{ Message string }
	}
	json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp.Errors) > 0 {
		t.Fatalf("errors: %v", resp.Errors)
	}
	search := resp.Data["search"].(map[string]interface{})
	if int(search["total_count"].(float64)) != 1 {
		t.Errorf("total_count = %v", search["total_count"])
	}
	items := search["items"].([]interface{})
	if len(items) != 1 || items[0].(map[string]interface{})["sku"] != "SEARCH-1" {
		t.Errorf("items = %v", items)
	}
}

func TestExecuteQuery_UnknownField(t *testing.T) {
	rec := runQuery(t, `{ unknownField { x } }`, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var resp struct {
		Data   map[string]interface{}
		Errors []struct{ Message string }
	}
	json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp.Errors) == 0 {
		t.Error("expected errors for unknown field")
	}
}
