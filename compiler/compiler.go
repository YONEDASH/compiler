package compiler

import (
	"fmt"

	"github.com/yonedash/comet/lexer"
	"github.com/yonedash/comet/parser"
)

type CompileError struct {
	message string
	trace   lexer.SourceTrace
}

func (e CompileError) Error() string {
	return e.message
}

func compileError(statement parser.Statement, message string) error {
	// Return error if unknown character is in source
	trace := statement.Trace

	row, col := trace.Row, trace.Column
	msg := fmt.Sprintf("%s @ %d:%d >> %+v", message, row, col, statement)

	return CompileError{message: msg, trace: trace}
}

type compiler struct {
	head            string
	prepend         string
	indent          int
	booleanImported bool
	currentScope    scope
}

type scope struct {
	parent *scope
	vars   []scopeVar
	fns    []scopeFn
}

type scopeVar struct {
	varType     parser.ActualType
	varName     string
	varConstant bool
	varValue    string
}

type scopeFn struct {
	fnTypes []parser.ActualType
	fnName  string
}

func (s scope) getVariable(name string) *scopeVar {
	for _, variable := range s.vars {
		if variable.varName == name {
			return &variable
		}
	}

	if s.parent != nil {
		return s.parent.getVariable(name)
	}

	return nil
}

func (s scope) getFunction(name string) *scopeFn {
	for _, function := range s.fns {
		if function.fnName == name {
			return &function
		}
	}

	if s.parent != nil {
		return s.parent.getFunction(name)
	}

	return nil
}

func CompileC(root parser.Statement) (string, error) {
	cl := &compiler{indent: -1, currentScope: scope{}}
	content, err := compile(cl, root)

	if err != nil {
		return "", err
	}

	return cl.head + cl.prepend + content, nil
}

func compile(cl *compiler, statement parser.Statement) (string, error) {
	switch statement.Type {
	case -1: // skip LF -> TODO: fix in parser to not be passed here
		return "", nil
	case parser.Root, parser.ScopeDeclaration:
		return compileScope(cl, statement)
	case parser.FunctionDeclaration:
		return compileFunction(cl, statement)
	case parser.VariableDeclaration:
		return compileVariableDeclaration(cl, statement)
	case parser.VariableAssignment:
		return compileVariableAssignment(cl, statement)
	case parser.BinaryExpression, parser.IdentifierExpression, parser.NumberExpression:
		return compileExpression(cl, statement)
	}

	return indent(cl) + fmt.Sprintf("// UNKNOWN STATEMENT %v", statement), nil
}

var internalTypes = map[parser.TypeId]string{
	parser.Void:          "void",
	parser.Bool:          inferBoolean(),
	parser.Int8:          "char",
	parser.Int16:         "short",
	parser.Int32:         "int",
	parser.Int64:         "long",
	parser.UnsignedInt8:  "unsigned char",
	parser.UnsignedInt16: "unsigned short",
	parser.UnsignedInt32: "unsigned int",
	parser.UnsignedInt64: "unsigned long",
	parser.Float:         "float",
	parser.Double:        "double",
}

func getTypeOfC(aType parser.ActualType) string {
	if aType.Id != parser.Custom {
		return internalTypes[aType.Id]
	}

	return aType.CustomName
}

func compileVariableAssignment(cl *compiler, statement parser.Statement) (string, error) {
	content := ""

	assignCount := len(statement.Expressions)

	for i := 0; i < assignCount; i++ {
		name := statement.ArgNames[i]
		expr := statement.Expressions[i]

		fmt.Println(name, statement.RunScope)

		compiledExpr, err := compile(cl, expr)

		if err != nil {
			return "", err
		}

		content += indent(cl) + name + " = " + compiledExpr + ";\n"
	}

	return content, nil
}

func compileVariableDeclaration(cl *compiler, statement parser.Statement) (string, error) {
	varActualType := statement.ArgTypes[0]
	varType := getTypeOfC(varActualType)

	for _, at := range statement.ArgTypes {
		if at.Id == parser.Bool {
			importBoolean(cl)
			break
		}
	}

	varName := statement.ArgNames[0]

	// Check if variable is already defined
	if cl.currentScope.getVariable(varName) != nil {
		return "", compileError(statement, "Variable is already defined")
	}

	value := statement.Value

	constant := ""

	if statement.Constant {
		constant = "const "
	}

	// Add variable to scope
	cl.currentScope.vars = append(cl.currentScope.vars, scopeVar{
		varName:     varName,
		varType:     varActualType,
		varConstant: statement.Constant,
		varValue:    value,
	})

	if len(value) == 0 {
		return indent(cl) + constant + varType + " " + varName + ";", nil
	}

	return indent(cl) + constant + varType + " " + varName + " = " + value + ";", nil
}

func compileExpression(cl *compiler, statement parser.Statement) (string, error) {
	if statement.Type == parser.NumberExpression || statement.Type == parser.IdentifierExpression {
		return statement.Value, nil
	}

	if statement.Type == parser.BinaryExpression {
		return compileBinaryExpression(cl, statement, 0)
	}

	return indent(cl) + fmt.Sprintf("// UNKNOWN EXPRESSION %v", statement), nil
}

func compileBinaryExpression(cl *compiler, statement parser.Statement, i int) (string, error) {
	left := statement.Left
	right := statement.Right
	operator := statement.Operator

	content := ""

	prioritized := operator != parser.AdditionOperation && operator != parser.SubtractionOperation

	if i > 0 && !prioritized {
		content += "("
	}

	if left.Type == parser.BinaryExpression {
		compiled, err := compileBinaryExpression(cl, *left, i+1)

		if err != nil {
			return "", nil
		}

		content += compiled
	} else {
		compiled, err := compile(cl, *left)

		if err != nil {
			return "", nil
		}

		content += compiled
	}

	switch operator {
	case parser.AdditionOperation:
		content += "+"
	case parser.SubtractionOperation:
		content += "-"
	case parser.MultiplicationOperation:
		content += "*"
	case parser.DivisionOperation:
		content += "/"
	case parser.ModulusOperation:
		content += "%"
	}

	if right.Type == parser.BinaryExpression {
		compiled, err := compileBinaryExpression(cl, *right, i+1)

		if err != nil {
			return "", nil
		}

		content += compiled
	} else {
		compiled, err := compile(cl, *right)

		if err != nil {
			return "", nil
		}

		content += compiled
	}

	if i > 0 && !prioritized {
		content += ")"
	}

	return content, nil
}

func compileFunction(cl *compiler, statement parser.Statement) (string, error) {
	importBooleanIfNeeded(cl, statement)

	content := ""

	functionName := statement.Value
	returnTypeC := "void"

	typeCount := len(statement.Types)

	if typeCount > 1 {
		returnTypeC = ""

		// Build a struct for return
		structName := inferReturnStructName(functionName)
		returnTypeC = "struct " + structName

		returnStruct := "struct " + structName + " {\n"

		cl.indent++
		for i := 0; i < typeCount; i++ {
			returnType := statement.Types[i]
			cType := getTypeOfC(returnType)

			returnStruct += indent(cl) + fmt.Sprintf("%s type%d;\n", cType, i)
		}
		cl.indent--

		returnStruct += "};\n"

		cl.prepend += returnStruct
	}

	if typeCount == 1 {
		returnTypeC = getTypeOfC(statement.Types[0])
	}

	content += indent(cl) + returnTypeC + " " + functionName + "("

	argCount := len(statement.ArgTypes)

	for i := 0; i < argCount; i++ {
		abstractArgType := statement.ArgTypes[i]
		argType := getTypeOfC(abstractArgType)
		argName := statement.ArgNames[i]

		content += argType + " " + argName

		if i != argCount-1 {
			content += ", "
		}
	}

	content += ") "

	scope := statement.RunScope

	if scope == nil {
		cl.indent++
		content += " {\n" + indent(cl) + "// NO RUN SCOPE\n}\n"
		cl.indent--
		return content, nil
	}

	compiled, err := compileScope(cl, *scope)

	if err != nil {
		return "", err
	}

	content += compiled

	return content, nil
}

func compileScope(cl *compiler, statement parser.Statement) (string, error) {
	content := ""
	indent := indent(cl)

	// Set scope in compiler
	parentScope := cl.currentScope
	cl.currentScope = scope{
		parent: &parentScope,
	}

	if statement.Type == parser.ScopeDeclaration {
		content += indent + "{\n"
	}

	cl.indent++

	for _, child := range statement.Children {
		code, err := compile(cl, child)

		if err != nil {
			return "", err
		}

		if len(code) > 0 {
			content += code + "\n"
		}
	}

	cl.indent--

	// Revert scope back to parent, since we left it
	cl.currentScope = parentScope

	if statement.Type == parser.ScopeDeclaration {
		content += indent + "}\n"
	}

	return content, nil
}

func indent(cl *compiler) string {
	str := ""
	for j := 0; j < cl.indent; j++ {
		str += "    "
	}
	return str
}

func inferName(name string) string {
	return "Comet_INTERNAL_" + name
}

func inferReturnStructName(name string) string {
	return "Return_" + inferName(name)
}

func inferBoolean() string {
	return "struct " + inferName("boolean")
}

func importBoolean(cl *compiler) {
	if cl.booleanImported {
		return
	}
	cl.head += inferBoolean() + " {\n    unsigned int value : 1;\n};\n"
	cl.booleanImported = true
}

func importBooleanIfNeeded(cl *compiler, statement parser.Statement) {
	if cl.booleanImported {
		return
	}

	for _, aType := range statement.ArgTypes {
		if aType.Id == parser.Bool {
			importBoolean(cl)
			return
		}
	}

	for _, aType := range statement.Types {
		if aType.Id == parser.Bool {
			importBoolean(cl)
			return
		}
	}
}
