package mcache

import (
	"container/list"
	"context"
	"sync"
	"time"
)

type lfuCache struct {
	clock    Clock
	items    map[string]*lfuItem
	freqList *list.List
	cap      int
	sync.RWMutex
}

type freqEntry struct {
	freq  uint
	items map[*lfuItem]struct{}
}

func (c *lfuCache) init(clock Clock) {
	c.clock = clock
	c.items = make(map[string]*lfuItem, defaultShardCap)
	c.freqList = list.New()
	c.cap = defaultShardCap
	c.freqList.PushFront(&freqEntry{
		freq:  0,
		items: make(map[*lfuItem]struct{}, 8),
	})
}

func (c *lfuCache) set(ctx context.Context, key string, val interface{}, ttl time.Duration) error {
	value := vderef(val)
	item, ok := c.items[key]
	if ok {
		item.value = value
	} else {
		c.evict(1)
		item = &lfuItem{
			clock:       c.clock,
			key:         key,
			value:       value,
			freqElement: nil,
		}
		el := c.freqList.Front()
		fe := el.Value.(*freqEntry)
		fe.items[item] = struct{}{}

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
			c.increment(item)
			return item.value, nil
		}
		c.removeItem(item)
	}
	return nil, KeyNotFoundError
}

func (c *lfuCache) evict(count int) {
	if len(c.items) < c.cap {
		return
	}

	entry := c.freqList.Front()
	for i := 0; i < count; {
		if entry == nil {
			return
		} else {
			for item := range entry.Value.(*freqEntry).items {
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

func (c *lfuCache) has(key string) bool {
	item, ok := c.items[key]
	if !ok {
		return false
	}
	return !item.IsExpired()
}

func (c *lfuCache) remove(key string) bool {
	if item, ok := c.items[key]; ok {
		c.removeItem(item)
		return true
	}
	return false
}

func (c *lfuCache) removeItem(item *lfuItem) {
	entry := item.freqElement.Value.(*freqEntry)
	delete(c.items, item.key)
	delete(entry.items, item)
	if isRemovableFreqEntry(entry) {
		c.freqList.Remove(item.freqElement)
	}
	item = nil
}

func (c *lfuCache) increment(item *lfuItem) {
	currentFreqElement := item.freqElement
	currentFreqEntry := currentFreqElement.Value.(*freqEntry)
	nextFreq := currentFreqEntry.freq + 1
	delete(currentFreqEntry.items, item)

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
				items: make(map[*lfuItem]struct{}),
			}, currentFreqElement)
		}
	case nextFreqElement.Value.(*freqEntry).freq == nextFreq:
		if removable {
			c.freqList.Remove(currentFreqElement)
		}
	default:
		panic("unreachable")
	}
	nextFreqElement.Value.(*freqEntry).items[item] = struct{}{}
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
