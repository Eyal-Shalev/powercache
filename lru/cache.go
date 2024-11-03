package lru

import (
	"iter"
	"maps"
	"sync"
	"time"

	"github.com/Eyal-Shalev/powercache"
	"github.com/Eyal-Shalev/powercache/internal/container/list"
)

type cacheEntry[K comparable, V any] struct {
	key   K
	value V
}

type Cache[K comparable, V any] struct {
	entries *list.List[cacheEntry[K, V]]
	m       *sync.RWMutex
}

func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.m.RLock()
	defer c.m.RUnlock()
	return c.unsafeGet(key)
}

func (c *Cache[K, V]) unsafeGet(key K) (V, bool) {
	var zero V
	root := c.entries.Front()
	if root == nil {
		return zero, false
	}
	entry := c.findEntry(key)
	if entry == nil {
		return zero, false
	}
	c.entries.MoveToFront(entry)
	return entry.Value.value, true
}

func (c *Cache[K, V]) findEntry(key K) *list.Element[cacheEntry[K, V]] {
	root := c.entries.Front()
	if root == nil {
		return nil
	}

	for cur := root; cur != nil; cur = cur.Next() {
		if cur.Value.key == key {
			return cur
		}
	}
	return nil
}

func (c *Cache[K, V]) Set(key K, value V) {
	c.m.Lock()
	defer c.m.Unlock()
	c.unsafeSet(key, value)
}

func (c *Cache[K, V]) unsafeSet(key K, value V) {
	if entry := c.findEntry(key); entry != nil {
		entry.Value.value = value
		c.entries.MoveToFront(entry)
	} else {
		c.entries.InsertBefore(cacheEntry[K, V]{key, value}, c.entries.Front())
	}
}

func (c *Cache[K, V]) Delete(key K) {
	c.m.Lock()
	defer c.m.Unlock()
	if entry := c.findEntry(key); entry != nil {
		c.entries.Remove(entry)
	}
}

func (c *Cache[K, V]) SetFromMap(data map[K]V) {
	c.SetFromIter(maps.All(data))
}

func (c *Cache[K, V]) SetFromIter(data iter.Seq2[K, V]) {
	c.m.Lock()
	defer c.m.Unlock()
	for k, v := range data {
		c.unsafeSet(k, v)
	}
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

var _ powercache.Cache[bool, bool] = (*Cache[bool, bool])(nil)
var _ powercache.Doable[bool, bool] = (*Cache[bool, bool])(nil)
var _ powercache.MultiSetter[bool, bool] = (*Cache[bool, bool])(nil)

func New[K comparable, V any](ttl time.Duration) *Cache[K, V] {
	return &Cache[K, V]{
		entries: list.New[cacheEntry[K, V]](),
		m:       new(sync.RWMutex),
	}
}
