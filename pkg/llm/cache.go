package llm

import (
	"container/list"
	"sync"
	"time"
)

type ResponseCache struct {
	capacity int
	items    map[string]*list.Element
	lru      *list.List
	mutex    sync.RWMutex
	ttl      time.Duration
}

type cacheItem struct {
	key       string
	value     *Response
	timestamp time.Time
}

func NewResponseCache(capacity int, ttl time.Duration) *ResponseCache {
	return &ResponseCache{
		capacity: capacity,
		items:    make(map[string]*list.Element),
		lru:      list.New(),
		ttl:      ttl,
	}
}

func (c *ResponseCache) Get(key string) *Response {
	c.mutex.RLock()
	element, found := c.items[key]
	c.mutex.RUnlock()

	if !found {
		return nil
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if element, found = c.items[key]; !found {
		return nil
	}

	item := element.Value.(*cacheItem)
	if time.Since(item.timestamp) > c.ttl {
		c.lru.Remove(element)
		delete(c.items, key)
		return nil
	}

	c.lru.MoveToFront(element)
	return item.value
}

func (c *ResponseCache) Set(key string, value *Response) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if element, found := c.items[key]; found {
		c.lru.MoveToFront(element)
		item := element.Value.(*cacheItem)
		item.value = value
		item.timestamp = time.Now()
		return
	}

	if c.lru.Len() >= c.capacity {
		oldest := c.lru.Back()
		if oldest != nil {
			item := oldest.Value.(*cacheItem)
			delete(c.items, item.key)
			c.lru.Remove(oldest)
		}
	}

	item := &cacheItem{
		key:       key,
		value:     value,
		timestamp: time.Now(),
	}
	element := c.lru.PushFront(item)
	c.items[key] = element
}

func (c *ResponseCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.items = make(map[string]*list.Element)
	c.lru = list.New()
}
