package mcache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCache(t *testing.T) {
	t.Run("simple cache", runCachePolicy[SimpleCache])
	t.Run("lfu cache", runCachePolicy[LfuCache])
	t.Run("lru cache", runCachePolicy[LruCache])
	t.Run("arc cache", runCachePolicy[ArcCache])
}

func runCachePolicy[T any, P CachePolicy[T]](t *testing.T) {
	var (
		ctx = context.TODO()
		fc  = NewFakeClock()
		cc  = P(new(T))
	)
	cc.init(fc, 5)

	val, err := cc.get(ctx, "key")
	assert.NotNil(t, err)

	cc.set(ctx, "key", "value", 0)
	val, err = cc.get(ctx, "key")
	assert.Equal(t, "value", val)

	cc.set(ctx, "k", "v", 100*time.Millisecond)
	val, _ = cc.get(ctx, "k")
	assert.Equal(t, "v", val)

	fc.Advance(101 * time.Millisecond)
	assert.True(t, cc.has(ctx, "key"))
	assert.False(t, cc.has(ctx, "k"))

	val, err = cc.get(ctx, "k")
	assert.NotNil(t, err)
	assert.Nil(t, val)

	assert.True(t, cc.remove(ctx, "key"))
	assert.False(t, cc.remove(ctx, "k"))

	cc.init(fc, 1)
	assert.Nil(t, cc.set(ctx, "ak", "av", 0))
	cc.evict(ctx, 1)
	assert.False(t, cc.has(ctx, "ak"))
}
