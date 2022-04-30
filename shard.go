package mcache

import (
	"context"
	"errors"
	"sync"
	"time"
)

type CachePolicy[T any] interface {
	Init(clock Clock, capacity int)
	Set(ctx context.Context, key string, val interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string) (interface{}, error)
	Exists(ctx context.Context, key string) bool
	Remove(ctx context.Context, key string) bool
	Evict(ctx context.Context, count int)

	sync.Locker
	*T
}

type cacheHandler[T any, P CachePolicy[T]] struct {
	cache

	shards []P
}

func newCacheHandler[T any, P CachePolicy[T]](b *builder[T, P]) Cache {
	c := &cacheHandler[T, P]{}
	c.cache = b.cache

	c.shards = make([]P, c.shardCount)
	for i := 0; i < c.shardCount; i++ {
		var p = P(new(T))
		p.Init(c.clock, c.shardCap)
		c.shards[i] = p
	}

	return c
}

func (c *cacheHandler[T, P]) Set(ctx context.Context, key string, value interface{}, opts ...Option) error {
	o := c.getOption(opts...)

	if c.redisCli.Client != nil {
		err := c.redisCli.set(ctx, key, value, o)
		if err != nil {
			return err
		}
	}

	s := c.getShard(key)
	s.Lock()
	defer s.Unlock()

	return s.Set(ctx, key, value, o.TTL)
}

func (c *cacheHandler[T, P]) MSet(ctx context.Context, keys []string, values []interface{}, opts ...Option) error {
	if len(keys) != len(values) {
		return KeyValueLenError
	}

	o := c.getOption(opts...)

	if c.redisCli.Client != nil {
		err := c.redisCli.mset(ctx, keys, values, o)
		if err != nil {
			return err
		}
	}

	for i, k := range keys {
		s := c.getShard(k)
		s.Lock()
		if err := s.Set(ctx, k, values[i], o.TTL); err != nil {
			s.Unlock()
			return err
		}
		s.Unlock()
	}
	return nil
}

func (c *cacheHandler[T, P]) Get(ctx context.Context, key string, opts ...Option) (interface{}, error) {
	o := c.getOption(opts...)

	s := c.getShard(key)
	s.Lock()
	defer s.Unlock()

	val, err := s.Get(ctx, key)
	if err == KeyNotFoundError {
		if o.RealLoaderFunc == nil {
			return nil, KeyNotFoundError
		}

		val, err := o.RealLoaderFunc(ctx, key)
		if err != nil {
			if !errors.Is(err, DefValSetError) {
				return nil, err
			}

			if errors.Is(err, DefValSetError) {
				o.TTL = time.Minute
			}

			if err := s.Set(ctx, key, val, o.TTL); err != nil {
				return nil, err
			}

			if errors.Is(err, DefValSetError) {
				return val, err
			}
		}
	}

	return val, nil
}

func (c *cacheHandler[T, P]) MGet(ctx context.Context, keys []string, opts ...Option) (map[string]interface{}, error) {
	o := c.getOption(opts...)

	res := make(map[string]interface{}, len(keys))
	miss := make(map[string]P, len(keys))
	for _, key := range keys {
		s := c.getShard(key)
		s.Lock()
		val, err := s.Get(ctx, key)
		if err == nil {
			res[key] = val
		} else if err == KeyNotFoundError {
			miss[key] = s
		}
		s.Unlock()
	}

	if len(miss) > 0 {
		if o.RealMLoaderFunc == nil {
			goto END
		}

		keys := make([]string, 0, len(miss))
		for key := range miss {
			keys = append(keys, key)
		}
		kvs, err := o.RealMLoaderFunc(ctx, keys)
		if err != nil {
			goto END
		}

		for key, val := range kvs {
			s := miss[key]
			s.Lock()
			err := s.Set(ctx, key, val, o.TTL)
			s.Unlock()
			if err != nil {
				goto END
			}

			res[key] = val
		}
	}

END:
	return res, nil
}

func (c *cacheHandler[T, P]) Remove(ctx context.Context, key string) bool {
	if c.redisCli.Client != nil {
		err := c.redisCli.del(ctx, key)
		if err != nil {
			return false
		}
	}

	s := c.getShard(key)
	s.Lock()
	defer s.Unlock()

	return s.Remove(ctx, key)
}

func (c *cacheHandler[T, P]) MRemove(ctx context.Context, keys []string) bool {
	if c.redisCli.Client != nil {
		err := c.redisCli.mdel(ctx, keys)
		if err != nil {
			return false
		}
	}

	for _, key := range keys {
		s := c.getShard(key)
		s.Lock()
		if !s.Remove(ctx, key) {
			s.Unlock()
			return false
		}
		s.Unlock()
	}

	return true
}

func (c *cacheHandler[T, P]) Exists(ctx context.Context, key string) bool {
	s := c.getShard(key)
	s.Lock()
	defer s.Unlock()

	return s.Exists(ctx, key)
}

func (c *cacheHandler[T, P]) getShard(key string) P {
	return c.shards[c.DebugShardIndex(key)]
}

func (c *cacheHandler[T, P]) DebugShardIndex(key string) uint64 {
	return MemHashString(key) & uint64(c.shardCount-1)
}

func (c *cacheHandler[T, P]) debugFromLocal2(ctx context.Context, key string, onLoad bool) (interface{}, error) {
	s := c.getShard(key)
	val, err := s.Get(ctx, key)
	if err != nil {
		if !onLoad {
		}
		return nil, err
	}

	return val, nil
}

func (c *cacheHandler[T, P]) debugFromLocal(ctx context.Context, key string, onLoad bool) (interface{}, error) {
	s := c.getShard(key)
	s.Lock()
	defer s.Unlock()

	val, err := s.Get(ctx, key)
	if err != nil {
		if !onLoad {
		}
		return nil, err
	}

	return val, nil
}

func (c *cacheHandler[T, P]) debugRemove(ctx context.Context, key string) bool {
	s := c.getShard(key)
	s.Lock()
	defer s.Unlock()

	return s.Remove(ctx, key)
}

func (c *cacheHandler[T, P]) serialize(ctx context.Context, val interface{}, opts ...Option) ([]byte, error) {
	o := c.getOption(opts...)

	if o.serializeFunc != nil {
		return o.serializeFunc(ctx, val)
	}

	if c.serializeFunc != nil {
		return c.serializeFunc(ctx, val)
	}
	return nil, errors.New("mcache: must set WithSafeValPtrFunc option!")
}

func (c *cacheHandler[T, P]) deserialize(ctx context.Context, data []byte, opts ...Option) (interface{}, error) {
	o := c.getOption(opts...)

	if o.deserializeFunc != nil {
		return o.deserializeFunc(ctx, data)
	}

	if c.deserializeFunc != nil {
		return c.deserializeFunc(ctx, data)
	}
	return nil, errors.New("mcache: must set WithSafeValPtrFunc option!")
}
