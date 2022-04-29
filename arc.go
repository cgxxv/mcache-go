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

func (c *arcCache) init(clock Clock) {
	c.clock = clock
	c.items = make(map[string]arcItem, defaultShardCap)
	c.cap = defaultShardCap

	l := defaultShardCap / 4
	c.t1 = newarcCacheList(l)
	c.t2 = newarcCacheList(l)
	c.b1 = newarcCacheList(l)
	c.b2 = newarcCacheList(l)
}

func (c *arcCache) set(ctx context.Context, key string, val interface{}, ttl time.Duration) error {
	value := vderef(val)
	item, ok := c.items[key]
	if ok {
		item.value = value
	} else {
		item.clock = c.clock
		item.key = key
		item.value = value
		c.items[key] = item
	}
	if ttl > 0 {
		item.expireAt = c.clock.Now().Add(ttl)
	}

	defer func() {
		if c.t1.Has(key) || c.t2.Has(key) {
			return
		}

		if elt := c.b1.Lookup(key); elt != nil {
			c.setPart(minInt(c.cap, c.part+maxInt(c.b2.Len()/c.b1.Len(), 1)))
			c.replace(key)
			c.b1.Remove(key, elt)
			c.t2.PushFront(key)
			return
		}

		if elt := c.b2.Lookup(key); elt != nil {
			c.setPart(maxInt(0, c.part-maxInt(c.b1.Len()/c.b2.Len(), 1)))
			c.replace(key)
			c.b2.Remove(key, elt)
			c.t2.PushFront(key)
			return
		}

		if c.isCacheFull() && c.t1.Len()+c.b1.Len() == c.cap {
			if c.t1.Len() < c.cap {
				c.b1.RemoveTail()
				c.replace(key)
			} else {
				pop := c.t1.RemoveTail()
				delete(c.items, pop)
			}
		} else {
			total := c.t1.Len() + c.b1.Len() + c.t2.Len() + c.b2.Len()
			if total >= c.cap {
				if total == (2 * c.cap) {
					if c.b2.Len() > 0 {
						c.b2.RemoveTail()
					} else {
						c.b1.RemoveTail()
					}
				}
				c.replace(key)
			}
		}
		c.t1.PushFront(key)
	}()
	return nil
}

func (c *arcCache) get(ctx context.Context, key string) (interface{}, error) {
	if elt := c.t1.Lookup(key); elt != nil {
		c.t1.Remove(key, elt)
		item := c.items[key]
		if !item.IsExpired() {
			c.t2.PushFront(key)
			return item.value, nil
		} else {
			delete(c.items, key)
			c.b1.PushFront(key)
		}
	}
	if elt := c.t2.Lookup(key); elt != nil {
		item := c.items[key]
		if !item.IsExpired() {
			c.t2.MoveToFront(elt)
			return item.value, nil
		} else {
			delete(c.items, key)
			c.t2.Remove(key, elt)
			c.b2.PushFront(key)
		}
	}

	return nil, KeyNotFoundError
}

func (c *arcCache) has(ctx context.Context, key string) bool {
	item, ok := c.items[key]
	if !ok {
		return false
	}
	return !item.IsExpired()
}

func (c *arcCache) remove(ctx context.Context, key string) bool {
	if elt := c.t1.Lookup(key); elt != nil {
		c.t1.Remove(key, elt)
		delete(c.items, key)
		c.b1.PushFront(key)
		return true
	}

	if elt := c.t2.Lookup(key); elt != nil {
		c.t2.Remove(key, elt)
		delete(c.items, key)
		c.b2.PushFront(key)
		return true
	}

	return false
}

func (c *arcCache) evict(ctx context.Context, count int) {
	if c.isCacheFull() && c.t1.Len()+c.b1.Len() == c.cap {
		pop := c.t1.RemoveTail()
		delete(c.items, pop)
	}
}

func (c *arcCache) replace(key string) {
	if !c.isCacheFull() {
		return
	}
	var old string
	if c.t1.Len() > 0 && ((c.b2.Has(key) && c.t1.Len() == c.part) || (c.t1.Len() > c.part)) {
		old = c.t1.RemoveTail()
		c.b1.PushFront(old)
	} else if c.t2.Len() > 0 {
		old = c.t2.RemoveTail()
		c.b2.PushFront(old)
	} else {
		old = c.t1.RemoveTail()
		c.b1.PushFront(old)
	}
	delete(c.items, old)
}

func (c *arcCache) setPart(p int) {
	if c.isCacheFull() {
		c.part = p
	}
}

func (c *arcCache) isCacheFull() bool {
	return (c.t1.Len() + c.t2.Len()) == c.cap
}

type arcItem struct {
	clock    Clock
	key      string
	value    interface{}
	expireAt time.Time
}

func (it *arcItem) IsExpired() bool {

	return it.expireAt.Before(it.clock.Now())
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
