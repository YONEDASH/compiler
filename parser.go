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
	mutableLeft := left // Change this variable
	leftPtr := &left

	if err != nil {
		return Statement{}, err
	}

	for {
		if parser.isDone() {
			break
		}

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

		mutableLeft = Statement{
			Type:     BinaryExpression,
			Left:     leftPtr,
			Right:    rightPtr,
			Operator: operation,
		}
		left := mutableLeft
		leftPtr = &left
	}

	return mutableLeft, nil
	//return parsePrimaryExpression(parser)
}

func parseMultiplicativeExpression(parser *tokenParser) (Statement, error) {
	left, err := parsePrimaryExpression(parser)
	mutableLeft := left // Change this variable
	leftPtr := &left

	if err != nil {
		return Statement{}, err
	}

	for {
		if parser.isDone() {
			break
		}

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

		mutableLeft = Statement{
			Type:     BinaryExpression,
			Left:     leftPtr,
			Right:    rightPtr,
			Operator: operation,
		}
		left := mutableLeft
		leftPtr = &left
	}

	return mutableLeft, nil
	//return parsePrimaryExpression(parser)
}

func parsePrimaryExpression(parser *tokenParser) (Statement, error) {
	expression := Statement{}

	token := parser.current()

	switch token.Type {
	case Null:
		parser.consume()
		return Statement{
			Type: NullExpression,
		}, nil
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
	case Function:
		parser.consume() // Consume function keyword

		// Get identifier
		current := parser.consume()

		if current.Type != Identifier {
			return Statement{}, parseError(current, "No identifier for function declaration given")
		}

		name := current.Value

		// Check for parenthesis
		current = parser.consume()

		if current.Type != OpenParenthesis {
			return Statement{}, parseError(current, "Open parenthesis expected for function declaration")
		}

		// Check for arguments
		argTypes := []LangType{}
		argNames := []string{}

		mode := 0

		for {
			current = parser.current()

			if parser.isDone() {
				break
			}

			if current.Type == CloseParenthesis {
				parser.consume()
				break
			}

			if current.Type != Identifier {
				return Statement{}, parseError(current, "Expected identifier or type for function arguments")
			}

			parser.consume()

			// TODO: check for actual types
			if mode == 0 {
				langType, err := parseType(current)

				if err != nil {
					return Statement{}, err
				}

				argTypes = append(argTypes, langType)
				mode = 1
			} else {
				argNames = append(argNames, current.Value)

				mode = 0
			}
		}

		if len(argNames) != len(argTypes) {
			return Statement{}, parseError(parser.before(), "No identifier for type set in function declaration")
		}

		returnTypes := []LangType{}

		for {
			current = parser.current()

			if parser.isDone() {
				break
			}

			if current.Type == OpenCurlyBracket {
				break
			}

			if current.Type != Identifier {
				break
			}

			current := parser.consume()

			langType, err := parseType(current)

			if err != nil {
				return Statement{}, err
			}

			fmt.Println(parser.index, current)

			returnTypes = append(returnTypes, langType)
		}

		// Default to void
		if len(returnTypes) == 0 {
			returnTypes = append(returnTypes, Void)
		}

		// Consume curly bracket
		if current.Type != OpenCurlyBracket {
			return Statement{}, parseError(current, "Function body needs to be opened with {")
		}

		parser.consume()

		children := []Statement{}

		// Check inside
		for {
			current = parser.current()

			if current.Type == CloseCurlyBracket {
				parser.consume()
				break
			}

			statement, err := parseExpression(parser)

			if err != nil {
				return Statement{}, err
			}

			// Do not allow function declaration in functions
			if statement.Type == FunctionDeclaration {
				return Statement{}, parseError(current, "Cannot declare function inside of function")
			}

			children = append(children, statement)

		}

		functionDecl := Statement{
			Type:        FunctionDeclaration,
			Value:       name,
			ReturnTypes: returnTypes,
			ArgTypes:    argTypes,
			ArgNames:    argNames,
			Children:    children,
		}

		return functionDecl, nil
	}

	return expression, parseError(token, "Unexpected token")
}

func parseType(token Token) (LangType, error) {
	switch token.Value {
	case "int":
		return Int, nil
	case "float":
		return Float, nil
	default:
		return Void, parseError(token, "Unknown type")
	}
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
