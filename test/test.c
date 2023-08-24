#include "sys/types.h"
#include "stdlib.h"
int32_t main() {
    int32_t UNUSED = 10;
    free(UNUSED);
    int32_t HELLO = 1;
    int32_t WORLD = 2;
    int32_t STR = HELLO;
    STR = HELLO+WORLD;
    free(WORLD);
    free(HELLO);
    free(STR);
}

