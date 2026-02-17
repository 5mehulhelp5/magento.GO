# GraphQL API — Architecture & Conventions

## Overview

GoGento uses [graph-gophers/graphql-go](https://github.com/graph-gophers/graphql-go) with an embedded schema. Store context is passed via headers, query params, or variables.

## Architecture

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  api/graphql/   │────▶│ graphqlserver/   │────▶│ graphql/        │
│  HTTP handler   │     │ RootResolver     │     │ resolvers/      │
│  POST /graphql  │     │ QueryResolver    │     │ (business logic)│
└─────────────────┘     └──────────────────┘     └────────┬────────┘
         │                           │                      │
         │ Store from header/        │                      │
         │ query/variables          │                      ▼
         ▼                           │             ┌─────────────────┐
┌─────────────────┐                   │             │ model/          │
│ graphql/context │                   │             │ repository/     │
│ StoreID in ctx  │                   │             │ (DB access)     │
└─────────────────┘                   │             └─────────────────┘
                                      │
                                      ▼
                             ┌─────────────────┐
                             │ graphql/models/  │
                             │ (gqlmodels)     │
                             │ Response DTOs   │
                             └─────────────────┘
```

### Layers

| Layer | Path | Role |
|-------|------|------|
| **HTTP** | `api/graphql/` | Parses request, extracts Store, calls relay handler |
| **Schema** | `graphql/schema.graphqls` | GraphQL types and Query definition |
| **Server** | `graphqlserver/` | Parses schema, wires RootResolver → QueryResolver |
| **Registry** | `graphql/registry/` | Dynamic resolver registration for `_extension` |
| **Resolvers** | `graphql/resolvers/` | Fetches data, maps to gqlmodels |
| **Models** | `graphql/models/` | Response DTOs (import as `gqlmodels`) |
| **Custom** | `custom/` | Packages that register via `gqlregistry.Register` in `init()` |
| **Repository** | `model/repository/` | DB access |

## Conventions

1. **Import alias:** Use `gqlmodels "magento.GO/graphql/models"` to distinguish from domain models.
2. **Store context:** Resolvers get `StoreID` via `graphql.StoreIDFromContext(ctx)`.
3. **Resolver pattern:** `graphqlserver.QueryResolver` delegates to `resolvers.QueryResolver` (created per request with store).
4. **Naming:** Schema fields use `camelCase`; Go structs use `PascalCase`; graphql-go maps automatically.

## How to Add a New GraphQL Endpoint

### Example: Add `featuredProducts(limit: Int): [Product!]!`

#### 1. Schema — `graphql/schema.graphqls`

```graphql
type Query {
  # ... existing fields ...
  featuredProducts(limit: Int = 5): [Product!]!
}
```

#### 2. Resolver — `graphql/resolvers/featured.go`

```go
package resolvers

import (
	"context"

	gqlmodels "magento.GO/graphql/models"
)

func (r *queryResolver) FeaturedProducts(ctx context.Context, limit *int) ([]*gqlmodels.Product, error) {
	n := 5
	if limit != nil && *limit > 0 {
		n = *limit
	}
	flat, err := r.ProductRepo.FetchWithAllAttributesFlat(r.StoreID)
	if err != nil {
		return nil, err
	}
	items := filterProductsForGuest(flat, r.CustomerGroupID)
	if len(items) > n {
		items = items[:n]
	}
	result := make([]*gqlmodels.Product, len(items))
	for i, p := range items {
		result[i] = flatToProduct(p)
	}
	return result, nil
}
```

#### 3. Interface — `graphql/query_resolver.go`

```go
type QueryResolver interface {
	// ... existing methods ...
	FeaturedProducts(ctx context.Context, limit *int) ([]*gqlmodels.Product, error)
}
```

#### 4. Server — `graphqlserver/server.go`

```go
func (r *QueryResolver) FeaturedProducts(ctx context.Context, args struct {
	Limit *int32
}) ([]*gqlmodels.Product, error) {
	storeID := graphql.StoreIDFromContext(ctx)
	res := resolvers.NewResolver(r.db, storeID)
	var limit *int
	if args.Limit != nil {
		l := int(*args.Limit)
		limit = &l
	}
	return res.Query().FeaturedProducts(ctx, limit)
}
```

#### 5. Mock (for tests) — `tests/graphql/mock_resolvers.go`

```go
func (m *MockQueryResolver) FeaturedProducts(ctx context.Context, args struct {
	Limit *int32
}) ([]*gqlmodels.Product, error) {
	name := "Featured"
	price := 49.99
	return []*gqlmodels.Product{{EntityID: "1", SKU: "FEAT-1", Name: &name, Price: &price}}, nil
}
```

#### 6. Test

```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -H "Store: 1" \
  -d '{"query":"query { featuredProducts(limit: 3) { sku name } }"}'
```

## Store Resolution

Store ID is resolved in order:

1. **Header:** `Store: 1`
2. **Query param:** `?__Store=1`
3. **Variables:** `{"variables": {"__Store": "1"}}`

## Custom Extensions

Add packages under `custom/` that call `gqlregistry.Register(name, resolve)` in `init()`. They are loaded via blank import in `api/graphql`. Use alias `gqlregistry "magento.GO/graphql/registry"`.

**Example** — `custom/example.go`:

```go
package custom

import (
	"context"

	gqlregistry "magento.GO/graphql/registry"
)

func init() {
	gqlregistry.Register("ping", func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return map[string]string{"pong": "ok"}, nil
	})
}
```

**Call via GraphQL:**

```graphql
query { _extension(name: "ping", args: "{}") }
```

Returns JSON string. `args` is optional; pass `"{}"` or omit for no arguments.

## Available Queries

| Query | Description |
|-------|-------------|
| `products` | Paginated products (pageSize, currentPage, skus, categoryId) |
| `product` | Single product by sku or url_key |
| `categories` | All categories |
| `category` | Category by id |
| `categoryTree` | Category tree |
| `magentoCategories` | Magento/Venia format, filter by category_uid |
| `magentoProducts` | Magento/Venia format, filter/sort by category |
| `search` | Elasticsearch full-text search |
| `_extension` | Call registered custom resolver by name (args: JSON string) |

## Custom Registries (cmd, cron, routes)

Same pattern as GraphQL extensions: add packages under `custom/` that call registry `Register` in `init()`.

| Registry | Package | Usage |
|----------|---------|-------|
| **Commands** | `cmd` | `cmd.Register(&cobra.Command{...})` |
| **Cron jobs** | `cron` | `cron.Register("name", "@every 1h", func(args ...string){...})` |
| **HTTP routes** | `api` | `api.RegisterGET("/path", handler)` or `api.Register(method, path, handler)` |

Import `_ "magento.GO/custom"` in `cli.go` (for cmd/cron) and in `api/graphql` (for routes, via custom). Example: `custom/example.go`.

## References

- [graph-gophers/graphql-go](https://github.com/graph-gophers/graphql-go)
- [README-GRAPHQL.md](../README-GRAPHQL.md) — Usage, store, search, env vars
