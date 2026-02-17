# Global Cache & Registry

## Cache (`core/cache`)

```go
cache := cache.GetInstance()
cache.Set("key", value)
val, ok := cache.Get("key")
cache.Delete("key")
```

## Registry (`core/registry`)

```go
registry.SetGlobal("site_name", "MySite")
site, ok := registry.GetGlobal("site_name")
registry.DeleteGlobal("site_name")
```

## Singleton Repositories

```go
repo := product.GetProductRepository(db)
```

One instance per DB; shared across requests.
