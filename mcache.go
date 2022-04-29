package mcache

import (
	"context"
	"errors"
	"math"
	"time"
)

type cache struct {
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

func (c *cache) getOption(opts ...Option) *options {
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

type builder[T any, P CachePolicy[T]] struct {
	cache
}

func New[T any, P CachePolicy[T]](size int, opts ...Option) Cache {
	o := options{
		RedisCli: RedisCli{},
	}
	for _, opt := range opts {
		opt(&o)
	}

	b := &builder[T, P]{
		cache{
			size:  size,
			clock: NewRealClock(),
		},
	}
	if b.size == 0 {
		b.size = defaultCacheSize
	}
	b.shardCount = int(math.Ceil(float64(b.size) / float64(defaultShardCap)))
	if b.shardCount == 0 {
		b.shardCount = defaultShardCount
	}
	b.shardCap = b.size / b.shardCount
	if b.shardCap == 0 {
		b.shardCap = defaultShardCap
	}
	b.formatByOpts(o)

	return newCacheHandler(b)
}

func (b *builder[T, P]) formatByOpts(o options) {
	if o.LoaderFunc != nil {
		b.loaderFunc = o.LoaderFunc
	}
	if o.MLoaderFunc != nil {
		b.mLoaderFunc = o.MLoaderFunc
	}
	if o.TTL > 0 {
		b.expiration = o.TTL
	}
	if &o.RedisCli != nil {
		b.redisCli = o.RedisCli
	}
	if o.serializeFunc != nil && o.deserializeFunc != nil {
		b.serializeFunc = o.serializeFunc
		b.deserializeFunc = o.deserializeFunc
	}
	if o.DefaultVal != nil {
		b.defaultVal = o.DefaultVal
	}
}
