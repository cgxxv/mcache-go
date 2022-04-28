package mcache

import (
	"reflect"
	"unsafe"
)

func minInt(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func maxInt(x, y int) int {
	if x > y {
		return x
	}
	return y
}

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

func vderef(val interface{}) interface{} {
	vt := reflect.ValueOf(val)
	for {
		if vt.Kind() != reflect.Ptr {
			return vt.Interface()
		}

		vt = vt.Elem()
	}
}
