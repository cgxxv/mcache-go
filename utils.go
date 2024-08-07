package mcache

import (
	"fmt"
	"math/rand"
	"reflect"
	"unsafe"
)

func bytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func stringToBytes(s *string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{*s, len(*s)},
	))
}

func deref(val interface{}) interface{} {
	vt := reflect.ValueOf(val)
	for {
		if vt.Kind() != reflect.Ptr {
			return vt.Interface()
		}

		vt = vt.Elem()
	}
}

func RandString(l int) string {
	buf := make([]byte, l)
	for i := 0; i < (l+1)/2; i++ {
		buf[i] = byte(rand.Intn(256))
	}
	return fmt.Sprintf("%x", buf)[:l]
}
