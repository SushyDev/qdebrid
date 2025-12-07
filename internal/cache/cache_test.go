package cache

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestCacheGetSet(t *testing.T) {
	logger := zap.NewNop()
	c := New(1*time.Minute, logger)
	defer c.Close()

	key := "test_key"
	value := []byte("test_value")
	ttl := 1 * time.Second

	// Test Set
	c.Set(key, value, ttl)

	// Test Get - should exist
	got, ok := c.Get(key)
	if !ok {
		t.Error("Get() expected to find key, got not found")
	}
	if string(got) != string(value) {
		t.Errorf("Get() = %v, want %v", string(got), string(value))
	}
}

func TestCacheExpiration(t *testing.T) {
	logger := zap.NewNop()
	c := New(100*time.Millisecond, logger)
	defer c.Close()

	key := "expiring_key"
	value := []byte("expiring_value")
	ttl := 200 * time.Millisecond

	// Set with TTL
	c.Set(key, value, ttl)

	// Should exist immediately
	if _, ok := c.Get(key); !ok {
		t.Error("Get() expected key to exist immediately after Set")
	}

	// Wait for expiration
	time.Sleep(250 * time.Millisecond)

	// Should not exist after expiration
	if got, ok := c.Get(key); ok {
		t.Errorf("Get() expected key to expire, but got: %v", string(got))
	}
}

func TestCacheDelete(t *testing.T) {
	logger := zap.NewNop()
	c := New(1*time.Minute, logger)
	defer c.Close()

	key := "delete_key"
	value := []byte("delete_value")

	c.Set(key, value, 1*time.Minute)

	// Verify exists
	if _, ok := c.Get(key); !ok {
		t.Error("Get() expected key to exist before Delete")
	}

	// Delete
	c.Delete(key)

	// Verify deleted
	if got, ok := c.Get(key); ok {
		t.Errorf("Get() expected key to be deleted, but got: %v", string(got))
	}
}

func TestCacheClear(t *testing.T) {
	logger := zap.NewNop()
	c := New(1*time.Minute, logger)
	defer c.Close()

	// Add multiple entries
	c.Set("key1", []byte("value1"), 1*time.Minute)
	c.Set("key2", []byte("value2"), 1*time.Minute)
	c.Set("key3", []byte("value3"), 1*time.Minute)

	// Verify they exist
	if _, ok := c.Get("key1"); !ok {
		t.Error("Expected key1 to exist")
	}

	// Clear all
	c.Clear()

	// Verify all are gone
	if _, ok := c.Get("key1"); ok {
		t.Error("Expected key1 to be cleared")
	}
	if _, ok := c.Get("key2"); ok {
		t.Error("Expected key2 to be cleared")
	}
	if _, ok := c.Get("key3"); ok {
		t.Error("Expected key3 to be cleared")
	}
}

func TestCacheJSON(t *testing.T) {
	logger := zap.NewNop()
	c := New(1*time.Minute, logger)
	defer c.Close()

	type TestStruct struct {
		Name  string
		Count int
	}

	key := "json_key"
	value := TestStruct{Name: "test", Count: 42}
	ttl := 1 * time.Minute

	// Set JSON
	err := c.SetJSON(key, value, ttl)
	if err != nil {
		t.Fatalf("SetJSON() error = %v", err)
	}

	// Get JSON
	var got TestStruct
	ok, err := c.GetJSON(key, &got)
	if err != nil {
		t.Fatalf("GetJSON() error = %v", err)
	}
	if !ok {
		t.Error("GetJSON() expected to find key")
	}

	if got.Name != value.Name {
		t.Errorf("GetJSON() Name = %v, want %v", got.Name, value.Name)
	}
	if got.Count != value.Count {
		t.Errorf("GetJSON() Count = %v, want %v", got.Count, value.Count)
	}
}

func TestCacheJSONNotFound(t *testing.T) {
	logger := zap.NewNop()
	c := New(1*time.Minute, logger)
	defer c.Close()

	type TestStruct struct {
		Name string
	}

	var got TestStruct
	ok, err := c.GetJSON("nonexistent", &got)
	if err != nil {
		t.Errorf("GetJSON() unexpected error = %v", err)
	}
	if ok {
		t.Error("GetJSON() expected not to find nonexistent key")
	}
}

func TestCacheAutomaticCleanup(t *testing.T) {
	logger := zap.NewNop()
	c := New(100*time.Millisecond, logger)
	defer c.Close()

	// Add entries with short TTL
	c.Set("expire1", []byte("value1"), 50*time.Millisecond)
	c.Set("expire2", []byte("value2"), 50*time.Millisecond)
	c.Set("expire3", []byte("value3"), 50*time.Millisecond)

	// Wait for expiration and cleanup
	time.Sleep(200 * time.Millisecond)

	// Check stats - expired entries should be cleaned up
	stats := c.Stats()
	if stats["expired"] > 0 {
		// If expired entries exist, they haven't been cleaned up yet
		// Wait a bit more for cleanup goroutine
		time.Sleep(150 * time.Millisecond)
		stats = c.Stats()
	}

	// After cleanup, active should be 0 (or at least less than 3)
	if stats["active"] == 3 {
		t.Errorf("Stats() expected cleanup to remove expired entries, got active=%v", stats["active"])
	}
}

func TestCacheStats(t *testing.T) {
	logger := zap.NewNop()
	c := New(1*time.Minute, logger)
	defer c.Close()

	// Add some entries
	c.Set("key1", []byte("value1"), 1*time.Minute)
	c.Set("key2", []byte("value2"), 1*time.Minute)
	c.Set("key3", []byte("value3"), 100*time.Millisecond) // Will expire soon

	stats := c.Stats()

	if stats["total"] != 3 {
		t.Errorf("Stats() total = %v, want 3", stats["total"])
	}

	// Wait for one to expire
	time.Sleep(150 * time.Millisecond)

	stats = c.Stats()
	if stats["expired"] < 1 {
		t.Error("Stats() expected at least 1 expired entry")
	}
}

func TestKeyBuilderFromString(t *testing.T) {
	kb := KeyBuilder{}

	key1 := kb.FromString("part1", "part2", "part3")
	key2 := kb.FromString("part1", "part2", "part3")
	key3 := kb.FromString("part1", "part2", "different")

	// Same inputs should produce same key
	if key1 != key2 {
		t.Error("FromString() expected same key for same input")
	}

	// Different inputs should produce different key
	if key1 == key3 {
		t.Error("FromString() expected different key for different input")
	}

	// Key should be non-empty hash
	if len(key1) == 0 {
		t.Error("FromString() produced empty key")
	}

	// Key should be hex-encoded (64 chars for SHA256)
	if len(key1) != 64 {
		t.Errorf("FromString() key length = %v, want 64 (SHA256 hex)", len(key1))
	}
}

func TestKeyBuilderFromRequest(t *testing.T) {
	kb := KeyBuilder{}

	// Test with different parameters
	key1 := kb.FromRequest("GET", "/api/v1/test", map[string]string{
		"param1": "value1",
		"param2": "value2",
	})

	key2 := kb.FromRequest("GET", "/api/v1/test", map[string]string{
		"param1": "value1",
		"param2": "value2",
	})

	key3 := kb.FromRequest("POST", "/api/v1/test", map[string]string{
		"param1": "value1",
		"param2": "value2",
	})

	// Same inputs should produce same key
	if key1 != key2 {
		t.Error("FromRequest() expected same key for same input")
	}

	// Different method should produce different key
	if key1 == key3 {
		t.Error("FromRequest() expected different key for different method")
	}

	// Key should be non-empty
	if len(key1) == 0 {
		t.Error("FromRequest() produced empty key")
	}
}

func TestConcurrentAccess(t *testing.T) {
	logger := zap.NewNop()
	c := New(1*time.Minute, logger)
	defer c.Close()

	done := make(chan bool)
	iterations := 100

	// Writer goroutine
	go func() {
		for i := 0; i < iterations; i++ {
			c.Set("concurrent_key", []byte("value"), 1*time.Minute)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < iterations; i++ {
			c.Get("concurrent_key")
		}
		done <- true
	}()

	// Delete goroutine
	go func() {
		for i := 0; i < iterations; i++ {
			c.Delete("concurrent_key")
		}
		done <- true
	}()

	// Wait for all goroutines
	<-done
	<-done
	<-done

	// If we get here without deadlock or panic, test passes
}
