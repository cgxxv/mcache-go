// simple.cc
#include "mcache/src/simple.hpp"

extern "C" {
#include "cache.h"
#include "simple.h"
}

using mcache::Simple;

simple new_simple(std::size_t _max_cap) {
    return (simple*)(new Simple<t_key, t_value>(_max_cap));
}

std::size_t simple_put(simple cc, t_key *key, t_value *value) {
    return ((Simple<t_key, t_value> *)cc)->Put(*key, *value);
}

t_value *simple_get(simple cc, t_key *key) {
    try {
        return (t_value*)&(((Simple<t_key, t_value> *)cc)->Get(*key));
    } catch (const std::exception &e) {
        std::cerr << e.what() << '\n';
        return nullptr;
    }
}

int simple_has(simple cc, t_key *key) {
    return ((Simple<t_key, t_value> *)cc)->Has(*key);
}
int simple_remove(simple cc, t_key *key) {
    return ((Simple<t_key, t_value> *)cc)->Remove(*key);
}
void simple_evict(simple cc, int count) {
    return ((Simple<t_key, t_value> *)cc)->Evict(count);
}
std::size_t simple_size(simple cc) {
    return ((Simple<t_key, t_value> *)cc)->Size();
}
void simple_debug(simple cc) { return ((Simple<t_key, t_value> *)cc)->debug(); }