package main

import "fmt"

type ParseError struct {
	message string
	trace   *SourceTrace
}

func (e ParseError) Error() string {
	return e.message
}

type tokenParser struct {
	tokens *[]Token
	length int
	index  int
}

func (r tokenParser) at(i int) Token {
	if i < 0 || i >= r.length {
		return Token{}
	}

	return (*r.tokens)[i]
}

func (r tokenParser) before() Token {
	return r.at(r.index - 1)
}

func (r tokenParser) after() Token {
	return r.at(r.index + 1)
}

func (r tokenParser) current() Token {
	return r.at(r.index)
}

func (r *tokenParser) consume() Token {
	i := r.index
	r.index = i + 1
	return r.at(i)
}

func (r tokenParser) isDone() bool {
	return r.index >= r.length || r.at(r.index).Type == EOF
}

func ParseTokens(tokens []Token) (Statement, error) {
	parser := tokenParser{
		tokens: &tokens,
		length: len(tokens),
		index:  0,
	}

	children := []Statement{}

	for {
		if parser.isDone() {
			break
		}

		statement, err := parseStatement(&parser)

		if err != nil {
			return Statement{}, err
		}

		// fmt.Printf("%d %+v\n", parser.index, statement)

		children = append(children, statement)
	}

	root := Statement{
		Type:     Root,
		Children: children,
	}

	return root, nil
}

func parseStatement(parser *tokenParser) (Statement, error) {
	// statement := Statement{}

	// Implement later

	return parseExpression(parser)
}

func parseExpression(parser *tokenParser) (Statement, error) {
	/*expression := Statement{}

	return expression, nil*/
	return parseAdditiveExpression(parser)
}

func parseAdditiveExpression(parser *tokenParser) (Statement, error) {
	left, err := parseMultiplicativeExpression(parser)

	if err != nil {
		return Statement{}, err
	}

	for {
		token := parser.current()

		if token.Type != Addition && token.Type != Subtraction {
			break
		}

		operatorType := parser.consume().Type

		operation := AdditionOperation
		if operatorType != Addition {
			operation = SubtractionOperation
		}

		right, err := parseMultiplicativeExpression(parser)

		if err != nil {
			return Statement{}, err
		}

		initialLeft := &left
		left = Statement{
			Type:     BinaryExpression,
			Left:     initialLeft,
			Right:    &right,
			Operator: operation,
		}
	}

	return left, nil
	//return parsePrimaryExpression(parser)
}

func parseMultiplicativeExpression(parser *tokenParser) (Statement, error) {
	left, err := parsePrimaryExpression(parser)

	if err != nil {
		return Statement{}, err
	}

	for {
		token := parser.current()

		if token.Type != Multiplication && token.Type != Division && token.Type != Modulus {
			break
		}

		operatorType := parser.consume().Type

		operation := MultiplicationOperation
		if operatorType == Division {
			operation = DivisionOperation
		} else {
			operation = ModulusOperation
		}

		right, err := parsePrimaryExpression(parser)

		if err != nil {
			return Statement{}, err
		}

		initialLeft := &left
		left = Statement{
			Type:     BinaryExpression,
			Left:     initialLeft,
			Right:    &right,
			Operator: operation,
		}
	}

	return left, nil
	//return parsePrimaryExpression(parser)
}

func parsePrimaryExpression(parser *tokenParser) (Statement, error) {
	expression := Statement{}

	token := parser.current()

	switch token.Type {
	case Identifier:
		parser.consume()
		return Statement{
			Type:  IdentifierExpression,
			Value: token.Value,
		}, nil
	case Number:
		parser.consume()
		return Statement{
			Type:  NumberExpression,
			Value: token.Value,
		}, nil
	}

	parser.consume()
	return expression, unexpectedTokenError(token, "")
}

func unexpectedTokenError(token Token, message string) error {
	// Return error if unknown character is in source
	trace := token.Trace
	row, col := trace.Row, trace.Column

	msg := fmt.Sprintf("Unexpected token @ %d:%d '%+v'", row, col, token)

	return ParseError{message: msg, trace: trace}
}

func PrintAST(statement Statement, i int) {
	if i > 10 {
		return
	}

	prefix := ""
	for j := 0; j < i; j++ {
		prefix += "\t"
	}

	fmt.Println(prefix, "Type:", statement.Type)
	fmt.Println(prefix, "Value:", statement.Value)
	fmt.Println(prefix, "Operator:", statement.Operator)

	if statement.Left != nil {
		fmt.Println(prefix, "Left: ")

		if statement.Left == &statement {
			fmt.Println(prefix, "Itself??")
		} else {
			PrintAST(*statement.Left, i+1)
		}
	}
	if statement.Right != nil {
		fmt.Println(prefix, "Right: ")

		if statement.Right == &statement {
			fmt.Println(prefix, "Itself??")
		} else {
			PrintAST(*statement.Right, i+1)
		}
	}
	if len(statement.Children) > 0 {
		fmt.Println(prefix, "Children: ")
		for _, child := range statement.Children {
			if &child == &statement {
				fmt.Println(prefix, "Itself??")
				continue
			}

			PrintAST(child, i+1)
		}
	}
}
