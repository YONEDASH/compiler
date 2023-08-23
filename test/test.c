#include "sys/types.h"
struct Comet_INTERNAL_boolean {
    unsigned int value : 1;
};
int32_t test() {
    int32_t t1 = 10;
    struct Comet_INTERNAL_boolean t2 = { value: 1 };
    // TODO FREE MEMORY
    t1 = 20;
    // TODO FREE MEMORY
    t2.value = 0;
    // TODO free(t1);
    // TODO free(t2);
}

void test() {
}

