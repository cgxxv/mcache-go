package main

/*
#include <stdlib.h>
typedef struct User {
	int id;
	int age;
	int number;
} User;

typedef void *t_user;

static void createUser(void **pUser) {
	if(pUser) *pUser = malloc(sizeof(User));
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

type User C.User

func main() {

	pointer := unsafe.Pointer(nil) //void *t_user

	C.createUser(&pointer)

	user := (*User)(pointer)

	fmt.Println(user)
}
