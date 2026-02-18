package registry

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"magento.GO/core/registry"
)

// ResolverFunc is the signature for custom resolvers. Args is JSON-decoded map.
type ResolverFunc func(ctx context.Context, args map[string]interface{}) (interface{}, error)

// QueryResolverFactory creates the Query resolver for graphql-go. Call from init().
type QueryResolverFactory func(db interface{}) interface{}

var mu sync.Mutex
var graphqlLocked int32
var queryResolverFactory QueryResolverFactory

// RegisterQueryResolverFactory sets the factory for the main Query resolver.
func RegisterQueryResolverFactory(fn QueryResolverFactory) {
	mu.Lock()
	defer mu.Unlock()
	queryResolverFactory = fn
}

// GetQueryResolver returns the Query resolver. Panics if not registered.
func GetQueryResolver(db interface{}) interface{} {
	if queryResolverFactory == nil {
		panic("graphql/registry: QueryResolverFactory not registered")
	}
	return queryResolverFactory(db)
}

func getEntries() map[string]ResolverFunc {
	if v, ok := registry.GlobalRegistry.GetGlobal(registry.KeyRegistryGraphQL); ok && v != nil {
		return v.(map[string]ResolverFunc)
	}
	return make(map[string]ResolverFunc)
}

// Register adds a resolver. Call from init() in custom packages. Name must be unique. Panics if locked.
func Register(name string, resolve ResolverFunc) {
	mu.Lock()
	defer mu.Unlock()
	if registry.GlobalRegistry.IsLocked(registry.KeyRegistryGraphQL) {
		panic("graphql/registry: locked (register only during init before first request)")
	}
	entries := getEntries()
	if _, ok := entries[name]; ok {
		panic("graphql/registry: duplicate " + name)
	}
	entries[name] = resolve
	registry.GlobalRegistry.SetGlobal(registry.KeyRegistryGraphQL, entries)
}

// Unregister removes a registration (for tests). Call UnlockForTesting first if registry is locked.
func Unregister(name string) {
	mu.Lock()
	defer mu.Unlock()
	registry.GlobalRegistry.UnlockForTesting(registry.KeyRegistryGraphQL)
	entries := getEntries()
	delete(entries, name)
	registry.GlobalRegistry.SetGlobal(registry.KeyRegistryGraphQL, entries)
}

// Resolve calls the resolver for the given field. Locks the graphql registry on first call.
func Resolve(ctx context.Context, field string, args map[string]interface{}) (interface{}, error) {
	if atomic.CompareAndSwapInt32(&graphqlLocked, 0, 1) {
		registry.GlobalRegistry.Lock(registry.KeyRegistryGraphQL)
	}
	// Read-only after init — no lock needed
	entries := getEntries()
	resolve, ok := entries[field]
	if !ok {
		return nil, fmt.Errorf("unknown extension: %s", field)
	}
	return resolve(ctx, args)
}

// Names returns all registered names.
func Names() []string {
	mu.Lock()
	defer mu.Unlock()
	entries := getEntries()
	names := make([]string, 0, len(entries))
	for n := range entries {
		names = append(names, n)
	}
	return names
}
