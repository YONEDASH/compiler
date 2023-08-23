package main

import (
	"fmt"
	"os"

	"github.com/yonedash/comet/compiler"
	"github.com/yonedash/comet/context"
	"github.com/yonedash/comet/lexer"
	"github.com/yonedash/comet/parser"
)

func main() {
	tokens, err := lexer.Tokenize("test.cl")

	if err != nil {
		fmt.Println(err)
		return
	}

	for _, token := range tokens {
		fmt.Println(token)
	}

	statement, err := parser.ParseTokens(tokens)

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("BEFORE POPULATION")
	parser.PrintAST(statement, 0)

	hints, err := context.Grow(&statement)

	fmt.Println("AFTER POPULATION")
	parser.PrintAST(statement, 0)

	if err != nil {
		fmt.Println(err)
		return
	}

	for _, hint := range hints {
		fmt.Println(hint.Message, hint.Statement.Trace)
	}

	c, err := compiler.CompileC(statement)

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(c)

	// Write to test file
	// create file
	f, err := os.Create("test/test.c")
	if err != nil {
		fmt.Println(err)
		return
	}
	// remember to close the file
	defer f.Close()

	f.WriteString(c)
}
