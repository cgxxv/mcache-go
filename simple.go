package mcache

import (
	"container/heap"
	"context"
	"sync"
	"time"
)

type SimpleCache struct {
	clock Clock
	items map[string]simpleItem
	pq    simplepq
	cap   int
	sync.Mutex
}

func (c *SimpleCache) Init(clock Clock, capacity int) {
	c.clock = clock
	c.items = make(map[string]simpleItem, capacity)
	c.pq = make(simplepq, 0, capacity)
	c.cap = capacity
}

func (c *SimpleCache) Set(ctx context.Context, key string, val interface{}, ttl time.Duration) error {
	c.Lock()
	defer c.Unlock()

	value := deref(val)
	var entry = simpleEntry{
		key: key,
	}

	item, ok := c.items[key]
	if ttl > 0 {
		item.expireAt = c.clock.Now().Add(ttl)
	} else {
		item.expireAt = c.clock.Now().Add(defaultExpireAt)
	}

	if ok {
		item.value = value
		entry.priority = item.expireAt.UnixNano()
		c.pq.update(entry.index)
	} else {
		c.evict(ctx, 1)
		item.value = value
		item.index = c.pq.Len()
		c.items[key] = item

		entry.item = &item
		entry.priority = item.expireAt.UnixNano()
		heap.Push(&c.pq, entry)
	}

	return nil
}

func (c *SimpleCache) Get(ctx context.Context, key string) (interface{}, error) {
	c.Lock()
	defer c.Unlock()

	item, ok := c.items[key]
	if ok {
		if !item.IsExpired(c.clock) {
			return item.value, nil
		}
		c.remove(ctx, key)
		return nil, KeyExpiredError
	}

	return nil, KeyNotFoundError
}

func (c *SimpleCache) Exists(ctx context.Context, key string) bool {
	c.Lock()
	defer c.Unlock()

	item, ok := c.items[key]
	if !ok {
		return false
	}

	if item.IsExpired(c.clock) {
		c.remove(ctx, key)
		return false
	}
	return true
}

func (c *SimpleCache) Remove(ctx context.Context, key string) bool {
	c.Lock()
	defer c.Unlock()

	return c.remove(ctx, key)
}

func (c *SimpleCache) Evict(ctx context.Context, count int) {
	c.Lock()
	defer c.Unlock()

	c.evict(ctx, count)
}

func (c *SimpleCache) remove(ctx context.Context, key string) bool {
	item, ok := c.items[key]
	if ok {
		heap.Remove(&c.pq, item.index)
		delete(c.items, key)
		return true
	}
	return false
}

func (c *SimpleCache) evict(ctx context.Context, count int) {
	if len(c.items) < c.cap {
		return
	}

	cnt := 0
	now := c.clock.Now()
	if n := c.pq.Len(); n > 0 {
		entry := c.pq[0]
		item := c.items[entry.key]
		if now.After(item.expireAt) {
			heap.Pop(&c.pq)
			delete(c.items, entry.key)
			cnt++
		}
	}

	for k, v := range c.items {
		if cnt >= count {
			return
		}

		heap.Remove(&c.pq, v.index)
		delete(c.items, k)
		cnt++
	}
}

type simpleItem struct {
	value    interface{}
	expireAt time.Time
	index    int
}

func (si simpleItem) IsExpired(clock Clock) bool {
	return si.expireAt.Before(clock.Now())
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
	entry := old[0]
	entry.index = -1
	*pq = old[1:n]
	return entry
}

func (pq *simplepq) update(index int) {
	heap.Fix(pq, index)
}
