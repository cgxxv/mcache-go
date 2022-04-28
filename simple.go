package mcache

// #cgo LDFLAGS: -L. -lstdc++
// #cgo CXXFLAGS: -std=c++17 -I.
// #include "mcache.h"
import "C"
import (
	"context"
	"errors"
	"sync"
	"unsafe"
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

func (m *simpleCache) set(ctx context.Context, key string, val unsafe.Pointer, o *options) error {
	ck := C.CString(key)
	// defer C.free(unsafe.Pointer(ck))

	tk := C.t_key(ck)
	// defer C.free(unsafe.Pointer(tk))
	tv := C.t_val(val)
	// defer C.free(unsafe.Pointer(tv))

	ret := uint(C.simple_put(m.simple, &tk, &tv))
	println(ret)
	return nil
}

func (m *simpleCache) get(ctx context.Context, key string) (unsafe.Pointer, error) {
	ck := C.CString(key)
	// defer C.free(unsafe.Pointer(ck))

	tk := C.t_key(ck)
	// defer C.free(unsafe.Pointer(tk))

	val := C.simple_get(m.simple, &tk)
	if val != nil {
		return unsafe.Pointer(val), nil
	}

	return nil, errors.New("Not found key")
}

func (m *simpleCache) has(ctx context.Context, key string) bool {
	ck := C.CString(key)
	defer C.free(unsafe.Pointer(ck))

	tk := C.t_key(ck)
	defer C.free(unsafe.Pointer(tk))

	return C.simple_has(m.simple, &tk) == 1
}

func (m *simpleCache) remove(ctx context.Context, key string) bool {
	ck := C.CString(key)
	defer C.free(unsafe.Pointer(ck))

	tk := C.t_key(ck)
	defer C.free(unsafe.Pointer(tk))

	return C.simple_remove(m.simple, &tk) == 1
}

func (m *simpleCache) evict(ctx context.Context, count int) {
	C.simple_evict(m.simple, C.int(count))
}

func (m *simpleCache) size(ctx context.Context) uint64 {
	return uint64(C.simple_size(m.simple))
}

func (m *simpleCache) debug() {
	C.simple_debug(m.simple)
}
