# Cache & Performance

## Product Flat Cache

- Global in-memory cache for flattened products
- Concurrent-safe (`sync.RWMutex`)
- Set `PRODUCT_FLAT_CACHE=off` to bypass (direct DB)

## No N+1

`fetchFlatProducts` uses GORM Preload with IN clauses — ~10 batch queries regardless of product count.

## Benchmarks

- **With cache:** ~300 req/s, ~1 ms single product
- **100 products, 100 attrs:** REST ~25 ms, GraphQL ~30 ms (`make test-perf`)
