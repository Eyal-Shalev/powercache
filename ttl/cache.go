package ttl

import (
	"iter"
	"maps"
	"sync"
	"time"

	"github.com/Eyal-Shalev/powercache"
)

type cacheEntry[V any] struct {
	value    V
	expireAt time.Time
}

type Cache[K comparable, V any] struct {
	data map[K]cacheEntry[V]
	ttl  time.Duration
	m    *sync.RWMutex
}

func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.m.RLock()
	defer c.m.RUnlock()
	return c.unsafeGet(key)
}

func (c *Cache[K, V]) unsafeGet(key K) (V, bool) {
	var zero V
	entry, ok := c.data[key]
	if !ok {
		return zero, false
	}
	if time.Now().After(entry.expireAt) {
		return zero, false
	}
	return entry.value, true
}

func (c *Cache[K, V]) Set(key K, value V) {
	c.m.Lock()
	defer c.m.Unlock()
	c.unsafeSet(key, value)
}

func (c *Cache[K, V]) unsafeSet(key K, value V) {
	c.data[key] = cacheEntry[V]{value, time.Now().Add(c.ttl)}
}

func (c *Cache[K, V]) Delete(key K) {
	c.m.Lock()
	defer c.m.Unlock()
	delete(c.data, key)
}

func (c *Cache[K, V]) Do(key K, fn func() (V, error)) (V, error) {
	// Try the faster [Cache.Get] which uses [RWMutex.RLock]
	if value, ok := c.Get(key); ok {
		return value, nil
	}

	// Lock for write
	c.m.Lock()
	defer c.m.Unlock()

	// Check if between [Cache.Get] and [Cache.m.Lock] the data was added.
	if value, ok := c.unsafeGet(key); ok {
		return value, nil
	}

	value, err := fn()
	if err != nil {
		return value, err
	}
	c.unsafeSet(key, value)

	return value, err
}

func (c *Cache[K, V]) SetFromMap(data map[K]V) {
	c.SetFromIter(maps.All(data))
}

func (c *Cache[K, V]) SetFromIter(data iter.Seq2[K, V]) {
	c.m.Lock()
	defer c.m.Unlock()
	for key, value := range data {
		c.unsafeSet(key, value)
	}
}

var _ powercache.Cache[bool, bool] = (*Cache[bool, bool])(nil)
var _ powercache.Doable[bool, bool] = (*Cache[bool, bool])(nil)
var _ powercache.MultiSetter[bool, bool] = (*Cache[bool, bool])(nil)

func New[K comparable, V any](ttl time.Duration) *Cache[K, V] {
	return &Cache[K, V]{
		data: make(map[K]cacheEntry[V]),
		m:    new(sync.RWMutex),
		ttl:  ttl,
	}
}
