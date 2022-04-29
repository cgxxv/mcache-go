package mcache

import (
	"testing"

	"github.com/alicebob/miniredis"
	"github.com/go-redis/redis/v8"
)

var (
	redisServer *miniredis.Miniredis
	redisClient *redis.Client
)

func init() {
	fakeRedis, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	redisServer = fakeRedis
	redisClient = redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
}

func TestSimpleMcacheNoRemote(t *testing.T) {
}
