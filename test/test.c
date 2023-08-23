#include "sys/types.h"
int32_t test() {
    void t1 = 10;
    void t2 = 1;
    t1 = 20;
    t2 = 0;
    // TODO free(t1);
    // TODO free(t2);
}

