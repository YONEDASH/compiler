package parser

import (
	"fmt"

	"github.com/yonedash/comet/lexer"
)

type ParseError struct {
	message string
	trace   *lexer.SourceTrace
}

func (e ParseError) Error() string {
	return e.message
}

type tokenParser struct {
	tokens *[]lexer.Token
	length int
	index  int
}

func (r tokenParser) at(i int) lexer.Token {
	if i < 0 || i >= r.length {
		return lexer.Token{}
	}

	return (*r.tokens)[i]
}

func (r tokenParser) before() lexer.Token {
	return r.at(r.index - 1)
}

func (r tokenParser) after() lexer.Token {
	return r.at(r.index + 1)
}

func (r tokenParser) current() lexer.Token {
	return r.at(r.index)
}

func (r *tokenParser) consume() lexer.Token {
	i := r.index
	r.index = i + 1
	return r.at(i)
}

func (r tokenParser) isDone() bool {
	return r.index >= r.length || r.at(r.index).Type == lexer.EOF
}

func ParseTokens(tokens []lexer.Token) (Statement, error) {
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

		if statement.Type < 0 {
			continue
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
	current := parser.current()
	switch current.Type {
	case lexer.LF: // Ignore line feed
		parser.consume()
		return Statement{Type: -1}, nil
	case lexer.OpenCurlyBracket:
		return parseScope(parser)
	case lexer.Function:
		return parseFunction(parser)
	case lexer.Var, lexer.Const:
		return parseVariableDeclaration(parser)
	}

	return Statement{}, parseError(current, "Unexpected token, statement expected")
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

		if token.Type != lexer.Addition && token.Type != lexer.Subtraction {
			break
		}

		operatorType := parser.consume().Type

		operation := AdditionOperation
		if operatorType != lexer.Addition {
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

		if token.Type != lexer.Multiplication && token.Type != lexer.Division && token.Type != lexer.Modulus {
			break
		}

		operatorType := parser.consume().Type

		operation := MultiplicationOperation
		if operatorType == lexer.Division {
			operation = DivisionOperation
		} else if operatorType == lexer.Modulus {
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
	case lexer.Identifier:
		parser.consume()
		return Statement{
			Type:  IdentifierExpression,
			Value: token.Value,
		}, nil
	case lexer.Number:
		parser.consume()
		return Statement{
			Type:  NumberExpression,
			Value: token.Value,
		}, nil
	case lexer.OpenParenthesis:
		parser.consume() // Consume opening

		wrappedExpression, err := parseExpression(parser)

		if err != nil {
			return Statement{}, nil
		}

		current := parser.current()

		if current.Type != lexer.CloseParenthesis {
			return Statement{}, parseError(current, "Parenthesis not closed")
		}

		parser.consume()

		return wrappedExpression, nil
	}

	return expression, parseError(token, "Unexpected token, expected expression")
}

func parseVariableDeclaration(parser *tokenParser) (Statement, error) {
	current := parser.current()

	isConstant := current.Type == lexer.Const

	// Consume keyword
	parser.consume()
	current = parser.current()

	// Get name
	if current.Type != lexer.Identifier {
		return Statement{}, parseError(current, "Expected identifier for variable declaration")
	}
	variableName := current.Value

	// Consume identifier
	parser.consume()
	current = parser.current()

	variableType := ActualType{
		Id: Void,
	}

	// Check if type is already assigned
	if current.Type == lexer.Colon {
		// Consume colon
		parser.consume()
		current = parser.current()

		// Get type
		if current.Type != lexer.Identifier {
			return Statement{}, parseError(current, "Expected type for implicit variable declaration")
		}

		parsedType, err := parseType(current)

		if err != nil {
			return Statement{}, err
		}

		if parsedType.Id == Void {
			return Statement{}, parseError(current, "Cannot assign void variable")
		}

		variableType = parsedType

		// Consume type
		parser.consume()
	}

	// Auto assign variable later

	current = parser.current()

	if isConstant && current.Type != lexer.Equals {
		return Statement{}, parseError(current, "Constant variable need to be defined with a value")
	}

	exp := Statement{}

	if current.Type == lexer.Equals {
		// Consume equals
		parser.consume()

		// Parse expression
		expression, err := parseExpression(parser)

		if err != nil {
			return Statement{}, err
		}

		exp = expression

	} else if variableType.Id == Void {
		return Statement{}, parseError(current, "Implicit declaration of type needed if no value present")
	}

	argNames := []string{variableName}
	argTypes := []ActualType{variableType}
	expressions := []Statement{exp}

	return Statement{
		Type:        VariableDeclaration,
		Expressions: expressions,
		ArgNames:    argNames,
		ArgTypes:    argTypes,
		Constant:    isConstant,
	}, nil
}

func parseFunction(parser *tokenParser) (Statement, error) {
	// Consume keyword
	parser.consume()

	current := parser.current()

	// Get identifier
	if current.Type != lexer.Identifier {
		return Statement{}, parseError(current, "Function has invalid identifier")
	}

	functionName := parser.consume().Value

	// Check for parenthesis

	current = parser.current()
	if current.Type != lexer.OpenParenthesis {
		return Statement{}, parseError(current, "Function is missing (")
	}

	// Consume (
	parser.consume()

	// Check for arguments
	argNames := []string{}
	argTypes := []ActualType{}

	for {
		current = parser.current()

		if current.Type == lexer.CloseParenthesis {
			parser.consume()
			break
		}

		argType, err := parseType(current)

		if err != nil {
			return Statement{}, err
		}

		// Consume type
		parser.consume()
		current = parser.current()

		// Check for identifier
		if current.Type != lexer.Identifier {
			return Statement{}, parseError(current, "Expected identifier for argument name")
		}

		argName := current.Value

		// Consume identifier
		parser.consume()
		current = parser.current()

		// Push to list
		argTypes = append(argTypes, argType)
		argNames = append(argNames, argName)

		// Check for )
		if current.Type == lexer.CloseParenthesis {
			parser.consume()
			break
		}

		// Check for more arguments
		if current.Type == lexer.Comma {
			parser.consume()
			continue
		}

		// Unexpected token
		return Statement{}, parseError(current, "Unexpected token in function argument declaration")
	}

	// Check for new scope OR return type(s)
	returnTypes := []ActualType{}
	current = parser.current()

	if current.Type == lexer.ArrowRight {
		// Consume arrow
		parser.consume()

		current = parser.current()

		// Check for multiple return values
		if current.Type == lexer.OpenParenthesis {
			// Consume (
			parser.consume()

			for {
				current = parser.current()

				if current.Type == lexer.CloseParenthesis {
					parser.consume()
					break
				}

				returnType, err := parseType(current)

				if err != nil {
					return Statement{}, err
				}

				returnTypes = append(returnTypes, returnType)

				// Consume type
				parser.consume()
				current = parser.current()

				// Check for )
				if current.Type == lexer.CloseParenthesis {
					parser.consume()
					break
				}

				// Check for more arguments
				if current.Type == lexer.Comma {
					parser.consume()
					continue
				}

				// Unexpected token
				return Statement{}, parseError(current, "Unexpected token in function return type declaration")
			}

		} else {
			// Check for single return value
			returnType, err := parseType(current)

			if err != nil {
				return Statement{}, err
			}

			returnTypes = append(returnTypes, returnType)

			// Consume type
			parser.consume()
		}

	} else {
		returnTypes = append(returnTypes, ActualType{
			Id: Void,
		})
	}

	current = parser.current()

	if current.Type != lexer.OpenCurlyBracket {
		return Statement{}, parseError(current, "Expected new scope for function")
	}

	scope, err := parseScope(parser)

	if err != nil {
		return Statement{}, err
	}

	return Statement{
		Type:     FunctionDeclaration,
		Value:    functionName,
		ArgTypes: argTypes,
		ArgNames: argNames,
		Types:    returnTypes,
		RunScope: &scope,
	}, nil
}

func parseType(token lexer.Token) (ActualType, error) {
	if token.Type != lexer.Identifier {
		return ActualType{}, parseError(token, "Expected type")
	}

	switch token.Value {
	case "void":
		return ActualType{Id: Void}, nil
	case "int":
		return ActualType{Id: Int32}, nil
	case "int8":
		return ActualType{Id: Int8}, nil
	case "int16":
		return ActualType{Id: Int16}, nil
	case "int32":
		return ActualType{Id: Int32}, nil
	case "int64":
		return ActualType{Id: Int64}, nil
	case "uint8":
		return ActualType{Id: UnsignedInt8}, nil
	case "uint16":
		return ActualType{Id: UnsignedInt16}, nil
	case "uint32":
		return ActualType{Id: UnsignedInt32}, nil
	case "uint64":
		return ActualType{Id: UnsignedInt64}, nil
	case "float":
		return ActualType{Id: Float}, nil
	case "double":
		return ActualType{Id: Double}, nil
	case "bool":
		return ActualType{Id: Bool}, nil
	default:
		return ActualType{Id: Custom, CustomName: token.Value}, nil
	}
}

func parseScope(parser *tokenParser) (Statement, error) {
	current := parser.current()

	if current.Type != lexer.OpenCurlyBracket {
		return Statement{}, parseError(current, "Scope needs to be opened with {")
	}

	parser.consume()

	children := []Statement{}
	closed := false

	for {
		current = parser.current()

		if current.Type == lexer.CloseCurlyBracket {
			closed = true
			parser.consume()
			break
		}

		statement, err := parseStatement(parser)

		if err != nil {
			return Statement{}, err
		}

		children = append(children, statement)
	}

	if !closed {
		return Statement{}, parseError(current, "Scope needs to be closed with }")
	}

	scope := Statement{
		Type:     ScopeDeclaration,
		Children: children,
	}

	return scope, nil
}

func parseError(token lexer.Token, message string) error {
	// Return error if unknown character is in source
	trace := token.Trace

	if trace == nil {
		return ParseError{message: "No trace found for error"}
	}

	row, col := trace.Row, trace.Column
	msg := fmt.Sprintf("%s @ %d:%d >> %+v", message, row, col, token)

	return ParseError{message: msg, trace: trace}
}
