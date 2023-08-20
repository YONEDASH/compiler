# ??? compiler

maybe compile/transpile .** files to C or Java.

- Scopes: once a scope is no longer used, memory will be freed

Set default values for functions
```
fn test(int i = 0) {

}

fn getWelcomeMessage(string greeting = "Hello", string name) string {
    return greeting + " " + name
}

fn main() {
    test() // valid i defaults to 0
    const msg = getWelcomeMessage(_, "Til")
    printf(msg + "\n")
}
```