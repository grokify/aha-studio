package cache

import (
	"testing"
	"time"

	"github.com/grokify/aha-studio/aql/ast"
	"github.com/grokify/aha-studio/result"
)

func TestCacheSetGet(t *testing.T) {
	c := New(DefaultOptions())

	res := &result.Result{
		Entity: ast.EntityFeatures,
		Records: []result.Record{
			{"name": "Test"},
		},
	}

	c.Set("key1", res)

	got := c.Get("key1")
	if got == nil {
		t.Fatal("expected to get cached result")
	}
	if len(got.Records) != 1 {
		t.Errorf("expected 1 record, got %d", len(got.Records))
	}
}

func TestCacheGetMissing(t *testing.T) {
	c := New(DefaultOptions())

	got := c.Get("nonexistent")
	if got != nil {
		t.Error("expected nil for missing key")
	}
}

func TestCacheExpiration(t *testing.T) {
	c := New(Options{
		TTL:     50 * time.Millisecond,
		MaxSize: 100,
	})

	res := &result.Result{
		Entity:  ast.EntityFeatures,
		Records: []result.Record{{"name": "Test"}},
	}

	c.Set("key1", res)

	// Should be available immediately
	if c.Get("key1") == nil {
		t.Error("expected result to be available")
	}

	// Wait for expiration
	time.Sleep(60 * time.Millisecond)

	// Should be expired now
	if c.Get("key1") != nil {
		t.Error("expected result to be expired")
	}
}

func TestCacheDelete(t *testing.T) {
	c := New(DefaultOptions())

	res := &result.Result{
		Entity:  ast.EntityFeatures,
		Records: []result.Record{{"name": "Test"}},
	}

	c.Set("key1", res)
	if c.Get("key1") == nil {
		t.Fatal("expected result to be cached")
	}

	c.Delete("key1")
	if c.Get("key1") != nil {
		t.Error("expected result to be deleted")
	}
}

func TestCacheClear(t *testing.T) {
	c := New(DefaultOptions())

	res := &result.Result{
		Entity:  ast.EntityFeatures,
		Records: []result.Record{{"name": "Test"}},
	}

	c.Set("key1", res)
	c.Set("key2", res)
	c.Set("key3", res)

	if c.Size() != 3 {
		t.Errorf("expected size 3, got %d", c.Size())
	}

	c.Clear()

	if c.Size() != 0 {
		t.Errorf("expected size 0 after clear, got %d", c.Size())
	}
}

func TestCacheMaxSize(t *testing.T) {
	c := New(Options{
		TTL:     time.Hour,
		MaxSize: 3,
	})

	res := &result.Result{
		Entity:  ast.EntityFeatures,
		Records: []result.Record{{"name": "Test"}},
	}

	c.Set("key1", res)
	c.Set("key2", res)
	c.Set("key3", res)
	c.Set("key4", res) // Should evict oldest

	if c.Size() > 3 {
		t.Errorf("expected max size 3, got %d", c.Size())
	}
}

func TestCacheStats(t *testing.T) {
	c := New(Options{
		TTL:     time.Hour,
		MaxSize: 100,
	})

	res := &result.Result{
		Entity:  ast.EntityFeatures,
		Records: []result.Record{{"name": "Test"}},
	}

	c.Set("key1", res)
	c.Set("key2", res)

	stats := c.Stats()
	if stats.Size != 2 {
		t.Errorf("expected size 2, got %d", stats.Size)
	}
	if stats.MaxSize != 100 {
		t.Errorf("expected maxSize 100, got %d", stats.MaxSize)
	}
	if stats.TTL != time.Hour {
		t.Errorf("expected TTL 1h, got %v", stats.TTL)
	}
}

func TestKeyFromQuery(t *testing.T) {
	key1 := KeyFromQuery("FROM features", "PROD1")
	key2 := KeyFromQuery("FROM features", "PROD1")
	key3 := KeyFromQuery("FROM features", "PROD2")
	key4 := KeyFromQuery("FROM ideas", "PROD1")

	if key1 != key2 {
		t.Error("same query and product should produce same key")
	}
	if key1 == key3 {
		t.Error("different product should produce different key")
	}
	if key1 == key4 {
		t.Error("different query should produce different key")
	}

	// Key should be 16 hex characters
	if len(key1) != 16 {
		t.Errorf("expected key length 16, got %d", len(key1))
	}
}

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.TTL != 5*time.Minute {
		t.Errorf("expected default TTL 5m, got %v", opts.TTL)
	}
	if opts.MaxSize != 100 {
		t.Errorf("expected default MaxSize 100, got %d", opts.MaxSize)
	}
}
