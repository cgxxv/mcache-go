package mcache

import (
	"container/heap"
	"context"
	"sync"
	"time"
)

type simpleCache struct {
	clock Clock
	items map[string]simpleItem
	pq    simplepq
	cap   int
	sync.Mutex
}

func (c *simpleCache) init(clock Clock) {
	c.clock = clock
	c.items = make(map[string]simpleItem, defaultShardCap)
	c.pq = make(simplepq, 0, defaultShardCap)
	c.cap = defaultShardCap
}

func (c *simpleCache) set(ctx context.Context, key string, val interface{}, ttl time.Duration) error {
	value := vderef(val)
	var entry = &simpleEntry{
		key: key,
	}

	item, ok := c.items[key]
	if ok {
		item.value = value
		entry.priority = item.expireAt.UnixNano()
	} else {
		c.evict(ctx, 1)
		exp := c.clock.Now().Add(defaultExpireAt)
		item.clock = c.clock
		item.value = value
		item.expireAt = exp
		item.index = c.pq.Len()
		c.items[key] = item

		entry.item = &item
		entry.priority = exp.UnixNano()
		heap.Push(&c.pq, entry)
	}

	if ttl > 0 {
		t := c.clock.Now().Add(ttl)
		item.expireAt = t
		entry.priority = t.UnixNano()
		c.pq.update(entry.index)
	}

	return nil
}

func (c *simpleCache) evict(ctx context.Context, count int) {
	if len(c.items) < c.cap {
		return
	}

	now := c.clock.Now()
	if n := c.pq.Len(); n > 0 {
		entry := c.pq[0]
		if now.After(entry.item.expireAt) {
			delete(c.items, entry.key)
			heap.Pop(&c.pq)
			return
		}
	}

	for k, v := range c.items {
		delete(c.items, k)
		heap.Remove(&c.pq, v.index)
		return
	}
}

func (c *simpleCache) get(ctx context.Context, key string) (interface{}, error) {
	item, ok := c.items[key]
	if ok {
		if !item.IsExpired() {
			return item.value, nil
		}
		delete(c.items, key)
	}

	return nil, KeyNotFoundError
}

func (c *simpleCache) has(ctx context.Context, key string) bool {
	item, ok := c.items[key]
	if !ok {
		return false
	}
	return !item.IsExpired()
}

func (c *simpleCache) remove(ctx context.Context, key string) bool {
	item, ok := c.items[key]
	if ok {
		delete(c.items, key)
		heap.Remove(&c.pq, item.index)
		return true
	}
	return false
}

type simpleItem struct {
	clock    Clock
	value    interface{}
	expireAt time.Time
	index    int
}

func (si simpleItem) IsExpired() bool {
	return si.expireAt.Before(si.clock.Now())
}

type simpleEntry struct {
	key      string
	item     *simpleItem
	priority int64
	index    int
}

type simplepq []simpleEntry

func (pq simplepq) Len() int { return len(pq) }

func (pq simplepq) Less(i, j int) bool {
	return pq[i].priority < pq[j].priority
}

func (pq simplepq) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[i].item.index = i
	pq[j].index = j
	pq[j].item.index = j
}

func (pq *simplepq) Push(x interface{}) {
	n := len(*pq)
	entry := x.(simpleEntry)
	entry.index = n
	*pq = append(*pq, entry)
}

func (pq *simplepq) Pop() interface{} {
	old := *pq
	n := len(old)
	entry := old[n-1]
	old[n-1] = simpleEntry{}
	entry.index = -1
	*pq = old[0 : n-1]
	return entry
}

func (pq *simplepq) update(index int) {
	heap.Fix(pq, index)
}
