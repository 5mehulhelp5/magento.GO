package cache

import (
	"path/filepath"
	"testing"
)

func TestNewCache(t *testing.T) {
	c := NewCache()
	if c == nil {
		t.Fatal("NewCache returned nil")
	}
}

func TestGetInstance(t *testing.T) {
	inst := GetInstance()
	if inst == nil {
		t.Fatal("GetInstance returned nil")
	}
	if GetInstance() != inst {
		t.Error("GetInstance should return same instance")
	}
}

func TestSet_Get(t *testing.T) {
	c := GetInstance()
	key := "test-set-get"
	c.Set(key, "val", 0, nil)
	got, ok := c.Get(key)
	if !ok {
		t.Fatal("Get: want true")
	}
	if got != "val" {
		t.Errorf("Get = %v, want val", got)
	}
	c.Delete(key)
}

func TestGet_Missing(t *testing.T) {
	c := GetInstance()
	_, ok := c.Get("nonexistent-key-xyz")
	if ok {
		t.Error("Get missing key: want false")
	}
}

func TestDelete(t *testing.T) {
	c := GetInstance()
	key := "test-delete"
	c.Set(key, "x", 0, nil)
	c.Delete(key)
	_, ok := c.Get(key)
	if ok {
		t.Error("Delete: key should be gone")
	}
}

func TestGetOrDefault(t *testing.T) {
	c := GetInstance()
	key := "test-default"
	def := "default"
	if got := c.GetOrDefault(key, def); got != def {
		t.Errorf("GetOrDefault missing = %v, want %v", got, def)
	}
	c.Set(key, "stored", 0, nil)
	if got := c.GetOrDefault(key, def); got != "stored" {
		t.Errorf("GetOrDefault found = %v, want stored", got)
	}
	c.Delete(key)
}

func TestDeleteMany(t *testing.T) {
	c := GetInstance()
	c.Set("dm1", 1, 0, nil)
	c.Set("dm2", 2, 0, nil)
	c.DeleteMany("dm1", "dm2")
	if _, ok := c.Get("dm1"); ok {
		t.Error("DeleteMany: dm1 should be gone")
	}
	if _, ok := c.Get("dm2"); ok {
		t.Error("DeleteMany: dm2 should be gone")
	}
}

func TestSetN_GetN_DeleteN(t *testing.T) {
	c := GetInstance()
	c.SetN([]interface{}{"a", "b"}, "composite-val", 0, nil)
	got, ok := c.GetN("a", "b")
	if !ok || got != "composite-val" {
		t.Errorf("GetN = %v, %v; want composite-val, true", got, ok)
	}
	c.DeleteN("a", "b")
	_, ok = c.GetN("a", "b")
	if ok {
		t.Error("DeleteN: key should be gone")
	}
}

func TestGetMany(t *testing.T) {
	c := GetInstance()
	c.Set("gm1", "v1", 0, nil)
	c.Set("gm2", "v2", 0, nil)
	results := c.GetMany("gm1", "gm2", "gm-missing")
	if len(results) != 3 {
		t.Fatalf("GetMany len = %d, want 3", len(results))
	}
	if results[0] != "v1" {
		t.Errorf("GetMany gm1 = %v, want v1", results[0])
	}
	if results[1] != "v2" {
		t.Errorf("GetMany gm2 = %v, want v2", results[1])
	}
	if results[2] != nil {
		t.Error("GetMany gm-missing: want nil")
	}
	c.DeleteMany("gm1", "gm2")
}

func TestTagKey_GetKeysByTag_DeleteByTag(t *testing.T) {
	c := GetInstance()
	key1, key2 := "tag-k1", "tag-k2"
	c.Set(key1, "v1", 0, nil)
	c.Set(key2, "v2", 0, nil)
	c.TagKey(key1, []string{"t1"})
	c.TagKey(key2, []string{"t1"})

	keys := c.GetKeysByTag("t1")
	if len(keys) != 2 {
		t.Errorf("GetKeysByTag = %d keys, want 2", len(keys))
	}

	c.DeleteByTag("t1")
	if _, ok := c.Get(key1); ok {
		t.Error("DeleteByTag: key1 should be gone")
	}
	if _, ok := c.Get(key2); ok {
		t.Error("DeleteByTag: key2 should be gone")
	}
}

func TestDelete_RemovesFromTagIndex(t *testing.T) {
	c := GetInstance()
	key := "del-tag-key"
	c.Set(key, "v", 0, nil)
	c.TagKey(key, []string{"t2"})
	c.Delete(key)
	keys := c.GetKeysByTag("t2")
	if len(keys) != 0 {
		t.Errorf("GetKeysByTag after Delete = %d keys, want 0", len(keys))
	}
}

func TestIterateFilter(t *testing.T) {
	c := GetInstance()
	c.Set("if1", 10, 0, nil)
	c.Set("if2", 20, 0, nil)
	c.Set("if3", 30, 0, nil)
	defer c.DeleteMany("if1", "if2", "if3")

	results := c.IterateFilter(func(key, value interface{}) bool {
		return key == "if1" || key == "if3"
	})
	if len(results) != 2 {
		t.Errorf("IterateFilter = %d results, want 2", len(results))
	}
	// IterateFilter returns unwrapped values (10, 30)
	has10, has30 := false, false
	for _, v := range results {
		if v == 10 {
			has10 = true
		}
		if v == 30 {
			has30 = true
		}
	}
	if !has10 || !has30 {
		t.Errorf("IterateFilter values = %v, want 10 and 30", results)
	}
}

func TestDumpToFile_RestoreFromFile(t *testing.T) {
	c := GetInstance()
	key := "dump-key"
	c.Set(key, "dump-val", 0, nil)
	defer c.Delete(key)

	tmp := filepath.Join(t.TempDir(), "cache.json")
	if err := c.DumpToFile(tmp); err != nil {
		t.Fatalf("DumpToFile: %v", err)
	}

	c.Delete(key)
	if err := c.RestoreFromFile(tmp); err != nil {
		t.Fatalf("RestoreFromFile: %v", err)
	}
	got, ok := c.Get(key)
	if !ok || got != "dump-val" {
		t.Errorf("after restore Get = %v, ok=%v; want dump-val, true", got, ok)
	}
	c.Delete(key)
}

func TestRestoreFromFile_MissingFile(t *testing.T) {
	c := GetInstance()
	err := c.RestoreFromFile("/nonexistent/path/cache.json")
	if err == nil {
		t.Error("RestoreFromFile missing file: want error")
	}
}
