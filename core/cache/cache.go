package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Cache is a simple thread-safe key-value store using sync.Map.
type Cache struct {
	m sync.Map
}

var (
	once     sync.Once
	instance *Cache
)

func GetInstance() *Cache {
	once.Do(func() {
		instance = NewCache()
	})
	return instance
}

// NewCache creates a new Cache instance.
func NewCache() *Cache {
	return &Cache{}
}

// Set stores a value for a key.
func (c *Cache) Set(key, value interface{}) {
	var cache *Cache
	if instance != nil {
		cache = instance
	} else {
		cache = GetInstance()
	}
	cache.m.Store(key, value)
}

// Get retrieves a value for a key. Returns (value, true) if found, (nil, false) otherwise.
func (c *Cache) Get(key interface{}) (interface{}, bool) {
	var cache *Cache
	if instance != nil {
		cache = instance
	} else {
		cache = GetInstance()
	}
	return cache.m.Load(key)
}

// Delete removes a key from the cache.
func (c *Cache) Delete(key interface{}) {
	var cache *Cache
	if instance != nil {
		cache = instance
	} else {
		cache = GetInstance()
	}
	cache.m.Delete(key)
}

func makeCompositeKey(keys ...interface{}) string {
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = fmt.Sprintf("%v", k)
	}
	return strings.Join(parts, "|")
}

func (c *Cache) SetN(keysAndValue ...interface{}) {
	if len(keysAndValue) < 2 {
		// Not enough arguments: need at least one key and a value
		return
	}
	value := keysAndValue[len(keysAndValue)-1]
	keys := keysAndValue[:len(keysAndValue)-1]
	c.Set(makeCompositeKey(keys...), value)
}

func (c *Cache) GetN(keys ...interface{}) (interface{}, bool) {
	return c.Get(makeCompositeKey(keys...))
}

func (c *Cache) DeleteN(keys ...interface{}) {
	c.Delete(makeCompositeKey(keys...))
}

// GetMany retrieves values for multiple keys. If a key is not found, the value is nil.
func (c *Cache) GetMany(keys ...interface{}) []interface{} {
	var cache *Cache
	if instance != nil {
		cache = instance
	} else {
		cache = GetInstance()
	}
	results := make([]interface{}, len(keys))
	for i, key := range keys {
		v, ok := cache.m.Load(key)
		if ok {
			results[i] = v
		} else {
			results[i] = nil
		}
	}
	return results
}

// DumpToFile saves all cache key-values to a file as JSON.
func (c *Cache) DumpToFile(filename string) error {
	var cache *Cache
	if instance != nil {
		cache = instance
	} else {
		cache = GetInstance()
	}
	m := make(map[interface{}]interface{})
	cache.m.Range(func(key, value interface{}) bool {
		m[key] = value
		return true
	})
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

// RestoreFromFile loads key-values from a file and populates the cache.
func (c *Cache) RestoreFromFile(filename string) error {
	var cache *Cache
	if instance != nil {
		cache = instance
	} else {
		cache = GetInstance()
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	m := make(map[string]interface{})
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	for k, v := range m {
		cache.m.Store(k, v)
	}
	return nil
}

// IterateFilter iterates over all cache entries and returns a slice of values for which the callback returns true.
func (c *Cache) IterateFilter(filter func(key, value interface{}) bool) []interface{} {
	var cache *Cache
	if instance != nil {
		cache = instance
	} else {
		cache = GetInstance()
	}
	var results []interface{}
	cache.m.Range(func(key, value interface{}) bool {
		if filter(key, value) {
			results = append(results, value)
		}
		return true
	})
	return results
}

/*
Usage Example:

cache := cache.NewCache()
cache.Set("foo", 123)
v, ok := cache.Get("foo") // v == 123, ok == true
cache.Delete("foo")
*/
