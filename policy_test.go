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
	cc.Init(fc, 5)

	val, err := cc.Get(ctx, "key")
	assert.NotNil(t, err)

	cc.Set(ctx, "key", "value", 0)
	val, err = cc.Get(ctx, "key")
	assert.Equal(t, "value", val)

	cc.Set(ctx, "k", "v", 100*time.Millisecond)
	val, _ = cc.Get(ctx, "k")
	assert.Equal(t, "v", val)

	fc.Advance(101 * time.Millisecond)
	assert.True(t, cc.Exists(ctx, "key"))
	assert.False(t, cc.Exists(ctx, "k"))

	val, err = cc.Get(ctx, "k")
	assert.NotNil(t, err)
	assert.Nil(t, val)

	assert.True(t, cc.Remove(ctx, "key"))
	assert.False(t, cc.Remove(ctx, "k"))

	cc.Init(fc, 1)
	assert.Nil(t, cc.Set(ctx, "ak", "av", 0))
	cc.Evict(ctx, 1)
	assert.False(t, cc.Exists(ctx, "ak"))
}
