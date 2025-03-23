package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2"
)

// ModelResultsCache caches model inference results
type ModelResultsCache struct {
	cache       *lru.Cache[string, cacheEntry]
	mutex       sync.RWMutex
	maxSize     int
	ttlSeconds  int
	hitCount    int64
	missCount   int64
	enabled     bool
}

// Cache entry with result and expiration time
type cacheEntry struct {
	result    map[string]interface{}
	expiresAt time.Time
}

// NewModelResultsCache creates a new cache for model results
func NewModelResultsCache(maxSize int, ttlSeconds int) (*ModelResultsCache, error) {
	if maxSize <= 0 {
		// Return a disabled cache
		return &ModelResultsCache{
			enabled: false,
		}, nil
	}

	cache, err := lru.New[string, cacheEntry](maxSize)
	if err != nil {
		return nil, err
	}

	return &ModelResultsCache{
		cache:      cache,
		maxSize:    maxSize,
		ttlSeconds: ttlSeconds,
		enabled:    true,
	}, nil
}

// Get retrieves a result from the cache
func (c *ModelResultsCache) Get(input map[string]interface{}) (map[string]interface{}, bool) {
	if !c.enabled {
		return nil, false
	}

	// Create a key from the input
	key, err := c.createKey(input)
	if err != nil {
		return nil, false
	}

	c.mutex.RLock()
	entry, found := c.cache.Get(key)
	c.mutex.RUnlock()

	if !found {
		c.missCount++
		return nil, false
	}

	// Check if the entry has expired
	if time.Now().After(entry.expiresAt) {
		c.mutex.Lock()
		c.cache.Remove(key)
		c.mutex.Unlock()
		c.missCount++
		return nil, false
	}

	c.hitCount++
	return entry.result, true
}

// Put adds a result to the cache
func (c *ModelResultsCache) Put(input map[string]interface{}, result map[string]interface{}) error {
	if !c.enabled {
		return nil
	}

	// Create a key from the input
	key, err := c.createKey(input)
	if err != nil {
		return err
	}

	// Create a deep copy of the result to avoid modifying the cached value
	resultCopy := make(map[string]interface{})
	for k, v := range result {
		resultCopy[k] = v
	}

	entry := cacheEntry{
		result:    resultCopy,
		expiresAt: time.Now().Add(time.Duration(c.ttlSeconds) * time.Second),
	}

	c.mutex.Lock()
	c.cache.Add(key, entry)
	c.mutex.Unlock()

	return nil
}

// GetStats returns cache statistics
func (c *ModelResultsCache) GetStats() map[string]interface{} {
	if !c.enabled {
		return map[string]interface{}{
			"enabled": false,
		}
	}

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return map[string]interface{}{
		"enabled":     true,
		"size":        c.cache.Len(),
		"max_size":    c.maxSize,
		"ttl_seconds": c.ttlSeconds,
		"hit_count":   c.hitCount,
		"miss_count":  c.missCount,
		"hit_ratio":   float64(c.hitCount) / float64(c.hitCount+c.missCount),
	}
}

// Clear clears the cache
func (c *ModelResultsCache) Clear() {
	if !c.enabled {
		return
	}

	c.mutex.Lock()
	c.cache.Purge()
	c.hitCount = 0
	c.missCount = 0
	c.mutex.Unlock()
}

// createKey creates a cache key from the input
func (c *ModelResultsCache) createKey(input map[string]interface{}) (string, error) {
	// Serialize the input to JSON
	bytes, err := json.Marshal(input)
	if err != nil {
		return "", err
	}

	// Create a hash of the serialized input
	hash := sha256.Sum256(bytes)
	return hex.EncodeToString(hash[:]), nil
}

// ResourceCache caches processed resources
type ResourceCache struct {
	cache       *lru.Cache[string, interface{}]
	mutex       sync.RWMutex
	maxSize     int
	enabled     bool
}

// NewResourceCache creates a new cache for resources
func NewResourceCache(maxSize int) (*ResourceCache, error) {
	if maxSize <= 0 {
		// Return a disabled cache
		return &ResourceCache{
			enabled: false,
		}, nil
	}

	cache, err := lru.New[string, interface{}](maxSize)
	if err != nil {
		return nil, err
	}

	return &ResourceCache{
		cache:       cache,
		maxSize:     maxSize,
		enabled:     true,
	}, nil
}