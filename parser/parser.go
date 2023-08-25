package parser

import (
	"fmt"

	"github.com/yonedash/comet/analysis"
	"github.com/yonedash/comet/lexer"
)

type ParseError struct {
	message string
	trace   *analysis.SourceTrace
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

	children := []*Statement{}

	for {
		if parser.isDone() {
			break
		}

		current := parser.current()

		statement, err := parseStatement(&parser)
		if err != nil {
			return Statement{}, err
		}

		skip, err := processStatement(current, &statement)

		if err != nil {
			return Statement{}, err
		}

		if skip {
			continue
		}

		// fmt.Printf("%d %+v\n", parser.index, statement)

		children = append(children, &statement)
	}

	root := Statement{
		Type:     Root,
		Children: children,
	}

	return root, nil
}

func processStatement(start lexer.Token, statement *Statement) (bool, error) {
	if statement.Type < 0 {
		return true, nil
	}

	statement.Trace = *start.Trace

	return false, nil
}

func demandNewLineOrSemicolon(parser *tokenParser, statement Statement) (Statement, error) {
	current := parser.current()

	if current.Type != lexer.LF && current.Type != lexer.Semicolon {
		return Statement{}, parseError(current, "Expected new line or semicolon")
	}

	return statement, nil
}

func parseStatement(parser *tokenParser) (Statement, error) {
	current := parser.current()
	switch current.Type {
	case lexer.LF, lexer.Semicolon: // Ignore line feed / semicolon
		parser.consume()
		return Statement{Type: -1}, nil
	case lexer.OpenCurlyBracket:
		return parseScope(parser)
	case lexer.Function:
		return parseFunction(parser)
	case lexer.Import:
		return parseImport(parser)
	case lexer.Var, lexer.Const:
		return parseVariableDeclaration(parser)
	case lexer.Identifier, lexer.OpenParenthesis:
		if current.Type == lexer.Identifier && parser.after().Type == lexer.OpenParenthesis {
			return Statement{}, parseError(current, "FUNC CALL")
		}
		return parseVariableAssign(parser)
	}

	return Statement{}, parseError(current, fmt.Sprintf("Unexpected token, statement expected (%d)", current.Type))
}

func parseExpression(parser *tokenParser) (Statement, error) {
	/*expression := Statement{}

	return expression, nil*/
	return parseAdditiveExpression(parser)
}

func parseAdditiveExpression(parser *tokenParser) (Statement, error) {
	left, err := parseMultiplicativeExpression(parser)
	mutableLeft := left // Reassign this variable
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

		leftPtrCopy := leftPtr
		rightPtrCopy := rightPtr

		mutableLeft = Statement{
			Type:     BinaryExpression,
			Left:     leftPtrCopy,
			Right:    rightPtrCopy,
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
	mutableLeft := left // Reassign this variable
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

		leftPtrCopy := leftPtr
		rightPtrCopy := rightPtr

		mutableLeft = Statement{
			Type:     BinaryExpression,
			Left:     leftPtrCopy,
			Right:    rightPtrCopy,
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
	case lexer.String:
		parser.consume()
		return Statement{
			Type:  StringExpression,
			Value: token.Value,
		}, nil
	case lexer.Boolean:
		parser.consume()
		return Statement{
			Type:  BooleanExpression,
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

func parseVariableAssign(parser *tokenParser) (Statement, error) {
	current := parser.current()

	varIdentifiers := []*Statement{}

	if current.Type == lexer.OpenParenthesis {
		// Consume parenthesis
		parser.consume()

		for {
			// Check for possible end
			if current.Type == lexer.CloseParenthesis && len(varIdentifiers) > 0 {
				return Statement{}, parseError(current, "Unexpected token, expected identifier")
			}

			// Get identifier
			current = parser.current()

			if current.Type != lexer.Identifier {
				return Statement{}, parseError(current, "Unexpected token, expected identifier")
			}

			// Parse (function also consumes it)
			identifier, err := parsePrimaryExpression(parser)

			if err != nil {
				return Statement{}, err
			}

			// Add identifier
			varIdentifiers = append(varIdentifiers, &identifier)

			current = parser.current()

			// Check for possible end
			if current.Type == lexer.CloseParenthesis {
				parser.consume()
				break
			}

			// Check for next var
			if current.Type == lexer.Comma {
				parser.consume()
				continue
			}

			return Statement{}, parseError(current, "Unexpected token")
		}
	} else {

		if current.Type != lexer.Identifier {
			return Statement{}, parseError(current, "Expected identifier")
		}

		// Parse (function also consumes it)
		identifier, err := parsePrimaryExpression(parser)

		if err != nil {
			return Statement{}, err
		}

		// Add identifier
		varIdentifiers = append(varIdentifiers, &identifier)

	}

	current = parser.current()

	// TODO add += -= etc here

	if current.Type == lexer.Equals {

		// Consume equals
		parser.consume()
		current = parser.current()

		varExpressions := []*Statement{}

		if current.Type == lexer.OpenParenthesis {
			// Consume parenthesis
			parser.consume()

			for {
				// Get expression
				expression, err := parseExpression(parser)

				if err != nil {
					return Statement{}, err
				}

				varExpressions = append(varExpressions, &expression)

				current = parser.current()

				// Check for possible end
				if current.Type == lexer.CloseParenthesis {
					parser.consume()
					break
				}

				// Check for next var
				if current.Type == lexer.Comma {
					parser.consume()
					continue
				}

				return Statement{}, parseError(current, "Unexpected token")
			}

			if len(varIdentifiers) == 1 && len(varExpressions) > 1 {
				return Statement{}, parseError(current, "Cannot assign multiple expressions to a single variable")
			}
		} else {
			if len(varIdentifiers) > 1 {
				return Statement{}, parseError(current, "Cannot assign one expression to multiple variables")
			}

			// Get expression
			expression, err := parseExpression(parser)

			if err != nil {
				return Statement{}, err
			}

			varExpressions = append(varExpressions, &expression)
		}

		if len(varIdentifiers) != len(varExpressions) {
			return Statement{}, parseError(current, "Identifier and expression count mismatch")
		}

		return demandNewLineOrSemicolon(parser, Statement{
			Type:        VariableAssignment,
			Identifiers: varIdentifiers,
			Expressions: varExpressions,
		})
	}

	return Statement{}, parseError(current, "Unknown operation on variable")
}

func parseVariableDeclaration(parser *tokenParser) (Statement, error) {
	current := parser.current()

	isConstant := current.Type == lexer.Const

	// Consume keyword
	parser.consume()
	current = parser.current()

	varIdentifiers := []*Statement{}

	if current.Type == lexer.OpenParenthesis {
		// Consume parenthesis
		parser.consume()

		for {
			// Check for possible end
			if current.Type == lexer.CloseParenthesis && len(varIdentifiers) > 0 {
				return Statement{}, parseError(current, "Unexpected token, expected identifier")
			}

			// Get identifier
			current = parser.current()

			if current.Type != lexer.Identifier {
				return Statement{}, parseError(current, "Unexpected token, expected identifier")
			}

			// Parse (function also consumes it)
			identifier, err := parsePrimaryExpression(parser)

			if err != nil {
				return Statement{}, err
			}

			// Add identifier
			varIdentifiers = append(varIdentifiers, &identifier)

			current = parser.current()

			// Check for possible end
			if current.Type == lexer.CloseParenthesis {
				parser.consume()
				break
			}

			// Check for next var
			if current.Type == lexer.Comma {
				parser.consume()
				continue
			}

			return Statement{}, parseError(current, "Unexpected token")
		}
	} else {
		if current.Type != lexer.Identifier {
			return Statement{}, parseError(current, "Expected identifier")
		}

		// Parse (function also consumes it)
		identifier, err := parsePrimaryExpression(parser)

		if err != nil {
			return Statement{}, err
		}

		// Add identifier
		varIdentifiers = append(varIdentifiers, &identifier)

	}

	current = parser.current()

	// Check if type is already assigned
	selfAssignedType := ActualType{}

	varTypes := []ActualType{}

	if current.Type == lexer.Colon {
		// Consume colon
		parser.consume()
		current = parser.current()

		if current.Type == lexer.OpenParenthesis {
			// Consume parenthesis
			parser.consume()

			for {
				current = parser.current()

				// Get type
				parsedType, err := parseType(current)

				if err != nil {
					return Statement{}, err
				}

				if err != nil {
					return Statement{}, err
				}

				if parsedType.Id == Void {
					return Statement{}, parseError(current, "Cannot declare variable as void")
				}

				varTypes = append(varTypes, parsedType)

				// Consume type
				parser.consume()

				current = parser.current()

				// Check for possible end
				if current.Type == lexer.CloseParenthesis {
					parser.consume()
					break
				}

				// Check for next var
				if current.Type == lexer.Comma {
					parser.consume()
					continue
				}

				return Statement{}, parseError(current, "Unexpected token, expecting ) or ,")
			}

			if len(varIdentifiers) == 1 && len(varTypes) > 1 {
				return Statement{}, parseError(current, "Cannot assign multiple types to a single variable")
			}
		} else {
			// Get type
			if current.Type != lexer.Identifier {
				return Statement{}, parseError(current, "Expected type for implicit variable declaration")
			}

			parsedType, err := parseType(current)

			if err != nil {
				return Statement{}, err
			}

			if parsedType.Id == Void {
				return Statement{}, parseError(current, "Cannot declare variable as void")
			}

			varTypes = append(varTypes, parsedType)

			// Consume type
			parser.consume()
		}
	} else {
		len := len(varIdentifiers)
		for i := 0; i < len; i++ {
			varTypes = append(varTypes, ActualType{})
		}
	}

	current = parser.current()
	varExpressions := []*Statement{}

	if current.Type == lexer.Equals {
		// Consume equals
		parser.consume()
		current = parser.current()

		if current.Type == lexer.OpenParenthesis {
			// Consume parenthesis
			parser.consume()

			for {
				// Get expression
				expression, err := parseExpression(parser)

				if err != nil {
					return Statement{}, err
				}

				varExpressions = append(varExpressions, &expression)

				current = parser.current()

				// Check for possible end
				if current.Type == lexer.CloseParenthesis {
					parser.consume()
					break
				}

				// Check for next var
				if current.Type == lexer.Comma {
					parser.consume()
					continue
				}

				return Statement{}, parseError(current, "Unexpected token, expecting ) or ,")
			}

			if len(varIdentifiers) == 1 && len(varExpressions) > 1 {
				return Statement{}, parseError(current, "Cannot assign multiple expressions to a single variable")
			}
		} else {
			if len(varIdentifiers) > 1 {
				return Statement{}, parseError(current, "Cannot assign one expression to multiple variables")
			}

			// Get expression
			expression, err := parseExpression(parser)

			if err != nil {
				return Statement{}, err
			}

			varExpressions = append(varExpressions, &expression)
		}
	}

	// Update current
	current = parser.current()

	if selfAssignedType.Id == Void && len(varExpressions) == 0 {
		return Statement{}, parseError(current, "Implicit declaration of type needed when not assigning a value")
	}

	if len(varIdentifiers) != len(varExpressions) && len(varExpressions) > 0 {
		return Statement{}, parseError(current, "Identifier and expression count mismatch")
	}

	if len(varIdentifiers) != len(varTypes) && len(varTypes) > 1 {
		return Statement{}, parseError(current, "Identifier and type count mismatch")
	}

	if len(varTypes) == 1 {
		count := len(varExpressions) - 1
		for i := 0; i < count; i++ {
			varTypes = append(varTypes, varTypes[0])
		}
	}

	return demandNewLineOrSemicolon(parser, Statement{
		Type:        VariableDeclaration,
		Identifiers: varIdentifiers,
		Expressions: varExpressions,
		Types:       varTypes,
		Constant:    isConstant,
	})
}

// Scans tokens for (X, X, X, ..., X) or X
func getOrMultiGetExpr(parser *tokenParser) ([]Statement, error) {
	result := []Statement{}

	current := parser.current()

	// Check for multiple return values
	if current.Type == lexer.OpenParenthesis {
		// Consume (
		parser.consume()

		for {
			current = parser.current()

			if current.Type == lexer.CloseParenthesis {
				parser.consume()

				// Catch something like this: -> (int, ) OR ()
				return []Statement{}, parseError(current, "Unexpected token in ()")
			}

			parsed, err := parseExpression(parser)

			if err != nil {
				return []Statement{}, err
			}

			result = append(result, parsed)

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
			return []Statement{}, parseError(current, "Unexpected token in ()")
		}

	} else {
		// Check for single return value
		parsed, err := parseExpression(parser)

		if err != nil {
			return []Statement{}, err
		}

		result = append(result, parsed)
	}

	return result, nil
}

func parseImport(parser *tokenParser) (Statement, error) {
	// Consume keyword
	token := parser.consume()

	// Check if function is native
	isNative := false
	if parser.current().Type == lexer.Native {
		isNative = true
		parser.consume()
	}

	// Values
	values, err := getOrMultiGetExpr(parser)
	if err != nil {
		return Statement{}, err
	}

	strings := []string{}
	for _, value := range values {
		if value.Type != StringExpression {
			return Statement{}, parseError(token, "Expecting strings")
		}

		strings = append(strings, value.Value)
	}

	return demandNewLineOrSemicolon(parser, Statement{
		Type:     ImportStatement,
		ArgNames: strings,
		Native:   isNative,
	})
}

func parseFunction(parser *tokenParser) (Statement, error) {
	// Consume keyword
	parser.consume()

	current := parser.current()

	// Check if function is native
	isNative := false
	if current.Type == lexer.Native {
		isNative = true
		parser.consume()
		current = parser.current()
	}

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

					// Catch something like this: -> (int, ) OR ()
					return Statement{}, parseError(current, "Unexpected token, expected type")
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

	if isNative && current.Type == lexer.OpenCurlyBracket {
		return Statement{}, parseError(current, "Native function cannot define a scope")
	}

	scope := Statement{}

	if !isNative {
		if current.Type != lexer.OpenCurlyBracket {
			return Statement{}, parseError(current, "Expected new scope for function")
		}

		parsedScope, err := parseScope(parser)

		if err != nil {
			return Statement{}, err
		}

		scope = parsedScope
	}

	return Statement{
		Type:     FunctionDeclaration,
		Value:    functionName,
		ArgTypes: argTypes,
		ArgNames: argNames,
		Types:    returnTypes,
		RunScope: &scope,
		Native:   isNative,
	}, nil
}

func parseType(token lexer.Token) (ActualType, error) {
	if token.Type != lexer.Identifier {
		return ActualType{}, parseError(token, "Expected type")
	}

	switch token.Value {
	case "void":
		return ActualType{Id: Void}, nil
	case "int8":
		return ActualType{Id: Int8}, nil
	case "int16":
		return ActualType{Id: Int16}, nil
	case "int32", "int":
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
	case "float32", "float":
		return ActualType{Id: Float32}, nil
	case "float64", "double":
		return ActualType{Id: Float64}, nil
	case "complex64":
		return ActualType{Id: Complex64}, nil
	case "complex128":
		return ActualType{Id: Complex128}, nil
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

	children := []*Statement{}
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

		skip, err := processStatement(current, &statement)

		if err != nil {
			return Statement{}, err
		}

		if skip {
			continue
		}

		children = append(children, &statement)
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
