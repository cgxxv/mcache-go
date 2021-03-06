package mcache_test

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis"
	"github.com/cgxxv/mcache-go/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
)

const (
	runTimes = 1 << 5
)

var (
	redisServer *miniredis.Miniredis
	redisClient *redis.Client
	keys        []string
	vals        []string
)

func init() {
	fakeRedis, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	redisServer = fakeRedis
	redisClient = redis.NewClient(&redis.Options{Addr: redisServer.Addr()})

	for i := 0; i < runTimes; i++ {
		keys = append(keys, mcache.RandString(i+10))
		vals = append(vals, mcache.RandString(i+11))
	}
}

func TestMcacheNoRemote(t *testing.T) {
	t.Run("simple cache", runMcacheNoRemote[mcache.SimpleCache])
	t.Run("lfu cache", runMcacheNoRemote[mcache.LfuCache])
	t.Run("lru cache", runMcacheNoRemote[mcache.LruCache])
	t.Run("arc cache", runMcacheNoRemote[mcache.ArcCache])
}

func runMcacheNoRemote[T any, P mcache.CachePolicy[T]](t *testing.T) {
	var (
		ctx = context.TODO()
		cc  = mcache.New[T, P](runTimes)
	)

	for i := 0; i < runTimes; i++ {
		err := cc.Set(ctx, keys[i], vals[i])
		assert.Nil(t, err)
	}

	for i := 0; i < runTimes; i++ {
		assert.True(t, cc.Exists(ctx, keys[i]))
		val, err := cc.Get(ctx, keys[i])
		assert.Nil(t, err)
		assert.Equal(t, vals[i], val)
	}
}

func TestMcacheRemote(t *testing.T) {
	t.Run("simple cache", runMcacheRemote[mcache.SimpleCache])
	t.Run("lfu cache", runMcacheRemote[mcache.LfuCache])
	t.Run("lru cache", runMcacheRemote[mcache.LruCache])
	t.Run("arc cache", runMcacheNoRemote[mcache.ArcCache])
}

func runMcacheRemote[T any, P mcache.CachePolicy[T]](t *testing.T) {
	var (
		ctx = context.TODO()
		cc  = mcache.New[T, P](runTimes, mcache.WithRedisClient(redisClient))
		val interface{}
		err error
	)

	for i := 0; i < runTimes; i++ {
		err = cc.Set(ctx, keys[i], vals[i])
		assert.Nil(t, err)
	}

	for i := 0; i < runTimes; i++ {
		val = redisClient.Get(ctx, keys[i]).Val()
		assert.Equal(t, vals[i], val)

		assert.True(t, cc.Exists(ctx, keys[i]))
		val, err = cc.Get(ctx, keys[i])
		assert.Nil(t, err)
		assert.Equal(t, vals[i], val)
	}
}
