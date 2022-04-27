package main

/*
#include <stdlib.h>
#include <stdio.h>

typedef int* pInt;

void foo(pInt p[]) { // you probably wanna pass a len to the function.
    *p[0] = 100;
    printf("foo()\n");
}

*/
import "C"
import "unsafe"

func main() {
	var (
		i, sz  = 0, 2
		arr    = (*C.pInt)(C.malloc(C.size_t(sz)))
		ps     = (*[100000]C.pInt)(unsafe.Pointer(arr))[:sz:sz]
		p1, p2 = (C.pInt)(unsafe.Pointer(&i)), (C.pInt)(unsafe.Pointer(&i))
	)
	ps[0], ps[1] = p1, p2
	C.foo(arr)
	C.free(unsafe.Pointer(arr))
	println("i", i)
}
