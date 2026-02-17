package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// Cache is a simple thread-safe key-value store using sync.Map.
type Cache struct {
	m sync.Map
	// tagIndex maps tag string to a set of keys (as map[interface{}]struct{})
	tagIndex sync.Map // map[string]map[interface{}]struct{}
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

// cacheItem holds a value and its expiration time.
type cacheItem struct {
	Value     interface{}
	ExpiresAt int64 // Unix timestamp in nanoseconds; 0 means no expiration
}

// Set stores a value for a key with an optional TTL (in seconds) and optional tags (as a string slice). If ttl is 0, the value does not expire. Tags can be provided as a []string.
func (c *Cache) Set(key, value interface{}, ttl int64, tags []string) {
	if c == nil {
		c = GetInstance()
	}
	var expiresAt int64
	if ttl > 0 {
		expiresAt = time.Now().Add(time.Duration(ttl) * time.Second).UnixNano()
	}
	c.m.Store(key, cacheItem{Value: value, ExpiresAt: expiresAt})
	if len(tags) > 0 {
		c.TagKey(key, tags)
	}
}

// Get retrieves a value for a key. Returns (value, true) if found and not expired, (nil, false) otherwise.
func (c *Cache) Get(key interface{}) (interface{}, bool) {
	if c == nil {
		c = GetInstance()
	}
	v, ok := c.m.Load(key)
	if !ok {
		return nil, false
	}
	if item, isItem := v.(cacheItem); isItem {
		if item.ExpiresAt > 0 && time.Now().UnixNano() > item.ExpiresAt {
			c.m.Delete(key)
			c.removeKeyFromTagIndex(key)
			return nil, false
		}
		return item.Value, true
	}
	// Fallback for legacy values (no TTL)
	return v, true
}

// GetOrDefault retrieves a value for a key. Returns the value if found, otherwise returns the default value.
func (c *Cache) GetOrDefault(key, defaultValue interface{}) interface{} {
	v, ok := c.Get(key)
	if ok {
		return v
	}
	return defaultValue
}

// Delete removes a key from the cache.
func (c *Cache) Delete(key interface{}) {
	if c == nil {
		c = GetInstance()
	}
	c.m.Delete(key)
	c.removeKeyFromTagIndex(key)
}

// removeKeyFromTagIndex removes a key from all tag indices.
func (c *Cache) removeKeyFromTagIndex(key interface{}) {
	c.tagIndex.Range(func(_, val interface{}) bool {
		val.(*sync.Map).Delete(key)
		return true
	})
}

// DeleteMany removes multiple keys from the cache.
func (c *Cache) DeleteMany(keys ...interface{}) {
	if c == nil {
		c = GetInstance()
	}
	for _, key := range keys {
		c.m.Delete(key)
		c.removeKeyFromTagIndex(key)
	}
}

func makeCompositeKey(keys ...interface{}) string {
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = fmt.Sprintf("%v", k)
	}
	return strings.Join(parts, "|")
}

// SetN stores a value for a composite key with an optional TTL (in seconds) and optional tags (as a string slice).
func (c *Cache) SetN(keys []interface{}, value interface{}, ttl int64, tags []string) {
	c.Set(makeCompositeKey(keys...), value, ttl, tags)
}

// GetN retrieves a value for a composite key. Returns (value, true) if found and not expired, (nil, false) otherwise.
func (c *Cache) GetN(keys ...interface{}) (interface{}, bool) {
	return c.Get(makeCompositeKey(keys...))
}

func (c *Cache) DeleteN(keys ...interface{}) {
	c.Delete(makeCompositeKey(keys...))
}

// GetMany retrieves values for multiple keys. If a key is not found, the value is nil.
func (c *Cache) GetMany(keys ...interface{}) []interface{} {
	if c == nil {
		c = GetInstance()
	}
	results := make([]interface{}, len(keys))
	for i, key := range keys {
		v, ok := c.m.Load(key)
		if !ok {
			results[i] = nil
			continue
		}
		if item, isItem := v.(cacheItem); isItem {
			if item.ExpiresAt > 0 && time.Now().UnixNano() > item.ExpiresAt {
				c.m.Delete(key)
				c.removeKeyFromTagIndex(key)
				results[i] = nil
			} else {
				results[i] = item.Value
			}
		} else {
			results[i] = v
		}
	}
	return results
}

// DumpToFile saves all cache key-values to a file as JSON.
func (c *Cache) DumpToFile(filename string) error {
	if c == nil {
		c = GetInstance()
	}
	m := make(map[string]interface{})
	c.m.Range(func(key, value interface{}) bool {
		k := fmt.Sprintf("%v", key)
		if item, isItem := value.(cacheItem); isItem {
			m[k] = map[string]interface{}{"Value": item.Value, "ExpiresAt": item.ExpiresAt}
		} else {
			m[k] = value
		}
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
	if c == nil {
		c = GetInstance()
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
		if vm, ok := v.(map[string]interface{}); ok {
			if val, hasVal := vm["Value"]; hasVal {
				var exp int64
				if e, ok := vm["ExpiresAt"].(float64); ok {
					exp = int64(e)
				}
				c.m.Store(k, cacheItem{Value: val, ExpiresAt: exp})
				continue
			}
		}
		c.m.Store(k, v)
	}
	return nil
}

// IterateFilter iterates over all cache entries and returns a slice of values for which the callback returns true.
func (c *Cache) IterateFilter(filter func(key, value interface{}) bool) []interface{} {
	if c == nil {
		c = GetInstance()
	}
	var results []interface{}
	c.m.Range(func(key, value interface{}) bool {
		var v interface{}
		if item, isItem := value.(cacheItem); isItem {
			if item.ExpiresAt > 0 && time.Now().UnixNano() > item.ExpiresAt {
				c.m.Delete(key)
				c.removeKeyFromTagIndex(key)
				return true
			}
			v = item.Value
		} else {
			v = value
		}
		if filter(key, v) {
			results = append(results, v)
		}
		return true
	})
	return results
}

// TagKey assigns one or more tags (as a string slice) to a cache key.
func (c *Cache) TagKey(key interface{}, tags []string) {
	if c == nil {
		return
	}
	for _, tag := range tags {
		val, _ := c.tagIndex.LoadOrStore(tag, &sync.Map{})
		km := val.(*sync.Map)
		km.Store(key, struct{}{})
	}
}

// UntagKey removes one or more tags (as a string slice) from a cache key.
func (c *Cache) UntagKey(key interface{}, tags []string) {
	if c == nil {
		return
	}
	for _, tag := range tags {
		if val, ok := c.tagIndex.Load(tag); ok {
			km := val.(*sync.Map)
			km.Delete(key)
		}
	}
}

// GetKeysByTag returns a slice of all keys assigned to a tag.
func (c *Cache) GetKeysByTag(tag string) []interface{} {
	if c == nil {
		c = GetInstance()
	}
	var keys []interface{}
	if val, ok := c.tagIndex.Load(tag); ok {
		km := val.(*sync.Map)
		km.Range(func(key, _ interface{}) bool {
			keys = append(keys, key)
			return true
		})
	}
	return keys
}

// DeleteByTag deletes all cache entries assigned to a tag.
func (c *Cache) DeleteByTag(tag string) {
	if c == nil {
		c = GetInstance()
	}
	if val, ok := c.tagIndex.Load(tag); ok {
		km := val.(*sync.Map)
		km.Range(func(key, _ interface{}) bool {
			c.Delete(key)
			return true
		})
		c.tagIndex.Delete(tag)
	}
}

/*
Updated Usage Example:

cache := cache.GetInstance()
cache.Set("foo", 123, 0, nil) // no expiration, no tags
cache.Set("bar", 456, 10, nil) // expires in 10s, no tags
cache.Set("baz", 789, 0, []string{"user", "session"}) // no expiration, tags: user, session
cache.Set("qux", 999, 5, []string{"user"}) // expires in 5s, tag: user

// For composite keys:
cache.SetN([]interface{}{ "a", "b" }, "val", 0, nil) // no expiration, no tags
cache.SetN([]interface{}{ "a", "b" }, "val", 5, nil) // expires in 5s, no tags
cache.SetN([]interface{}{ "a", "b" }, "val", 0, []string{"tag1", "tag2"}) // no expiration, tags: tag1, tag2
*/
