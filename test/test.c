#include <sys/types.h>
#include <stdio.h>

int main() {
    int32_t d = 0;
    uint64_t f = 0;
 float _Complex x = 0;
    printf("%lu\n", (sizeof(x) * 8));
    return 0;
}

