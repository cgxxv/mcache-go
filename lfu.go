package mcache

import (
	"container/list"
	"context"
	"sync"
	"time"
)

type lfuCache struct {
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

func (c *lfuCache) init(clock Clock) {
	c.clock = clock
	c.items = make(map[string]lfuItem, defaultShardCap)
	c.freqList = list.New()
	c.cap = defaultShardCap
	c.freqList.PushFront(&freqEntry{
		freq:  0,
		items: make(map[string]*lfuItem, 8),
	})
}

func (c *lfuCache) set(ctx context.Context, key string, val interface{}, ttl time.Duration) error {
	value := vderef(val)
	item, ok := c.items[key]
	if ok {
		item.value = value
	} else {
		c.evict(ctx, 1)
		item.clock = c.clock
		item.key = key
		item.value = value
		item.freqElement = nil

		el := c.freqList.Front()
		fe := el.Value.(*freqEntry)
		fe.items[key] = &item

		item.freqElement = el
		c.items[key] = item
	}

	if ttl > 0 {
		item.expireAt = c.clock.Now().Add(ttl)
	}

	return nil
}

func (c *lfuCache) get(ctx context.Context, key string) (interface{}, error) {
	item, ok := c.items[key]
	if ok {
		if !item.IsExpired() {
			c.increment(&item)
			return item.value, nil
		}
		c.removeItem(&item)
	}
	return nil, KeyNotFoundError
}

func (c *lfuCache) evict(ctx context.Context, count int) {
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

func (c *lfuCache) has(ctx context.Context, key string) bool {
	item, ok := c.items[key]
	if !ok {
		return false
	}
	return !item.IsExpired()
}

func (c *lfuCache) remove(ctx context.Context, key string) bool {
	if item, ok := c.items[key]; ok {
		c.removeItem(&item)
		return true
	}
	return false
}

func (c *lfuCache) removeItem(item *lfuItem) {
	entry := item.freqElement.Value.(*freqEntry)
	delete(c.items, item.key)
	entry.items[item.key] = nil
	delete(entry.items, item.key)
	if isRemovableFreqEntry(entry) {
		c.freqList.Remove(item.freqElement)
	}
	item = nil
}

func (c *lfuCache) increment(item *lfuItem) {
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
	clock       Clock
	key         string
	value       interface{}
	freqElement *list.Element
	expireAt    time.Time
}

func (it *lfuItem) IsExpired() bool {
	return it.expireAt.Before(it.clock.Now())
}
