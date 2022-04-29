package mcache

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

type Shard interface {
	init(clock Clock)
	set(ctx context.Context, key string, val interface{}, ttl time.Duration) error
	get(ctx context.Context, key string) (interface{}, error)
	has(ctx context.Context, key string) bool
	remove(ctx context.Context, key string) bool
	evict(ctx context.Context, count int)

	sync.Locker
}
type baseShard struct {
	baseCache

	shards []Shard
	procs  int32
}

func newShard(cb *CacheBuilder) Cache {
	c := &baseShard{}
	c.baseCache = cb.baseCache

	c.shards = make([]Shard, c.shardCount)
	for i := 0; i < c.shardCount; i++ {
		var s Shard

		switch c.typ {
		case typeSimple:
			s = &simpleCache{}
		case typeLfu:
			s = &lfuCache{}
		case typeLru:
			s = &lruCache{}
		case typeArc:
			s = &arcCache{}
		default:
			panic("unreachable")
		}
		s.init(c.clock)

		c.shards[i] = s
	}

	c.purge()
	return c
}

func (c *baseShard) purge() {
	if runUnitTest {
		return
	}

	const lockP = 50
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	go func() {
		for {
			for i := 0; i < c.shardCount; i++ {
				if atomic.LoadInt32(&c.procs) != 0 {
					continue
				}
				if r.Intn(100) >= lockP {
					continue
				}
				s := c.shards[i]
				s.Lock()
				s.evict(nil, 1)
				s.Unlock()
			}

			time.Sleep(time.Second)
		}
	}()
}

func (c *baseShard) Set(ctx context.Context, key string, value interface{}, opts ...Option) error {
	atomic.AddInt32(&c.procs, 1)
	defer atomic.AddInt32(&c.procs, -1)

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

	return s.set(ctx, key, value, o.TTL)
}

func (c *baseShard) MSet(ctx context.Context, keys []string, values []interface{}, opts ...Option) error {
	if len(keys) != len(values) {
		return KeyValueLenError
	}

	atomic.AddInt32(&c.procs, 1)
	defer atomic.AddInt32(&c.procs, -1)

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
		if err := s.set(ctx, k, values[i], o.TTL); err != nil {
			s.Unlock()
			return err
		}
		s.Unlock()
	}
	return nil
}

func (c *baseShard) Get(ctx context.Context, key string, opts ...Option) (interface{}, error) {
	atomic.AddInt32(&c.procs, 1)
	defer atomic.AddInt32(&c.procs, -1)

	o := c.getOption(opts...)

	s := c.getShard(key)
	s.Lock()
	defer s.Unlock()

	val, err := s.get(ctx, key)
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

			if err := s.set(ctx, key, val, o.TTL); err != nil {
				return nil, err
			}

			if errors.Is(err, DefValSetError) {
				return val, err
			}
		}
	}

	return val, nil
}

func (c *baseShard) MGet(ctx context.Context, keys []string, opts ...Option) (map[string]interface{}, error) {
	atomic.AddInt32(&c.procs, 1)
	defer atomic.AddInt32(&c.procs, -1)

	o := c.getOption(opts...)

	res := make(map[string]interface{}, len(keys))
	miss := make(map[string]Shard, len(keys))
	for _, key := range keys {
		s := c.getShard(key)
		s.Lock()
		val, err := s.get(ctx, key)
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
			err := s.set(ctx, key, val, o.TTL)
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

func (c *baseShard) Remove(ctx context.Context, key string) bool {
	if c.redisCli.Client != nil {
		err := c.redisCli.del(ctx, key)
		if err != nil {
			return false
		}
	}

	atomic.AddInt32(&c.procs, 1)
	defer atomic.AddInt32(&c.procs, -1)

	s := c.getShard(key)
	s.Lock()
	defer s.Unlock()

	return s.remove(ctx, key)
}

func (c *baseShard) MRemove(ctx context.Context, keys []string) bool {
	if c.redisCli.Client != nil {
		err := c.redisCli.mdel(ctx, keys)
		if err != nil {
			return false
		}
	}

	atomic.AddInt32(&c.procs, 1)
	defer atomic.AddInt32(&c.procs, -1)

	for _, key := range keys {
		s := c.getShard(key)
		s.Lock()
		if !s.remove(ctx, key) {
			s.Unlock()
			return false
		}
		s.Unlock()
	}

	return true
}

func (c *baseShard) Exists(ctx context.Context, key string) bool {
	atomic.AddInt32(&c.procs, 1)
	defer atomic.AddInt32(&c.procs, -1)

	s := c.getShard(key)
	s.Lock()
	defer s.Unlock()

	return s.has(ctx, key)
}

func (c *baseShard) getFromLocal2(ctx context.Context, key string, onLoad bool) (interface{}, error) {
	atomic.AddInt32(&c.procs, 1)
	defer atomic.AddInt32(&c.procs, -1)

	s := c.getShard(key)
	val, err := s.get(ctx, key)
	if err != nil {
		if !onLoad {
		}
		return nil, err
	}

	return val, nil
}

func (c *baseShard) getFromLocal(ctx context.Context, key string, onLoad bool) (interface{}, error) {
	s := c.getShard(key)
	s.Lock()
	defer s.Unlock()

	atomic.AddInt32(&c.procs, 1)
	defer atomic.AddInt32(&c.procs, -1)

	val, err := s.get(ctx, key)
	if err != nil {
		if !onLoad {
		}
		return nil, err
	}

	return val, nil
}

func (c *baseShard) getShard(key string) Shard {
	return c.shards[MemHashString(key)%uint64(c.shardCount)]
}

func (c *baseShard) serialize(ctx context.Context, val interface{}, opts ...Option) ([]byte, error) {
	o := c.getOption(opts...)
	if o.serializeFunc != nil {
		return o.serializeFunc(ctx, val)
	}

	if c.serializeFunc != nil {
		return c.serializeFunc(ctx, val)
	}
	return nil, errors.New("mcache: must set WithSafeValPtrFunc option!")
}

func (c *baseShard) deserialize(ctx context.Context, data []byte, opts ...Option) (interface{}, error) {
	o := c.getOption(opts...)
	if o.deserializeFunc != nil {
		return o.deserializeFunc(ctx, data)
	}

	if c.deserializeFunc != nil {
		return c.deserializeFunc(ctx, data)
	}
	return nil, errors.New("mcache: must set WithSafeValPtrFunc option!")
}

func (c *baseShard) remove(ctx context.Context, key string) bool {
	atomic.AddInt32(&c.procs, 1)
	defer atomic.AddInt32(&c.procs, -1)

	s := c.getShard(key)
	s.Lock()
	defer s.Unlock()

	return s.remove(ctx, key)
}
