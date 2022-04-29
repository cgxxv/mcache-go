package mcache

import (
	"container/list"
	"context"
	"sync"
	"time"
)

type arcCache struct {
	clock Clock
	items map[string]arcItem
	cap   int
	sync.Mutex

	part int
	t1   *arcList
	t2   *arcList
	b1   *arcList
	b2   *arcList
}

func (c *arcCache) init(clock Clock, capacity int) {
	c.clock = clock
	c.items = make(map[string]arcItem, capacity)
	c.cap = capacity

	l := capacity / 2
	c.t1 = newarcCacheList(l)
	c.t2 = newarcCacheList(c.cap - l)
	c.b1 = newarcCacheList(c.cap - l)
	c.b2 = newarcCacheList(l)
}

func (c *arcCache) set(ctx context.Context, key string, val interface{}, ttl time.Duration) error {
	value := deref(val)
	item, ok := c.items[key]
	if ttl > 0 {
		item.expireAt = c.clock.Now().Add(ttl)
	} else {
		item.expireAt = c.clock.Now().Add(defaultExpireAt)
	}

	if ok {
		item.value = value
	} else {
		item.key = key
		item.value = value
		c.items[key] = item
	}

	defer func() {
		c.evict(ctx, 1)
		if c.t1.Has(key) || c.t2.Has(key) {
			return
		}

		if ok {
			c.update(ctx, key)
		} else {
			c.t1.PushFront(key)
		}
	}()
	return nil
}

func (c *arcCache) get(ctx context.Context, key string) (interface{}, error) {
	item, ok := c.items[key]
	if !ok {
		return nil, KeyNotFoundError
	}

	c.update(ctx, key)
	if item.IsExpired(c.clock) {
		c.remove(ctx, key)
		return nil, KeyExpiredError
	}

	return item.value, nil
}

func (c *arcCache) has(ctx context.Context, key string) bool {
	item, ok := c.items[key]
	if !ok {
		return false
	}

	c.update(ctx, key)
	if item.IsExpired(c.clock) {
		c.remove(ctx, key)
		return false
	}
	return true
}

func (c *arcCache) remove(ctx context.Context, key string) bool {
	delete(c.items, key)
	if elt := c.b1.Lookup(key); elt != nil {
		c.b1.Remove(key, elt)
		return true
	}
	if elt := c.t1.Lookup(key); elt != nil {
		c.t1.Remove(key, elt)
		return true
	}
	if elt := c.b2.Lookup(key); elt != nil {
		c.b2.Remove(key, elt)
		return true
	}
	if elt := c.t2.Lookup(key); elt != nil {
		c.t2.Remove(key, elt)
		return true
	}

	return false
}

func (c *arcCache) evict(ctx context.Context, count int) {
	if !c.isCacheFull() {
		return
	}

	cnt := 0
	for {
		if cnt >= count {
			break
		}

		if c.isCacheFull() && c.t1.Len()+c.b1.Len() == c.cap {
			if c.t1.Len() < c.cap {
				if c.b1.Len() > 0 {
					pop := c.b1.RemoveTail()
					delete(c.items, pop)
					cnt++
				}
			} else {
				pop := c.t1.RemoveTail()
				delete(c.items, pop)
				cnt++
			}
		} else {
			total := c.t1.Len() + c.b1.Len() + c.t2.Len() + c.b2.Len()
			if total == c.cap<<1 {
				if c.b2.Len() > 0 {
					pop := c.b2.RemoveTail()
					delete(c.items, pop)
					cnt++
					continue
				}
				if c.b1.Len() > 0 {
					pop := c.b1.RemoveTail()
					delete(c.items, pop)
					cnt++
				}
			}
		}
	}
}

func (c *arcCache) update(ctx context.Context, key string) {
	if e := c.b1.Lookup(key); e != nil {
		c.b1.Remove(key, e)
		c.t1.PushFront(key)
		return
	}

	if e := c.t1.Lookup(key); e != nil {
		c.t1.Remove(key, e)
		c.t2.PushFront(key)
		return
	}

	if e := c.t2.Lookup(key); e != nil {
		c.t2.MoveToFront(e)
		return
	}

	if e := c.b2.Lookup(key); e != nil {
		c.b2.Remove(key, e)
		c.t1.PushFront(key)
		if pop := c.t1.RemoveTail(); pop != "" {
			c.b1.PushFront(pop)
		}
	}
}

func (c *arcCache) isCacheFull() bool {
	return (c.t1.Len() + c.t2.Len()) == c.cap
}

type arcItem struct {
	key      string
	value    interface{}
	expireAt time.Time
}

func (it *arcItem) IsExpired(clock Clock) bool {
	return it.expireAt.Before(clock.Now())
}

type arcList struct {
	l    *list.List
	keys map[string]*list.Element
}

func newarcCacheList(cap int) *arcList {
	return &arcList{
		l:    list.New(),
		keys: make(map[string]*list.Element, cap),
	}
}

func (al *arcList) Has(key string) bool {
	_, ok := al.keys[key]
	return ok
}

func (al *arcList) Lookup(key string) *list.Element {
	elt := al.keys[key]
	return elt
}

func (al *arcList) MoveToFront(elt *list.Element) {
	al.l.MoveToFront(elt)
}

func (al *arcList) PushFront(key string) {
	if elt, ok := al.keys[key]; ok {
		al.l.MoveToFront(elt)
		return
	}
	elt := al.l.PushFront(key)
	al.keys[key] = elt
}

func (al *arcList) Remove(key string, elt *list.Element) {
	delete(al.keys, key)
	al.l.Remove(elt)
}

func (al *arcList) RemoveTail() string {
	elt := al.l.Back()
	al.l.Remove(elt)
	key := elt.Value.(string)
	delete(al.keys, key)
	return key
}

func (al *arcList) Len() int {
	return al.l.Len()
}
