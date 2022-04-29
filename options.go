package mcache

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/go-redis/redis/v8"
)

type Option func(*options)

type options struct {
	RedisCli        RedisCli
	TTL             time.Duration
	LoaderFunc      LoaderFunc
	RealLoaderFunc  LoaderFunc
	MLoaderFunc     MLoaderFunc
	RealMLoaderFunc MLoaderFunc
	IsWait          bool
	DefaultVal      interface{}

	serializeFunc   serializeFunc
	deserializeFunc deserializeFunc
}

func WithRedisClient(client *redis.Client) Option {
	return func(o *options) {
		o.RedisCli = RedisCli{client}
	}
}

func WithUnSafeValBind(valPtrFunc valPtrFunc) Option {
	return func(o *options) {
		var s = newMsgpackSerializer()
		o.serializeFunc = func(ctx context.Context, val interface{}) (bs []byte, err error) {
			var (
				obj = valPtrFunc()
				ot  = reflect.TypeOf(obj)
				rt  = reflect.TypeOf(val)
			)
			defer func() {
				rt = nil
				ot = nil
				obj = nil
			}()

			if ot.Kind() == reflect.Ptr {
				ot = ot.Elem()
			}

			if rt.Kind() == reflect.Ptr {
				rt = rt.Elem()
			}

			if ot.Kind() != rt.Kind() {
				return nil, fmt.Errorf("mcache: unmached value type, expect %s, got %#v!", ot.Kind(), val)
			}

			return s.marshal(ctx, val)
		}
		o.deserializeFunc = func(ctx context.Context, data []byte) (v interface{}, err error) {
			var (
				obj = valPtrFunc()
				ot  = reflect.TypeOf(obj)
			)
			defer func() {
				ot = nil
				obj = nil
			}()
			if ot.Kind() != reflect.Ptr {
				return nil, errors.New("mcache: valPtrFn must return a pointer type!")
			}

			return s.unmarshal(ctx, data, obj)
		}
	}
}

func WithTTL(ttl time.Duration) Option {
	return func(o *options) {
		o.TTL = ttl
	}
}

func WithLoaderFn(fn LoaderFunc) Option {
	return func(o *options) {
		o.LoaderFunc = fn
	}
}

func WithMLoaderFn(fn MLoaderFunc) Option {
	return func(o *options) {
		o.MLoaderFunc = fn
	}
}

func WithIsWait(isWait bool) Option {
	return func(o *options) {
		o.IsWait = isWait
	}
}

func WithDefaultVal(defaultVal interface{}) Option {
	return func(o *options) {
		o.DefaultVal = defaultVal
	}
}

func WithRedisOptions(opts *redis.Options) Option {
	return func(o *options) {
		o.RedisCli = RedisCli{redis.NewClient(opts)}
	}
}
