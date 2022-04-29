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
	sync.Mutex
}

func (c *lruCache) init(clock Clock, capacity int) {
	c.clock = clock
	c.items = make(map[string]*list.Element, capacity+1)
	c.evictList = list.New()
	c.cap = capacity
}

func (c *lruCache) set(ctx context.Context, key string, val interface{}, ttl time.Duration) error {
	value := deref(val)
	it, ok := c.items[key]
	if ok {
		item := it.Value.(*lfuItem)
		item.value = value
		if ttl > 0 {
			item.expireAt = c.clock.Now().Add(ttl)
		} else {
			item.expireAt = c.clock.Now().Add(defaultExpireAt)
		}
		c.evictList.MoveToFront(it)
	} else {
		c.evict(ctx, 1)
		item := lruItem{
			key:   key,
			value: value,
		}
		if ttl > 0 {
			item.expireAt = c.clock.Now().Add(ttl)
		} else {
			item.expireAt = c.clock.Now().Add(defaultExpireAt)
		}
		c.items[key] = c.evictList.PushFront(&item)
	}

	return nil
}

func (c *lruCache) get(ctx context.Context, key string) (interface{}, error) {
	item, ok := c.items[key]
	if ok {
		it := item.Value.(*lruItem)
		if !it.IsExpired(c.clock) {
			c.evictList.MoveToFront(item)
			return it.value, nil
		}
		c.removeElement(item)
	}
	return nil, KeyNotFoundError
}

func (c *lruCache) evict(ctx context.Context, count int) {
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

func (c *lruCache) has(ctx context.Context, key string) bool {
	item, ok := c.items[key]
	if !ok {
		return false
	}

	if item.Value.(*lruItem).IsExpired(c.clock) {
		c.removeElement(item)
		return false
	}
	return true
}

func (c *lruCache) remove(ctx context.Context, key string) bool {
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
	key      string
	value    interface{}
	expireAt time.Time
}

func (it *lruItem) IsExpired(clock Clock) bool {
	return it.expireAt.Before(clock.Now())
}
