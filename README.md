# GoGento Catalog — Magento GraphQL & REST API in Go

Magento 2 API and Frontend in Go — HTML server side rendering, GraphQL and REST, one binary, no slow PHP.

## The world’s fastest framework for building e-Commerce MAGENTO websites!

![image](https://github.com/user-attachments/assets/eaacc9e0-e497-4d3c-a4d9-faeadc7fd6e5)

**GoGento Catalog** connects your Magento 2 database to modern frontends via Echo and GORM. Single binary, ~300+ req/s on a single CPU, sub-30ms for 100 products with 100 EAV attributes. EAV flattening, stock, prices, categories. Concurrent-safe cache, extensible registry, standalone GraphQL mode. Works with Venia, React, Next.js, Vue.

**Why GoGento?** If you run Magento 2 and want a fast, headless API without PHP — deploy one binary, point it at your MySQL, and serve catalog data to PWA, mobile apps, or third-party integrations. No Magento runtime, no Composer, no PHP-FPM. Lower memory, faster cold starts, simpler ops. Use your existing Magento DB schema; products, categories, EAV attributes, stock, and prices flow through unchanged.

**Architecture.** Repository layer for DB access, service layer for logic, API handlers for HTTP. GraphQL schema matches Magento/Venia conventions so frontends built for Magento GraphQL work with minimal changes. REST flat endpoints return products with attributes as keys. Optional global in-memory cache for hot paths; disable with `PRODUCT_FLAT_CACHE=off` for direct DB. Cron jobs for background tasks. Extensible registry for custom resolvers and fields.

**Deployment.** Run full API (REST + GraphQL) or standalone GraphQL server. Single executable, config via env vars. systemd, Docker, or bare metal. No separate app server. Scale horizontally by running more instances behind a load balancer.

## Features

| Feature | Description | Doc |
|---------|-------------|-----|
| **GraphQL API** | Products, categories, search; Magento/Venia-compatible schema. Store header, pagination, filters. | [graphql.md](doc/graphql.md) |
| **REST API** | Flat products (EAV as keys), orders CRUD. Basic auth. Optional store ID. | [rest-api.md](doc/rest-api.md) |
| **Standalone GraphQL** | Run GraphQL only: `go run ./cmd/graphql`. No REST, smaller footprint. | [installation.md](doc/installation.md) |
| **EAV flattening** | Attributes as keys, stock_item, index_prices. FetchWithAllAttributesFlat. | [eav-products.md](doc/eav-products.md) |
| **Global cache** | In-memory, concurrent-safe. ~300 req/s. Set `PRODUCT_FLAT_CACHE=off` to bypass. | [cache.md](doc/cache.md) |
| **Registry & cache** | Global cache, registry, singleton repos. Per-request isolation. | [registry.md](doc/registry.md) |
| **Cron jobs** | Scheduled and on-demand. `go run cli.go cron:start --job Name`. | [cron.md](doc/cron.md) |
| **Extending** | Add entities, custom resolvers, GraphQL extensions. Tailwind. | [extending.md](doc/extending.md) |

## Quick Start

```bash
cd gogento-catalog
go mod tidy
go run magento.go
```

- **GraphQL:** `POST http://localhost:8080/graphql`
- **Playground:** `GET http://localhost:8080/playground`

```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -H "Store: 1" \
  -d '{"query":"query { products { total_count } }"}'
```

## Documentation

[Technical index](doc/technical.md) · [Installation](doc/installation.md) · [Production](doc/production.md)

## Performance

- **With cache:** ~300 req/s, ~1 ms single product (ApacheBench)
- **100 products, 100 attrs:** REST ~25 ms, GraphQL ~30 ms (`make test-perf`)
- No N+1: batch Preload with IN clauses

## Environment

```bash
MYSQL_USER=magento MYSQL_PASS=magento MYSQL_HOST=localhost MYSQL_DB=magento
API_USER=admin API_PASS=secret PORT=8080
```

## Tests

```bash
make test      # All tests
make test-perf # GraphQL vs REST benchmark
```
