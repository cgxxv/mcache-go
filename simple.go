package mcache

// #cgo LDFLAGS: -L. -lstdc++
// #cgo CXXFLAGS: -std=c++17 -I.
// #include "./mcache/src/simple.h"
import "C"
import (
	"sync"
)

// simpleCache evict expired item lazily.
type simpleCache struct {
	simple C.simple
	cap    uint
	sync.RWMutex
}

func (m *simpleCache) init(_cap uint) {
	m.cap = _cap //defaultShardCap
	m.simple = C.new_simple(C.size_t(_cap))
}

// func (m *simpleCache) set(ctx context.Context, key string, val interface{}, o *options) error {
// 	ret := int(C.put((C.simple)(unsafe.Pointer(m.simple)), (C.t_key)(unsafe.Pointer(&key)), (C.t_value)(unsafe.Pointer(&val))))
// 	println(ret)
// 	return nil
// }

// func (m *simpleCache) get(ctx context.Context, key string) (interface{}, error) {
// 	val := C.get((C.simple)(unsafe.Pointer(m.simple)), (C.t_key)(unsafe.Pointer(&key)))
// 	if val != nil {
// 		return val, nil
// 	}

// 	return nil, errors.New("Not found key")
// }

// func (m *simpleCache) has(ctx context.Context, key string) bool {
// 	return C.has((C.simple)(unsafe.Pointer(m.simple)), (C.t_key)(unsafe.Pointer(&key)))
// }

// func (m *simpleCache) remove(ctx context.Context, key string) bool {
// 	return C.remove((C.simple)(unsafe.Pointer(m.simple)), (C.t_key)(unsafe.Pointer(&key)))
// }

// func (m *simpleCache) evict(ctx context.Context, count int) {
// 	C.evict((C.simple)(unsafe.Pointer(m.simple)), (C.int)(unsafe.Pointer(&count)))
// }

// func (m *simpleCache) size(ctx context.Context) int {
// 	return C.size((C.simple)(unsafe.Pointer(m.simple)))
// }
