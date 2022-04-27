package main

// #include <stdio.h>
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
	info := &C.info{size: C.int(unsafe.Sizeof(data)), data: unsafe.Pointer(&data)}
	C.test(info)
}
