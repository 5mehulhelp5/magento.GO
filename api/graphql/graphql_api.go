package graphql

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/graph-gophers/graphql-go"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	_ "magento.GO/custom"
	graphqlpkg "magento.GO/graphql"
	"magento.GO/graphqlserver"
)

// GraphQLRequest is the standard GraphQL request body
type GraphQLRequest struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
}

// GraphQLResponse is the standard GraphQL response
type GraphQLResponse struct {
	Data   interface{}   `json:"data,omitempty"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

type GraphQLError struct {
	Message string `json:"message"`
}

func RegisterGraphQLRoutes(e *echo.Echo, db *gorm.DB) {
	schema, err := graphqlserver.NewSchema(db)
	if err != nil {
		panic("graphql schema: " + err.Error())
	}
	registerRoutes(e, schema)
}

// RegisterGraphQLRoutesWithSchema registers /graphql with a custom schema (for tests with mocks).
func RegisterGraphQLRoutesWithSchema(e *echo.Echo, schema *graphql.Schema) {
	registerRoutes(e, schema)
}

func registerRoutes(e *echo.Echo, schema *graphql.Schema) {
	handler := graphqlserver.Handler(schema)
	h := storeContextMiddleware(handler)
	e.POST("/graphql", echo.WrapHandler(h))
	e.GET("/graphql", echo.WrapHandler(h))
	e.GET("/playground", echo.WrapHandler(playgroundHandler()))
}

func storeContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		storeID := uint16(0)
		if h := r.Header.Get("Store"); h != "" {
			if id, err := strconv.ParseUint(h, 10, 16); err == nil {
				storeID = uint16(id)
			}
		}
		if r.Method == http.MethodPost {
			body, _ := io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewReader(body))
			var req struct {
				Variables map[string]interface{} `json:"variables"`
			}
			if json.Unmarshal(body, &req) == nil && req.Variables != nil {
				if v, ok := req.Variables["__Store"]; ok {
					switch val := v.(type) {
					case string:
						if id, err := strconv.ParseUint(val, 10, 16); err == nil {
							storeID = uint16(id)
						}
					case float64:
						storeID = uint16(val)
					}
				}
			}
		}
		if q := r.URL.Query().Get("__Store"); q != "" {
			if id, err := strconv.ParseUint(q, 10, 16); err == nil {
				storeID = uint16(id)
			}
		}
		ctx := graphqlpkg.WithStoreID(r.Context(), storeID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func playgroundHandler() http.Handler {
	html := `<!DOCTYPE html>
<html>
<head>
	<title>GraphQL Playground</title>
	<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/static/css/index.css"/>
</head>
<body>
	<div id="root"/>
	<script src="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/static/js/middleware.js"></script>
	<script>window.addEventListener('load', function() {
		GraphQLPlayground.init({ endpoint: '/graphql' });
	})</script>
</body>
</html>`
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})
}
