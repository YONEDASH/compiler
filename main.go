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
}
