// simple.cc
#include "mcache/src/simple.hpp"

extern "C" {
// #include "cache.h"
#include "simple.h"
}

using mcache::Simple;

simple new_simple(std::size_t _max_cap) {
    return (simple*)(new Simple<char, void*>(_max_cap));
}

std::size_t simple_put(simple cc, const char *key, const void **value) {
    return ((Simple<char, void*> *)cc)->Put(*key, value);
}

void *simple_get(simple cc, const char *key) {
    try {
        return (void*)&((((Simple<char, void*> *)cc)->Get(*key)));
    } catch (const std::exception &e) {
        std::cerr << e.what() << '\n';
        return nullptr;
    }
}

int simple_has(simple cc, const char *key) {
    return ((Simple<char, void*> *)cc)->Has(*key);
}
int simple_remove(simple cc, const char *key) {
    return ((Simple<char, void*> *)cc)->Remove(*key);
}
void simple_evict(simple cc, int count) {
    return ((Simple<char, void*> *)cc)->Evict(count);
}
std::size_t simple_size(simple cc) {
    return ((Simple<char, void*> *)cc)->Size();
}
void simple_debug(simple cc) { return ((Simple<char, void*> *)cc)->debug(); }