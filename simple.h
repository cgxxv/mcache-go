// simple.h
#ifndef _mcache_simple_h_
#define _mcache_simple_h_

#include <stdlib.h>
// #include "cache.h"

typedef void *simple;
simple new_simple(size_t _max_cap);
size_t simple_put(simple cc, const char *key, const void **value);
void *simple_get(simple cc, const char *key);
int simple_has(simple cc, const char *key);
int simple_remove(simple cc, const char *key);
void simple_evict(simple cc, int count);
size_t simple_size(simple cc);
void simple_debug(simple cc);

#endif
