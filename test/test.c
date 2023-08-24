#include "stdlib.h"
#include "sys/types.h"
free(FACTOR);
const int32_t FACTOR = 100;
int32_t test() {
    int32_t HELLO = 1;
    int32_t WORLD = 2;
    int32_t STR = HELLO;
    free(STR);
    free(WORLD);
    free(HELLO);
    STR = HELLO+WORLD;
}

