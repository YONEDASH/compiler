# ??? compiler

work in progress!

maybe compile/transpile .** files to C or Java.

# Todos

## Now
- Function expressions / calls
- String type
- Arg... & Arrays

## Later
- Standard library
- VariableDeclarations and VariableAssignments for only one decl/assign each -> expand (x, y) = (123, 456) to 2 single statements 
- main fn should be a void. exit code with Exit(0)
- first parse root (note function) THEN parse functions
- parse arrays as identifiers: IdentifierExpression with ArraySizes set example: int[4][5] is ArraySizes: []int{ 4, 5}
- memory freeing and allocation -> variables should only live in scope: destroy memory once scope is left
- optimize memory allocation to free variables/memory as soon as possible in scope (for example once variable is used for the last time)
- pass memory allocated state of variable to new variable, for example copies or returns
- compile function should return more information on what it actually did

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
