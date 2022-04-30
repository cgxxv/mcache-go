package mcache

import (
	"context"
	"errors"
)

var (
	KeyNotFoundError   = errors.New("mcache: key not found.")
	KeyExpiredError    = errors.New("mcache: key expired.")
	RedisNotFoundError = errors.New("mcache: redis not found.")
	SerializeError     = errors.New("mcache: serialize error.")
	KeyValueLenError   = errors.New("mcache: len of key != len of value.")
	DefValSetError     = errors.New("mcache: set def val, 1min expiration.")
)

type Cache interface {
	Set(ctx context.Context, key string, value interface{}, opts ...Option) error
	MSet(ctx context.Context, keys []string, values []interface{}, opts ...Option) error

	Get(ctx context.Context, key string, opts ...Option) (interface{}, error)
	MGet(ctx context.Context, keys []string, opts ...Option) (map[string]interface{}, error)

	Remove(ctx context.Context, key string) bool
	MRemove(ctx context.Context, keys []string) bool
	Exists(ctx context.Context, key string) bool

	//only for debug
	DebugShardIndex(key string) uint64
	debugFromLocal2(ctx context.Context, key string, onLoad bool) (interface{}, error)
	debugFromLocal(ctx context.Context, key string, onLoad bool) (interface{}, error)
	debugRemove(ctx context.Context, key string) bool
	serializer
}

type (
	LoaderFunc  func(context.Context, string) (interface{}, error)
	MLoaderFunc func(context.Context, []string) (map[string]interface{}, error)

	valPtrFunc func() interface{}
)

const (
	defaultCacheSize  = 1 << 7                  //默认缓存容量
	defaultShardCap   = 1 << 6                  //默认单片容量
	defaultShardCount = 1 << 5                  //默认分片数量
	defaultExpireAt   = 100 * 365 * 24 * 3600e9 //100years
)
