package cache

import (
	"sync"
	"time"
)

type Entry struct {
	Value      []byte
	Expiration time.Time
}

type CacheMap struct {
	m sync.Map
}

func (c *CacheMap) Store(key string, value Entry) {
	c.m.Store(key, value)
}

func (c *CacheMap) Load(key string) (Entry, bool) {
	value, ok := c.m.Load(key)
	if !ok {
		return Entry{}, ok
	}
	return value.(Entry), ok
}

func (c *CacheMap) Delete(key string) {
	c.m.Delete(key)
}

func (c *CacheMap) Range(f func(key string, value Entry) bool) {
	c.m.Range(func(key, value any) bool {
		return f(key.(string), value.(Entry))
	})
}

type Cache struct {
	cache *CacheMap
}

func NewCache() *Cache {
	return &Cache{
		cache: &CacheMap{},
	}
}

func (instance *Cache) Get(key string) []byte {
	entry, ok := instance.cache.Load(key)
	if !ok {
		return nil
	}

	if time.Now().After(entry.Expiration) {
		instance.cache.Delete(key)
		return nil
	}

	return entry.Value
}

func (instance *Cache) Store(key string, entry Entry) {
	instance.cache.Store(key, entry)
}

func (instance *Cache) Clear() {
	instance.cache.m.Clear()
}
