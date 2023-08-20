package main

import "fmt"

func main() {
	tokens, err := Tokenize("test.cl")

	if err != nil {
		fmt.Println(err)
		return
	}

	for _, token := range tokens {
		fmt.Println(token)
	}

	statement, err := ParseTokens(tokens)

	if err != nil {
		fmt.Println(err)
		return
	}

	PrintAST(statement, 0)

	c := GenerateC(statement, 0)

	fmt.Println(c)
}
