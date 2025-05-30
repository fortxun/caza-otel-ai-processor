package llm

import (
	"container/list"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/fortxun/idop/pkg/types/models"
)

type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, ttl time.Duration)
	Delete(key string)
}

type cacheItem struct {
	key       string
	value     interface{}
	expiresAt time.Time
	element   *list.Element
}

type LRUCache struct {
	maxSize int
	items   map[string]*cacheItem
	lru     *list.List
	mutex   sync.RWMutex
}

func NewLRUCache(maxSize int, defaultTTL time.Duration) *LRUCache {
	if maxSize <= 0 {
		maxSize = 100 // Default size
	}
	
	return &LRUCache{
		maxSize: maxSize,
		items:   make(map[string]*cacheItem, maxSize),
		lru:     list.New(),
	}
}

func (c *LRUCache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	item, found := c.items[key]
	c.mutex.RUnlock()
	
	if !found {
		return nil, false
	}
	
	if time.Now().After(item.expiresAt) {
		c.mutex.Lock()
		c.delete(key)
		c.mutex.Unlock()
		return nil, false
	}
	
	c.mutex.Lock()
	c.lru.MoveToFront(item.element)
	c.mutex.Unlock()
	
	return item.value, true
}

func (c *LRUCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	if item, found := c.items[key]; found {
		item.value = value
		item.expiresAt = time.Now().Add(ttl)
		c.lru.MoveToFront(item.element)
		return
	}
	
	if len(c.items) >= c.maxSize {
		oldest := c.lru.Back()
		if oldest != nil {
			item := oldest.Value.(*cacheItem)
			c.delete(item.key)
		}
	}
	
	item := &cacheItem{
		key:       key,
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	
	item.element = c.lru.PushFront(item)
	c.items[key] = item
}

func (c *LRUCache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.delete(key)
}

func (c *LRUCache) delete(key string) {
	if item, found := c.items[key]; found {
		c.lru.Remove(item.element)
		delete(c.items, key)
	}
}

func generateHashKey(prefix string, anomalies []models.Anomaly, contextData ContextData) string {
	data := struct {
		Prefix       string
		Anomalies    []models.Anomaly
		RelatedMetrics []models.Metric
		TimeRange    TimeRange
	}{
		Prefix:       prefix,
		Anomalies:    anomalies,
		RelatedMetrics: contextData.RelatedMetrics,
		TimeRange:    contextData.TimeRange,
	}
	
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Sprintf("%s-%d-%d", prefix, len(anomalies), time.Now().Unix())
	}
	
	hash := md5.Sum(jsonData)
	return prefix + "-" + hex.EncodeToString(hash[:])
}
