// Package translator defines the Translator interface and provides an
// LRU-cache layer that wraps any concrete implementation.
package translator

import (
	"container/list"
	"context"
	"sync"
)

// Translator is the common interface for all translation backends.
type Translator interface {
	// Translate converts text from sourceLang to targetLang.
	// Language codes: "TR", "EN", "EN-US" (DeepL style accepted by both backends).
	Translate(ctx context.Context, text, sourceLang, targetLang string) (string, error)
}

// cacheKey uniquely identifies a translation request.
type cacheKey struct {
	text, src, dst string
}

// cacheEntry is stored in the LRU list.
type cacheEntry struct {
	key   cacheKey
	value string
}

// CachedTranslator wraps a Translator with an in-memory LRU cache.
type CachedTranslator struct {
	inner    Translator
	mu       sync.Mutex
	capacity int
	list     *list.List
	items    map[cacheKey]*list.Element
}

// NewCachedTranslator creates a CachedTranslator. If capacity <= 0 caching is disabled.
func NewCachedTranslator(inner Translator, capacity int) *CachedTranslator {
	return &CachedTranslator{
		inner:    inner,
		capacity: capacity,
		list:     list.New(),
		items:    make(map[cacheKey]*list.Element, capacity),
	}
}

// Translate checks the cache first; on miss it delegates to the inner Translator.
func (c *CachedTranslator) Translate(ctx context.Context, text, src, dst string) (string, error) {
	if c.capacity <= 0 {
		return c.inner.Translate(ctx, text, src, dst)
	}

	key := cacheKey{text, src, dst}

	c.mu.Lock()
	if el, ok := c.items[key]; ok {
		c.list.MoveToFront(el)
		result := el.Value.(*cacheEntry).value
		c.mu.Unlock()
		return result, nil
	}
	c.mu.Unlock()

	// Cache miss — call backend (without holding the lock to avoid blocking).
	result, err := c.inner.Translate(ctx, text, src, dst)
	if err != nil {
		return "", err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	// Evict if at capacity.
	if c.list.Len() >= c.capacity {
		oldest := c.list.Back()
		if oldest != nil {
			c.list.Remove(oldest)
			delete(c.items, oldest.Value.(*cacheEntry).key)
		}
	}
	el := c.list.PushFront(&cacheEntry{key, result})
	c.items[key] = el
	return result, nil
}
