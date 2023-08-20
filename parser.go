package main

import (
	"fmt"
)

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
	leftPtr := &left
	newLeft := left

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
		rightPtr := &right

		if err != nil {
			return Statement{}, err
		}

		newLeft = Statement{
			Type:     BinaryExpression,
			Left:     leftPtr,
			Right:    rightPtr,
			Operator: operation,
		}
		leftPtr = &newLeft
	}

	return newLeft, nil
	//return parsePrimaryExpression(parser)
}

func parseMultiplicativeExpression(parser *tokenParser) (Statement, error) {
	left, err := parsePrimaryExpression(parser)
	leftPtr := &left
	newLeft := left

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
		} else if operatorType == Modulus {
			operation = ModulusOperation
		}

		right, err := parsePrimaryExpression(parser)
		rightPtr := &right

		if err != nil {
			return Statement{}, err
		}

		newLeft = Statement{
			Type:     BinaryExpression,
			Left:     leftPtr,
			Right:    rightPtr,
			Operator: operation,
		}
		leftPtr = &newLeft
	}

	return newLeft, nil
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
	case OpenParenthesis:
		parser.consume() // Consume opening

		wrappedExpression, err := parseExpression(parser)

		if err != nil {
			return Statement{}, nil
		}

		current := parser.current()

		if current.Type != CloseParenthesis {
			return Statement{}, parseError(current, "Parenthesis not closed")
		}

		parser.consume()

		return wrappedExpression, nil
	}

	return expression, parseError(token, "Unexpected token")
}

func parseError(token Token, message string) error {
	// Return error if unknown character is in source
	trace := token.Trace

	if trace == nil {
		return ParseError{message: "No trace found for error"}
	}

	row, col := trace.Row, trace.Column
	msg := fmt.Sprintf("%s @ %d:%d >> %+v", message, row, col, token)

	return ParseError{message: msg, trace: trace}
}
