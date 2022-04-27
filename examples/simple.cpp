#include <iostream>
#include <string>
#include <vector>
#include <cstdlib>
#include <cassert>
#include <random>

extern "C" {
#include "../cache.h"
#include "../simple.h"
}

int rand() {
    std::random_device crypto_random_generator;
    std::uniform_int_distribution<int> int_distribution(0, 9);

    return int_distribution(crypto_random_generator);
}

void demo(simple cc);

int main() {
    for (int i = 0; i < 100; i++) {
        demo(new_simple(100));
    }
}

void demo(simple cc) {
    const int count = 10;
    std::vector<int> keys = {0,1,2,3,4,5,6,7,8,9};
    std::vector<int> vals = {10,11,12,13,14,15,16,17,18,19};
    for (int i = 0; i <count;i++) {
        int ret = simple_put(cc, &keys[i], (t_val*)&vals[i]);
        assert(ret == 1);
    }

    int xk = 8;
    t_val xv = simple_get(cc, &xk);
    assert(xv != NULL);
    assert(*(int*)xv == vals[xk]);

    /*
      front->0->1->2->3->4->5->6->7->8->9->back->
    <-front<-0<-1<-2<-3<-4<-5<-6<-7<-8<-9<-back
    */

    for (int i = 0; i <count;i++) {
        t_val v = simple_get(cc, &keys[i]);
        assert(v != NULL);
        assert(*(int*)v == vals[i]);
    }

    for (int i = 0; i < 10e3; i++)
    {
        int r = rand();
        // std::cout << r << std::endl;
        t_val v = simple_get(cc, &r);
        assert(v != NULL);
        assert(*(int*)v == vals[r]);
    }

    simple_debug(cc);
    std::cout << std::endl;

    int spec = 5;
    for (int i = 0; i <count;i++) {
        const void *v = simple_get(cc, &keys[i]);
        assert(v != NULL);
        assert(*(int*)v == vals[i]);
        assert(simple_has(cc, &keys[i]) == 1);

        if (i < spec) {
            assert(simple_has(cc, &spec) == 1);
        } else if (i > spec) {
            assert(simple_has(cc, &spec) == 0);
        } else {
            assert(simple_remove(cc, &keys[i]) == 1);
            assert(vals[i] == 10+i);
        }
    }

    simple_evict(cc, 1);

    simple_debug(cc);
    std::cout << std::endl;

    assert(simple_size(cc) == 8);

    //assert(!cc.Has(5));
    //assert(vals[5] == 10+keys[5]);
    //assert(keys[5] == 5);

    // assert(vals[0] == cc.Get(0));
    // assert(vals[0] == 10);
    // int v = 10000;
    // vals[0] = v;
    // assert(cc.Get(0) == 10);
}
