package cache

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

type PersistenceManager struct {
	cache *Cache;
	filePath string;
	saveInterval time.Duration;
	stopping chan struct{};
	mu sync.Mutex;
}

func NewPersistenceManager (cache *Cache, filePath string, saveInterval time.Duration) *PersistenceManager {
	return &PersistenceManager{
		cache: cache,
		filePath: filePath,
		saveInterval: saveInterval,
		stopping: make(chan struct{}),
	}
}

// Starts the persistence manager
func (pm *PersistenceManager) Start() {
	// Try to load cached data
	pm.loadFromDisk();

	// Start periodic save
	go func () {
		ticker := time.NewTicker(pm.saveInterval);
		defer ticker.Stop();

		for {
			select {
			case <- ticker.C:
				if err := pm.saveToDisk(); err != nil {
					log.Printf("Error saving cache to disk: %v", err);
				}
			case <- pm.stopping:
				// One last save before stopping
				if err := pm.saveToDisk(); err != nil {
					log.Printf("Error saving cache to disk during shutdown: %v", err);
				}
				return;
			}
		}
	}()
}

// Stops the persistence manager
func (pm *PersistenceManager) Stop () {
	pm.mu.Lock();
	defer pm.mu.Unlock();

	close(pm.stopping);
}

// Saves the cache to the disk
func (pm *PersistenceManager) saveToDisk () error {
	pm.mu.Lock();
	defer pm.mu.Unlock();

	// Extract data from cache
	pm.cache.mu.RLock();
	data := make(map[string]interface{});
	now := time.Now().UnixNano();

	for key, item := range pm.cache.items {
		// Skip expired items
		if item.Expiration > 0 && item.Expiration < now {
			continue;
		}

		// Store the item with its metadata
		data[key] = map[string]interface{} {
			"value": item.Value,
			"expiration": item.Expiration,
			"lastAccess":item.LastAccess,
		}
	}
	pm.cache.mu.RUnlock();

	// Create a temporary file
	tempFilePath := pm.filePath + ".tmp";
	file, err := os.Create(tempFilePath);
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err);
	}
	defer file.Close();

	// Write data to file
	encoder := json.NewEncoder(file);
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode cache data: %w", err);
	}

	// Ensure data is written to disk
	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err);
	}

	// Rename temp file to the actual file name (atomic operation)
	if err := os.Rename(tempFilePath, pm.filePath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err);
	}

	log.Printf("Cache successfully saved to %s", pm.filePath);
	return nil;
}

func (pm *PersistenceManager) loadFromDisk () error {
	pm.mu.Lock();
	defer pm.mu.Unlock();

	// Check if the file exists
	if _, err := os.Stat(pm.filePath); os.IsNotExist(err) {
		log.Printf("No cache file found at %s, starting with empty cache", pm.filePath);
		return nil;
	}

	// Open the file
	file, err := os.Open(pm.filePath);
	if err != nil {
		return fmt.Errorf("failed to open ache file: %w", err);
	}
	defer file.Close();

	// Read data from file
	data := make(map[string]map[string]interface{});

	decoder := json.NewDecoder(file);
	if err := decoder.Decode(&data); err != nil {
		if err == io.EOF {
			// Empty file, hence not an error
			return nil;
		}
		return fmt.Errorf("failed to decode cached data: %w", err);
	}

	// Restore the data to the cache
	pm.cache.mu.Lock();
	defer pm.cache.mu.Unlock();

	now := time.Now().UnixNano();
	for key, itemData := range data {
		// Skip expired items
		expiration, ok := itemData["expiration"].(float64);
		if ok && expiration > 0 && int64(expiration) < now {
			continue;
		}

		// Restore
		pm.cache.items[key] = CacheItem{
			Value: itemData["value"],
			Expiration: int64(expiration),
			LastAccess: int64(itemData["lastAccess"].(float64)),
		}

		// Update access count for LFU
		pm.cache.accessCount[key] = 1;
	}

	log.Printf("Cache successfully loaded from %s with %d items", pm.filePath, len(pm.cache.items));
	return nil;
}