import native ("stdio.h")

fn native printf(string... s) -> int

fn main() -> int {
    printf("Hello World!\n")
}