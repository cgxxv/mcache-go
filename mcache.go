package mcache

import (
	"context"
	"errors"
	"math"
	"time"
)

type baseCache struct {
	typ         string
	name        string
	clock       Clock
	size        int
	shardCount  int
	shardCap    int
	defaultVal  interface{}
	expiration  time.Duration
	loaderFunc  LoaderFunc
	mLoaderFunc MLoaderFunc
	redisCli    RedisCli

	serializeFunc   serializeFunc
	deserializeFunc deserializeFunc
}

func (c *baseCache) getOption(opts ...Option) *options {
	o := &options{
		IsWait:          true,
		TTL:             c.expiration,
		LoaderFunc:      c.loaderFunc,
		MLoaderFunc:     c.mLoaderFunc,
		DefaultVal:      c.defaultVal,
		serializeFunc:   c.serializeFunc,
		deserializeFunc: c.deserializeFunc,
	}

	for _, opt := range opts {
		opt(o)
	}

	if o.LoaderFunc == nil {
		if c.redisCli.Client != nil {
			o.RealLoaderFunc = func(ctx context.Context, k string) (interface{}, error) {
				v, err := c.redisCli.get(ctx, k, o)
				if err == nil {
					return v, nil
				}
				return nil, err
			}
		}
	} else {
		o.RealLoaderFunc = func(ctx context.Context, k string) (interface{}, error) {
			if v, err := c.redisCli.get(ctx, k, o); err == nil {
				return v, nil
			}

			v, err := o.LoaderFunc(ctx, k)
			if err == nil {
				if err := c.redisCli.set(ctx, k, v, o); err != nil && !errors.Is(err, RedisNotFoundError) {
					return nil, err
				}
				return v, nil
			} else if o.DefaultVal != nil {
				return o.DefaultVal, DefValSetError
			}
			return nil, err
		}
	}

	if o.MLoaderFunc == nil {
		if c.redisCli.Client != nil {
			o.RealMLoaderFunc = func(ctx context.Context, keys []string) (map[string]interface{}, error) {
				result, err := c.redisCli.mget(ctx, keys, o)
				return result, err
			}
		}
	} else {
		o.RealMLoaderFunc = func(ctx context.Context, keys []string) (map[string]interface{}, error) {
			result, _ := c.redisCli.mget(ctx, keys, o)
			keysG := make([]string, 0, len(keys))
			for _, key := range keys {
				if _, ok := result[key]; !ok {
					keysG = append(keysG, key)
				}
			}

			val, err := o.MLoaderFunc(ctx, keysG)
			if err == nil {
				if err := c.redisCli.msetmap(ctx, val, o); err != nil && !errors.Is(err, RedisNotFoundError) {
					return nil, err
				}

				for k, v := range val {
					result[k] = v
				}
			}
			return result, nil
		}
	}

	return o
}

type CacheBuilder struct {
	baseCache
}

func New(size int, opts ...Option) Cache {
	o := options{
		Typ:      typeSimple,
		RedisCli: RedisCli{},
	}
	for _, opt := range opts {
		opt(&o)
	}
	if !o.check() {
		panic("mcache: build option check fail")
	}

	cb := &CacheBuilder{
		baseCache{
			size:  size,
			clock: NewRealClock(),
		},
	}
	if cb.size == 0 {
		cb.size = defaultCacheSize
	}
	cb.shardCount = int(math.Ceil(float64(cb.size) / float64(defaultShardCap)))
	if cb.shardCount == 0 {
		cb.shardCount = defaultShardCount
	}
	cb.shardCap = cb.size / cb.shardCount
	if cb.shardCap == 0 {
		cb.shardCap = defaultShardCap
	}
	cb.formatByOpts(o)

	return cb.build()
}

func NewArc(size int) Cache {
	return New(size, WithCacheType(typeArc))
}

func NewLRU(size int) Cache {
	return New(size, WithCacheType(typeLru))
}

func NewLFU(size int) Cache {
	return New(size, WithCacheType(typeLfu))
}

func (cb *CacheBuilder) formatByOpts(o options) {
	cb.typ = o.Typ
	cb.name = o.Name
	if o.LoaderFunc != nil {
		cb.loaderFunc = o.LoaderFunc
	}
	if o.MLoaderFunc != nil {
		cb.mLoaderFunc = o.MLoaderFunc
	}
	if o.TTL > 0 {
		cb.expiration = o.TTL
	}
	if &o.RedisCli != nil {
		cb.redisCli = o.RedisCli
	}
	if o.serializeFunc != nil && o.deserializeFunc != nil {
		cb.serializeFunc = o.serializeFunc
		cb.deserializeFunc = o.deserializeFunc
	}
	if o.DefaultVal != nil {
		cb.defaultVal = o.DefaultVal
	}
}

func (cb *CacheBuilder) Clock(clock Clock) *CacheBuilder {
	cb.clock = clock
	return cb
}

func (cb *CacheBuilder) build() Cache {
	if cb.name == "" {
		cb.name = cb.typ
	}

	return newShard(cb)
}
