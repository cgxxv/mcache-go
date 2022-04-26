package mcache

import (
	"time"
)

type Option func(*options)

type options struct {
	Name string
	Typ  string // 缓存类型 simple/lru等.
	// RedisCli        RedisCli       // redis 二级缓存.
	Exp *time.Duration // Exp 缓存的过期时间 未设置时，采用mcache的过期时间.
	// LoaderFunc      LoaderFunc     // LoaderFunc 当缓存获取不到时，会执行此方法保底，并将放回结果设置到各层缓存中.
	// RealLoaderFunc  LoaderFunc     // 封装redis + LoaderFunc
	// MLoaderFunc     MLoaderFunc    // MLoaderFunc 当缓存获取不到时，会执行此方法保底，并将放回结果设置到各层缓存中 批量.
	// RealMLoaderFunc MLoaderFunc    // 封装redis + MLoaderFunc
	IsWait     bool        // IsWait 当针对一个key并发请求二级缓存，例如：redis 时，是否等待前面的请求返回.
	DefaultVal interface{} // DefaultVal 降级的默认值，默认有效期为1分钟.

	// value 序列化反序列化方法
	// serializeFunc   serializeFunc
	// deserializeFunc deserializeFunc
}

// // check 校验参数.
// func (o *options) check() bool {
// 	if o.Typ != TYPE_SIMPLE && o.Typ != TYPE_ARC && o.Typ != TYPE_LFU && o.Typ != TYPE_LRU {
// 		panic("mcache: Unknown type " + o.Typ)
// 	}

// 	// if size <= 0 && o.Typ != TYPE_SIMPLE {
// 	// 	panic("mcache: Cache size <= 0")
// 	// }

// 	return true
// }

func WithName(name string) Option {
	return func(o *options) {
		o.Name = name
	}
}

// WithCacheType 设置要实例化的缓存类型.
func WithCacheType(typ string) Option {
	return func(o *options) {
		o.Typ = typ
	}
}

// WithRedisCli 设置要实例化的缓存类型, 仅限于初始化使用.
// func WithRedisCli(pool redis.Pool) Option {
// 	return func(o *options) {
// 		o.RedisCli.cli = pool
// 	}
// }

// // WithSafeValPtrFn 动态绑定 redis 实例中单个 key 对应的 value 指针对象
// func WithSafeValPtrFn(valPtrFunc valPtrFunc) Option {
// 	return func(o *options) {
// 		var s = newMsgpackSerializer()
// 		o.serializeFunc = func(ctx context.Context, val interface{}) (bs []byte, err error) {
// 			var (
// 				obj = valPtrFunc()
// 				ot  = reflect.TypeOf(obj)
// 				rt  = reflect.TypeOf(val)
// 			)
// 			defer func() {
// 				rt = nil
// 				ot = nil
// 				obj = nil
// 			}()

// 			if ot.Kind() == reflect.Ptr {
// 				ot = ot.Elem()
// 			}

// 			if rt.Kind() == reflect.Ptr {
// 				rt = rt.Elem()
// 			}

// 			if ot.Kind() != rt.Kind() {
// 				return nil, fmt.Errorf("mcache: unmached value type, expect %s, got %#v!", ot.Kind(), val)
// 			}

// 			return s.marshal(ctx, val)
// 		}
// 		o.deserializeFunc = func(ctx context.Context, data []byte) (v interface{}, err error) {
// 			var (
// 				obj = valPtrFunc()
// 				ot  = reflect.TypeOf(obj)
// 			)
// 			defer func() {
// 				ot = nil
// 				obj = nil
// 			}()
// 			if ot.Kind() != reflect.Ptr {
// 				return nil, errors.New("mcache: valPtrFn must return a pointer type!")
// 			}

// 			return s.unmarshal(ctx, data, obj)
// 		}
// 	}
// }

// WithExp 过期时间.
func WithExp(exp time.Duration) Option {
	return func(o *options) {
		o.Exp = &exp
	}
}

// // WithLoaderFn 当未能查询到缓存时的，使用该函数获取数据.
// func WithLoaderFn(fn LoaderFunc) Option {
// 	return func(o *options) {
// 		o.LoaderFunc = fn
// 	}
// }

// // WithMLoaderFn 当未能查询到缓存时的，使用该函数获取数据 批量.
// // NOTE: 这里必须要保证key，value 始终是成对出现的，所以最好的方式是使用map，否则程序可能会出现数据错乱
// func WithMLoaderFn(fn MLoaderFunc) Option {
// 	return func(o *options) {
// 		o.MLoaderFunc = fn
// 	}
// }

// WithIsWait 当针对一个key并发请求二级缓存，例如：redis 时，是否等待前面的请求返回.
func WithIsWait(isWait bool) Option {
	return func(o *options) {
		o.IsWait = isWait
	}
}

// WithDefaultVal 降级数据，默认有效期为1分钟.
func WithDefaultVal(defaultVal interface{}) Option {
	return func(o *options) {
		o.DefaultVal = defaultVal
	}
}

// // WithRedisAddrs 设置要实例化的缓存类型, 仅限于初始化使用.
// // NOTE: 如果在运行时不符合要求，会panic
// func WithRedisAddrs(masterAddr string, slaveAddrs []string, opts *RedisOptions) Option {
// 	var redisOptions = &RedisOptions{}
// 	if opts != nil {
// 		redisOptions = opts
// 	}

// 	if !strings.Contains(masterAddr, "://") {
// 		panic(fmt.Errorf("mcache: masterAddr must containe protocol."))
// 	}

// 	master, err := url.Parse(masterAddr)
// 	if err != nil {
// 		panic(fmt.Errorf("mcache: %v", err))
// 	}
// 	redisOptions.MasterAddr = master
// 	for _, addr := range slaveAddrs {
// 		slave, err := url.Parse(addr)
// 		if err != nil {
// 			panic(fmt.Errorf("mcache: %v", err))
// 		}
// 		redisOptions.SlaveAddrs = append(redisOptions.SlaveAddrs, slave)
// 	}
// 	pool, err := redis.DiscoveryWithOptions("", redisOptions)
// 	if err != nil {
// 		panic(fmt.Errorf("mcache: %v", err))
// 	}
// 	return func(o *options) {
// 		o.RedisCli.cli = pool
// 	}
// }

// // RedisOptions
// type RedisOptions = redis.Options
