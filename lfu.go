package mcache

import (
	"container/list"
	"context"
	"sync"
	"time"
)

type LfuCache struct {
	clock    Clock
	items    map[string]lfuItem
	freqList *list.List
	cap      int
	sync.Mutex
}

type freqEntry struct {
	freq  uint
	items map[string]*lfuItem
}

func (c *LfuCache) init(clock Clock, capacity int) {
	c.clock = clock
	c.items = make(map[string]lfuItem, capacity)
	c.freqList = list.New()
	c.cap = capacity
	c.freqList.PushFront(&freqEntry{
		freq:  0,
		items: make(map[string]*lfuItem, 8),
	})
}

func (c *LfuCache) set(ctx context.Context, key string, val interface{}, ttl time.Duration) error {
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
		c.evict(ctx, 1)
		item.key = key
		item.value = value
		item.freqElement = nil

		el := c.freqList.Front()
		fe := el.Value.(*freqEntry)
		fe.items[key] = &item

		item.freqElement = el
		c.items[key] = item
	}

	return nil
}

func (c *LfuCache) get(ctx context.Context, key string) (interface{}, error) {
	item, ok := c.items[key]
	if ok {
		if !item.IsExpired(c.clock) {
			c.increment(&item)
			return item.value, nil
		}
		c.removeItem(&item)
		return nil, KeyExpiredError
	}
	return nil, KeyNotFoundError
}

func (c *LfuCache) evict(ctx context.Context, count int) {
	if len(c.items) < c.cap {
		return
	}

	entry := c.freqList.Front()
	for i := 0; i < count; {
		if entry == nil {
			return
		} else {
			for _, item := range entry.Value.(*freqEntry).items {
				if i >= count {
					return
				}
				c.removeItem(item)
				i++
			}
			entry = entry.Next()
		}
	}
}

func (c *LfuCache) has(ctx context.Context, key string) bool {
	item, ok := c.items[key]
	if !ok {
		return false
	}

	if item.IsExpired(c.clock) {
		c.removeItem(&item)
		return false
	}

	return true
}

func (c *LfuCache) remove(ctx context.Context, key string) bool {
	item, ok := c.items[key]
	if ok {
		c.removeItem(&item)
		return true
	}
	return false
}

func (c *LfuCache) removeItem(item *lfuItem) {
	entry := item.freqElement.Value.(*freqEntry)
	delete(c.items, item.key)
	entry.items[item.key] = nil
	delete(entry.items, item.key)
	if isRemovableFreqEntry(entry) {
		c.freqList.Remove(item.freqElement)
	}
	item = nil
}

func (c *LfuCache) increment(item *lfuItem) {
	currentFreqElement := item.freqElement
	currentFreqEntry := currentFreqElement.Value.(*freqEntry)
	nextFreq := currentFreqEntry.freq + 1
	delete(currentFreqEntry.items, item.key)

	removable := isRemovableFreqEntry(currentFreqEntry)

	nextFreqElement := currentFreqElement.Next()
	switch {
	case nextFreqElement == nil || nextFreqElement.Value.(*freqEntry).freq > nextFreq:
		if removable {
			currentFreqEntry.freq = nextFreq
			nextFreqElement = currentFreqElement
		} else {
			nextFreqElement = c.freqList.InsertAfter(&freqEntry{
				freq:  nextFreq,
				items: make(map[string]*lfuItem),
			}, currentFreqElement)
		}
	case nextFreqElement.Value.(*freqEntry).freq == nextFreq:
		if removable {
			c.freqList.Remove(currentFreqElement)
		}
	default:
		panic("unreachable")
	}
	nextFreqElement.Value.(*freqEntry).items[item.key] = item
	item.freqElement = nextFreqElement
}

func isRemovableFreqEntry(entry *freqEntry) bool {
	return entry.freq != 0 && len(entry.items) == 0
}

type lfuItem struct {
	key         string
	value       interface{}
	freqElement *list.Element
	expireAt    time.Time
}

func (it *lfuItem) IsExpired(clock Clock) bool {
	return it.expireAt.Before(clock.Now())
}
