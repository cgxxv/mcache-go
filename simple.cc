// simple.cc
#include <cstdlib>
#include "mcache/src/simple.hpp"

extern "C" {
#include "cache.h"
#include "simple.h"
}

using mcache::Simple;

simple new_simple(std::size_t _max_cap) {
    return (simple*)(new Simple<t_key, t_val*>(_max_cap));
}

std::size_t simple_put(simple cc, const t_key *key, const t_val *_value, const size_t vlen) {
    t_val *value = nullptr; //std::calloc(0, vlen*sizeof(t_val*));
    std::memcpy(value, _value, vlen);
    return ((Simple<t_key, t_val*> *)cc)->Put(*key, &value);
}

t_val *simple_get(simple cc, const t_key *key) {
    try {
        return ((Simple<t_key, t_val*> *)cc)->Get(*key);
    } catch (const std::exception &e) {
        std::cerr << e.what() << '\n';
        return nullptr;
    }
}

int simple_has(simple cc, const t_key *key) {
    return ((Simple<t_key, t_val*> *)cc)->Has(*key);
}
int simple_remove(simple cc, const t_key *key) {
    return ((Simple<t_key, t_val*> *)cc)->Remove(*key);
}
void simple_evict(simple cc, int count) {
    return ((Simple<t_key, t_val*> *)cc)->Evict(count);
}
std::size_t simple_size(simple cc) {
    return ((Simple<t_key, t_val*> *)cc)->Size();
}
void simple_debug(simple cc) { return ((Simple<t_key, t_val*> *)cc)->debug(); }