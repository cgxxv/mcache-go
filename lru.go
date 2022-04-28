package mcache

import (
	"container/list"
	"context"
	"sync"
	"time"
)

type lruCache struct {
	clock     Clock
	items     map[string]*list.Element
	evictList *list.List
	cap       int
	sync.RWMutex
}

func (c *lruCache) init(clock Clock) {
	c.clock = clock
	c.items = make(map[string]*list.Element, defaultShardCap+1)
	c.evictList = list.New()
	c.cap = defaultShardCap
}

func (c *lruCache) set(ctx context.Context, key string, val interface{}, ttl time.Duration) error {
	value := vderef(val)
	var item *lruItem
	if it, ok := c.items[key]; ok {
		c.evictList.MoveToFront(it)
		item = it.Value.(*lruItem)
		item.value = value
	} else {
		c.evict(1)
		item = &lruItem{
			clock: c.clock,
			key:   key,
			value: value,
		}
		c.items[key] = c.evictList.PushFront(item)
	}

	if ttl > 0 {
		item.expireAt = c.clock.Now().Add(ttl)
	}

	return nil
}

func (c *lruCache) get(ctx context.Context, key string) (interface{}, error) {
	item, ok := c.items[key]
	if ok {
		it := item.Value.(*lruItem)
		if !it.IsExpired() {
			c.evictList.MoveToFront(item)
			return it.value, nil
		}
		c.removeElement(item)
	}
	return nil, KeyNotFoundError
}

func (c *lruCache) evict(count int) {
	if c.evictList.Len() < c.cap {
		return
	}

	for i := 0; i < count; i++ {
		ent := c.evictList.Back()
		if ent == nil {
			return
		} else {
			c.removeElement(ent)
		}
	}
}

func (c *lruCache) has(key string) bool {
	item, ok := c.items[key]
	if !ok {
		return false
	}
	return !item.Value.(*lruItem).IsExpired()
}

func (c *lruCache) remove(key string) bool {
	if ent, ok := c.items[key]; ok {
		c.removeElement(ent)
		return true
	}
	return false
}

func (c *lruCache) removeElement(e *list.Element) {
	c.evictList.Remove(e)
	entry := e.Value.(*lruItem)
	delete(c.items, entry.key)
}

type lruItem struct {
	clock    Clock
	key      string
	value    interface{}
	expireAt time.Time
}

func (it *lruItem) IsExpired() bool {
	return it.expireAt.Before(it.clock.Now())
}
