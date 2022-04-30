package mcache

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-redis/redis/v8"
	"github.com/vmihailenco/msgpack/v5"
)

const maxBatchExecLength = 1000

type (
	serializeFunc   func(context.Context, interface{}) ([]byte, error)
	deserializeFunc func(context.Context, []byte) (interface{}, error)
)

type RedisCli struct {
	*redis.Client
}

func (r *RedisCli) mget(ctx context.Context, keys []string, opt options) (map[string]interface{}, error) {
	res := make(map[string]interface{}, len(keys))
	if r.Client == nil {
		return res, RedisNotFoundError
	}

	var (
		pipelined int
		cmders    = make([]*redis.StringCmd, 0, len(keys))
		pipe      = r.Pipeline()
	)
	defer pipe.Close()

	for _, key := range keys {
		if pipelined > maxBatchExecLength {
			if _, err := pipe.Exec(ctx); err != nil {

				goto RESULT
			} else {
				pipelined = 0
			}
		}
		cmder := pipe.Get(ctx, key)
		cmders = append(cmders, cmder)
		pipelined++
	}

	if _, err := pipe.Exec(ctx); err != nil {

	}

RESULT:
	for index, cmder := range cmders {
		reply, err := cmder.Bytes()
		if err != nil {
			continue
		}
		if opt.deserializeFunc != nil {
			val, err := opt.deserializeFunc(ctx, reply)
			if err != nil {
				continue
			}
			res[keys[index]] = val
		} else {
			res[keys[index]] = reply
		}
	}

	return res, nil
}

func (r *RedisCli) get(ctx context.Context, key string, opt options) (interface{}, error) {
	if r.Client == nil {
		return nil, RedisNotFoundError
	}

	if v, err := r.Get(ctx, key).Bytes(); err == nil && len(v) > 0 {
		if opt.deserializeFunc != nil {
			val, err := opt.deserializeFunc(ctx, v)
			if err != nil {
				return nil, err
			}
			return val, nil
		}
		return v, err
	}

	return nil, KeyNotFoundError
}

func (r *RedisCli) msetmap(ctx context.Context, kv map[string]interface{}, opt options) error {

	rks := make([]string, 0, len(kv))
	rvs := make([]interface{}, 0, len(kv))
	for k, v := range kv {
		rks = append(rks, k)
		rvs = append(rvs, v)
	}
	return r.mset(ctx, rks, rvs, opt)
}

func (r *RedisCli) mset(ctx context.Context, keys []string, values []interface{}, opt options) error {
	if r.Client == nil {
		return RedisNotFoundError
	}

	if len(keys) != len(values) {
		return KeyValueLenError
	}

	var (
		err       error
		pipelined int
		pipe      = r.Pipeline()
	)
	defer pipe.Close()

	for i, key := range keys {
		var val interface{}
		if opt.serializeFunc != nil {
			val, err = opt.serializeFunc(ctx, values[i])
			if err != nil {

				continue
			}
		} else {
			val = values[i]
		}

		if pipelined > maxBatchExecLength {
			if _, err := pipe.Exec(ctx); err != nil {
				return err
			} else {
				pipelined = 0
			}
		}

		if opt.TTL == 0 {
			pipe.Set(ctx, key, val, 0)
		} else {
			pipe.SetEX(ctx, key, val, opt.TTL)
		}
		pipelined++
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}

	return nil
}

func (r *RedisCli) set(ctx context.Context, key string, val interface{}, opt options) error {
	if r.Client == nil {
		return RedisNotFoundError
	}

	var (
		err error
	)
	if opt.serializeFunc != nil {
		val, err = opt.serializeFunc(ctx, val)
		if err != nil {
			return err
		}
	}

	if opt.TTL == 0 {
		err = r.Set(ctx, key, val, 0).Err()
	} else {
		err = r.SetEX(ctx, key, val, opt.TTL).Err()
	}
	return err
}

func (r *RedisCli) del(ctx context.Context, key string) error {
	if r.Client == nil {
		return RedisNotFoundError
	}

	return r.Del(ctx, key).Err()
}

func (r *RedisCli) mdel(ctx context.Context, keys []string) error {
	if r.Client == nil {
		return RedisNotFoundError
	}

	var (
		pipelined int
		pipe      = r.Pipeline()
	)
	defer pipe.Close()

	for _, key := range keys {
		if pipelined > maxBatchExecLength {
			if _, err := pipe.Exec(ctx); err != nil {
				return err
			} else {
				pipelined = 0
			}
		}

		pipe.Del(ctx, key)
		pipelined++
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}

	return nil
}

type dataWrapper interface {
	marshal(context.Context, interface{}) ([]byte, error)
	unmarshal(context.Context, []byte, interface{}) (interface{}, error)
}

type msgpackSerializer struct{}

func (s *msgpackSerializer) marshal(ctx context.Context, o interface{}) (bs []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("mcache: marshal panic error, %v", r)
			bs = nil
		}
	}()

	bs, err = msgpack.Marshal(o)
	if err != nil {
		return nil, err
	}
	return
}

func (s *msgpackSerializer) unmarshal(ctx context.Context, d []byte, o interface{}) (v interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("mcache: unmarshal panic error, %v", r)
			v = nil
		}
	}()

	err = msgpack.Unmarshal(d, o)
	if err == nil {
		v = reflect.ValueOf(o).Elem().Interface()
		return
	}
	return nil, err
}

func newMsgpackSerializer() dataWrapper {
	return &msgpackSerializer{}
}

type serializer interface {
	serialize(context.Context, interface{}, ...Option) ([]byte, error)
	deserialize(context.Context, []byte, ...Option) (interface{}, error)
}
