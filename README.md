# ??? compiler

work in progress!

maybe compile/transpile .** files to C or Java.

- first parse root (note function) THEN parse functions

add tests:
- make sure int/double/float size are correct -> c code for sizeof(TYPE)

Scopes: once a scope is no longer used, memory will be freed

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

Static should prevent:

- main() function from having any return values
- check if const vars are being changed
- check if types are defined
