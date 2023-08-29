// Import printf function from C
import native ("stdio.h")
fn native printf(string..?) -> int

fn println(string..? f) {
    printf(f, "lol")
    printf("\n")
}

// Call printf function
fn main() -> int {
    printf("Hello World! %i\n", 99)
}