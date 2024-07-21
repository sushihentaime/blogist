package common

import "testing"

func setupTestEnvironment(t *testing.T) (*Cache, func()) {
	t.Helper()

	// Set up the test environment
	cache := NewCache(0, 0)

	cleanup := func() {
		cache.Flush()
	}

	return cache, cleanup
}

func TestCache_Set(t *testing.T) {
	cache, cleanup := setupTestEnvironment(t)
	defer cleanup()

	cache.Set("key", "value")

	if _, ok := cache.Get("key"); !ok {
		t.Error("expected key to be set")
	}
}

func TestCache_Get(t *testing.T) {
	cache, cleanup := setupTestEnvironment(t)
	defer cleanup()

	cache.Set("key", "value")

	if _, ok := cache.Get("key"); !ok {
		t.Error("expected key to be set")
	}
}

func TestCache_Flush(t *testing.T) {
	cache, cleanup := setupTestEnvironment(t)
	defer cleanup()

	cache.Set("key", "value")
	cache.Flush()

	if _, ok := cache.Get("key"); ok {
		t.Error("expected cache to be flushed")
	}
}
