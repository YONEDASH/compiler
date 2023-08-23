#include "sys/types.h"
#include "stdlib.h"
int32_t test() {
    float speed = 50.0;
    free(speed);
    float time = 60.0;
    const float s = speed;
    free(s);
    time = 5.0;
    free(time);
    const float v = s;
    free(v);
}

