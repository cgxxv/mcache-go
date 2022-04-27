package main

// #include <stdio.h>
// #include <stdlib.h>
// typedef struct {
//     int   size;
//     void *data;
// } info;
//
// void test(info *infoPtr) {
//     printf("FUCK %d\n", infoPtr->size);
// }
import "C"

import "unsafe"

func main() {
	var data uint8 = 5

	cdata := C.malloc(C.size_t(unsafe.Sizeof(data)))
	*(*C.char)(cdata) = C.char(data)
	defer C.free(cdata)

	info := &C.info{size: C.int(unsafe.Sizeof(data)), data: cdata}
	C.test(info)
}
