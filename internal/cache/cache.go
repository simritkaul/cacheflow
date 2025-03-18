package cache

import (
	"sync"
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