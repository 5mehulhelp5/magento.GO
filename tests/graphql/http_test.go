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

func TestGraphQL_HTTPRequestToResult(t *testing.T) {
	e := echo.New()
	graphqlApi.RegisterGraphQLRoutesWithSchema(e, NewMockSchema())

	body := map[string]interface{}{
		"query":     `query { products { items { sku name } total_count } }`,
		"variables": map[string]interface{}{},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Store", "1")
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
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Errors) > 0 {
		t.Fatalf("errors: %v", resp.Errors)
	}
	products := resp.Data["products"].(map[string]interface{})
	if int(products["total_count"].(float64)) != 1 {
		t.Errorf("total_count = %v, want 1", products["total_count"])
	}
	items := products["items"].([]interface{})
	if len(items) != 1 || items[0].(map[string]interface{})["sku"] != "MOCK-SKU-1" {
		t.Errorf("items = %v", items)
	}
}

func TestGraphQL_StoreFromHeader(t *testing.T) {
	e := echo.New()
	graphqlApi.RegisterGraphQLRoutesWithSchema(e, NewMockSchema())

	body := map[string]interface{}{
		"query": `{ categories { entity_id } }`,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Store", "2")
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
	if resp.Data["categories"] == nil {
		t.Fatal("categories missing")
	}
}
