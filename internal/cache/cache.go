package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Entry represents a cached value with expiration
type Entry struct {
	Value      []byte
	Expiration time.Time
}

// IsExpired checks if the entry has expired
func (e Entry) IsExpired() bool {
	return time.Now().After(e.Expiration)
}

// Cache provides thread-safe in-memory caching with TTL
type Cache struct {
	store   sync.Map
	logger  *zap.Logger
	cleaner *time.Ticker
	stopCh  chan struct{}
}

// New creates a new cache with automatic cleanup
func New(cleanupInterval time.Duration, logger *zap.Logger) *Cache {
	if logger == nil {
		logger = zap.NewNop()
	}

	c := &Cache{
		logger:  logger,
		cleaner: time.NewTicker(cleanupInterval),
		stopCh:  make(chan struct{}),
	}

	// Start background cleanup
	go c.cleanupLoop()

	return c
}

// Get retrieves a value from cache
func (c *Cache) Get(key string) ([]byte, bool) {
	value, ok := c.store.Load(key)
	if !ok {
		c.logger.Debug("cache miss", zap.String("key", key))
		return nil, false
	}

	entry := value.(Entry)
	if entry.IsExpired() {
		c.store.Delete(key)
		c.logger.Debug("cache expired", zap.String("key", key))
		return nil, false
	}

	c.logger.Debug("cache hit", zap.String("key", key))
	return entry.Value, true
}

// Set stores a value in cache with TTL
func (c *Cache) Set(key string, value []byte, ttl time.Duration) {
	entry := Entry{
		Value:      value,
		Expiration: time.Now().Add(ttl),
	}

	c.store.Store(key, entry)
	c.logger.Debug("cache set",
		zap.String("key", key),
		zap.Int("size_bytes", len(value)),
		zap.Duration("ttl", ttl))
}

// SetJSON stores a JSON-encodable value in cache
func (c *Cache) SetJSON(key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	c.Set(key, data, ttl)
	return nil
}

// GetJSON retrieves and unmarshals a JSON value from cache
func (c *Cache) GetJSON(key string, dest interface{}) (bool, error) {
	data, ok := c.Get(key)
	if !ok {
		return false, nil
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return false, fmt.Errorf("unmarshal json: %w", err)
	}

	return true, nil
}

// Delete removes a value from cache
func (c *Cache) Delete(key string) {
	c.store.Delete(key)
	c.logger.Debug("cache delete", zap.String("key", key))
}

// Clear removes all entries from cache
func (c *Cache) Clear() {
	c.store.Range(func(key, value interface{}) bool {
		c.store.Delete(key)
		return true
	})
	c.logger.Info("cache cleared")
}

// Close stops the cleanup goroutine
func (c *Cache) Close() {
	close(c.stopCh)
	c.cleaner.Stop()
	c.logger.Info("cache closed")
}

// cleanupLoop periodically removes expired entries
func (c *Cache) cleanupLoop() {
	for {
		select {
		case <-c.cleaner.C:
			c.cleanup()
		case <-c.stopCh:
			return
		}
	}
}

// cleanup removes expired entries
func (c *Cache) cleanup() {
	count := 0
	c.store.Range(func(key, value interface{}) bool {
		entry := value.(Entry)
		if entry.IsExpired() {
			c.store.Delete(key)
			count++
		}
		return true
	})

	if count > 0 {
		c.logger.Debug("cache cleanup", zap.Int("expired_entries", count))
	}
}

// Stats returns cache statistics
func (c *Cache) Stats() map[string]int {
	total := 0
	expired := 0

	c.store.Range(func(key, value interface{}) bool {
		total++
		entry := value.(Entry)
		if entry.IsExpired() {
			expired++
		}
		return true
	})

	return map[string]int{
		"total":   total,
		"expired": expired,
		"active":  total - expired,
	}
}

// KeyBuilder helps build consistent cache keys
type KeyBuilder struct{}

// FromRequest builds a cache key from request parameters
func (kb KeyBuilder) FromRequest(method, path string, params map[string]string) string {
	h := sha256.New()
	h.Write([]byte(method))
	h.Write([]byte(path))

	// Sort params for consistency
	for k, v := range params {
		h.Write([]byte(k))
		h.Write([]byte(v))
	}

	return hex.EncodeToString(h.Sum(nil))
}

// FromString builds a cache key from a string
func (kb KeyBuilder) FromString(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		h.Write([]byte(part))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// Cacheable wraps a function with caching
type Cacheable struct {
	cache  *Cache
	ttl    time.Duration
	logger *zap.Logger
}

// NewCacheable creates a new cacheable wrapper
func NewCacheable(cache *Cache, ttl time.Duration, logger *zap.Logger) *Cacheable {
	return &Cacheable{
		cache:  cache,
		ttl:    ttl,
		logger: logger,
	}
}

// Do executes a function with caching
func (c *Cacheable) Do(ctx context.Context, key string, fn func(ctx context.Context) (interface{}, error)) (interface{}, error) {
	// Try to get from cache
	var result interface{}
	ok, err := c.cache.GetJSON(key, &result)
	if err != nil {
		c.logger.Warn("cache get error", zap.Error(err))
	}
	if ok {
		return result, nil
	}

	// Execute function
	result, err = fn(ctx)
	if err != nil {
		return nil, err
	}

	// Store in cache
	if err := c.cache.SetJSON(key, result, c.ttl); err != nil {
		c.logger.Warn("cache set error", zap.Error(err))
	}

	return result, nil
}
