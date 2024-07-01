package mcache

import (
	"context"
	"errors"
)

var (
	KeyNotFoundError     = errors.New("mcache: key not found.")
	KeyExpiredError      = errors.New("mcache: key expired.")
	RedisNotFoundError   = errors.New("mcache: redis not found.")
	SerializeError       = errors.New("mcache: must set WithSafeValPtrFunc option!")
	KeyValueLenError     = errors.New("mcache: len of key != len of value.")
	DefaultValueSetError = errors.New("mcache: set def val, 1min expiration.")
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
	debugLocalGet(ctx context.Context, key string) (interface{}, error)
	debugLocalRemove(ctx context.Context, key string) bool
	serializer
}

type (
	LoaderFunc  func(context.Context, string) (interface{}, error)
	MLoaderFunc func(context.Context, []string) (map[string]interface{}, error)

	valuePtrFunc func() interface{}
)

const (
	defaultCacheSize  = 1 << 7                  //默认缓存容量
	defaultShardCap   = 1 << 6                  //默认单片容量
	defaultShardCount = 1 << 5                  //默认分片数量
	defaultExpiredAt  = 100 * 365 * 24 * 3600e9 //100years
)
