package cache

import (
	"sync"
	"time"
)

type CacheItem struct {
	Value			interface{}
	Expiration		int64
	LastAccess 		int64
}

type Cache struct {
	items map[string]CacheItem
	mu sync.RWMutex
	evictionType string // "lru" or "lfu"
	maxItems int
	accessCount map[string]int // Track frequency of access for LFU
}

// Creates a new cache instance and returns a pointer to that cache
func NewCache (evictionType string, maxItems int) *Cache {
	return &Cache{
		items: make(map[string]CacheItem),
		evictionType: evictionType,
		maxItems: maxItems,
		accessCount: make(map[string]int), 
	}
}

// Adds a new key-value pair to the cache
func (c *Cache) Set (key string, value interface{}, ttl time.Duration) {
	c.mu.Lock();
	defer c.mu.Unlock();

	// Check if we need to evict an item
	if (len(c.items) >= c.maxItems) {
		c.evict();
	}

	expiration := time.Now().Add(ttl).UnixNano();
	c.items[key] = CacheItem{
		Value: value,
		Expiration: expiration,
		LastAccess: time.Now().UnixNano(),
	}
}

// Get a value from the cache
func (c *Cache) Get (key string) (interface{}, bool) {
	c.mu.Lock();
	defer c.mu.Unlock();

	item, found := c.items[key];

	if !found {
		return nil, false;
	}

	// Check if the item is expired
	if item.Expiration > 0 &&  item.Expiration < time.Now().UnixNano() {
		delete(c.items, key);
		delete(c.accessCount, key);

		return nil, false;
	}

	// Update last access
	item.LastAccess = time.Now().UnixNano();
	c.items[key] = item;
	c.accessCount[key]++;

	return item.Value, true;
}

// Delete a key from the cache
func (c *Cache) Delete (key string) {
	c.mu.Lock();
	defer c.mu.Unlock();

	delete(c.items, key);
	delete(c.accessCount, key);
}

// Evict an item based on the eviction policy
func (c *Cache) evict() {
	if (c.evictionType == "lru") {
		c.evictLRU();
	} else {
		c.evictLFU();
	}
}

// Evict the least recently used item
func (c *Cache) evictLRU () {
	var oldestKey string;
	var oldestAccess int64 = time.Now().UnixNano();

	for k,v := range c.items {
		if v.LastAccess < oldestAccess {
			oldestAccess = v.LastAccess;
			oldestKey = k;
		}
	}

	delete(c.items, oldestKey);
	delete(c.accessCount, oldestKey);
}

// Evict the least frequently used item
func (c *Cache) evictLFU () {
	var leastUsedKey string;
	var leastUseCount int = -1;

	for k,v := range c.accessCount {
		if leastUseCount == -1 || v < leastUseCount {
			leastUseCount = v;
			leastUsedKey = k;
		}
	}

	delete(c.items, leastUsedKey);
	delete(c.accessCount, leastUsedKey);
}