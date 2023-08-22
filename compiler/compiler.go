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
	imports         []string
}

func (c *compiler) cImportLib(path string) {
	for _, i := range c.imports {
		if i == path {
			return
		}
	}
	c.imports = append(c.imports, path)
}

type scope struct {
	parent *scope
	vars   []scopeVar
	fns    []scopeFn
	types  []scopeType
}

type scopeVar struct {
	varType      parser.ActualType
	varName      string
	varConstant  bool
	varValue     string
	varAllocated bool
}

type scopeFn struct {
	fnTypes []parser.ActualType
	fnName  string
}

type scopeType struct {
	typeName string
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

func (s scope) getType(name string) *scopeType {
	for _, t := range s.types {
		if t.typeName == name {
			return &t
		}
	}

	if s.parent != nil {
		return s.parent.getType(name)
	}

	return nil
}

func CompileC(root parser.Statement) (string, error) {
	cl := &compiler{indent: -1, currentScope: scope{}}
	content, err := compile(cl, root)

	if err != nil {
		return "", err
	}

	imports := ""

	for _, i := range cl.imports {
		imports += "#include \"" + i + "\"\n"
	}

	return imports + cl.head + cl.prepend + content, nil
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
	case parser.BinaryExpression, parser.IdentifierExpression, parser.NumberExpression, parser.BooleanExpression:
		return compileExpression(cl, statement)
	}

	return indent(cl) + fmt.Sprintf("// UNKNOWN STATEMENT %v", statement), nil
}

var internalTypes = map[parser.TypeId]string{
	// TODO: __UINT_FAST16_TYPE__ __INT16_TYPE__
	parser.Void:          "void",
	parser.Bool:          inferBoolean(),
	parser.Int8:          "int8_t",
	parser.Int16:         "int16_t",
	parser.Int32:         "int32_t",
	parser.Int64:         "int64_t",
	parser.UnsignedInt8:  "uint8_t",
	parser.UnsignedInt16: "uint16_t",
	parser.UnsignedInt32: "uint32_t",
	parser.UnsignedInt64: "uint64_t",
	parser.Float32:       "float",
	parser.Float64:       "double",
	parser.Complex64:     "float _Complex",
	parser.Complex128:    "double _Complex",
}

func getTypeOfC(aType parser.ActualType) string {
	if aType.Id != parser.Custom {
		return internalTypes[aType.Id]
	}

	return aType.CustomName
}

func compileVariableAssignment(cl *compiler, statement parser.Statement) (string, error) {
	cl.cImportLib("sys/types.h")

	content := ""

	assignCount := len(statement.Expressions)

	for i := 0; i < assignCount; i++ {
		identifier := statement.Identifiers[i]
		name := identifier.Value

		// Check if variable is defined
		variable := cl.currentScope.getVariable(name)
		if variable == nil {
			return "", compileError(statement, fmt.Sprintf("Variable %s is not defined", name))
		}

		// Check if variable is constant
		if variable.varConstant {
			return "", compileError(statement, fmt.Sprintf("Variable %s is immutable", name))
		}

		// Free memory from before
		if variable.varAllocated {
			content += indent(cl) + "// TODO FREE MEMORY\n"
		}

		compiledIdentifier, err := compileExpression(cl, identifier)

		if err != nil {
			return "", err
		}

		expr := statement.Expressions[i]

		inferredType, err := inferType(cl, expr, statement)

		if err != nil {
			return "", err
		}

		// ALSO DO THIS FOR DECLARATION INFERRING
		if inferredType.Id != variable.varType.Id {
			return "", compileError(statement, "Expression type does not match variable type")
		}

		compiledExpr, err := compile(cl, expr)

		if err != nil {
			return "", err
		}

		content += indent(cl) + compiledIdentifier + " = " + compiledExpr + ";\n"
	}

	return content, nil
}

func inferTypes(cl *compiler, statement parser.Statement) ([]parser.ActualType, error) {
	types := []parser.ActualType{}
	expressions := statement.Expressions
	len := len(statement.Expressions)

	for i := 0; i < len; i++ {
		expression := expressions[i]
		actualType, err := inferType(cl, expression, statement)

		if err != nil {
			return types, err
		}

		types = append(types, actualType)
	}

	return types, nil
}

func inferType(cl *compiler, expression parser.Statement, parent parser.Statement) (parser.ActualType, error) {
	switch expression.Type {
	case parser.NumberExpression:
		value := expression.Value

		// Check for unsigned ints
		if value[0] != '-' {

		}

		// TODO get number type by MAX_SIZE
		return parser.ActualType{Id: parser.Int32}, nil
	case parser.BooleanExpression:
		return parser.ActualType{Id: parser.Bool}, nil
	case parser.IdentifierExpression:
		value := expression.Value

		if value == internalTypes[parser.Void] {
			return parser.ActualType{Id: parser.Void}, nil
		}
	}

	return parser.ActualType{}, compileError(parent, fmt.Sprintf("Undefined type %s of id %d", expression.Value, expression.Type))
}

func compileVariableDeclaration(cl *compiler, statement parser.Statement) (string, error) {
	cl.cImportLib("sys/types.h")

	content := ""

	assignCount := len(statement.Expressions)

	if len(statement.Types) == 0 {
		types, err := inferTypes(cl, statement)
		if err != nil {
			return "", err
		}
		statement.Types = types
	}

	for i := 0; i < assignCount; i++ {
		identifier := statement.Identifiers[i]
		name := identifier.Value

		// Check if variable is defined
		variable := cl.currentScope.getVariable(name)
		if variable != nil {
			return "", compileError(statement, fmt.Sprintf("Variable %s is already defined", name))
		}

		//
		// !!! TODO Check if allocation needed, always true for testing right now
		//

		compiledIdentifier, err := compileExpression(cl, identifier)

		if err != nil {
			return "", err
		}

		expr := statement.Expressions[i]

		compiledExpr, err := compile(cl, expr)

		if err != nil {
			return "", err
		}

		constant := ""

		if statement.Constant {
			constant = "const "
		}

		varType := statement.Types[i]

		content += indent(cl) + constant + getTypeOfC(varType) + " " + compiledIdentifier

		if varType.Id == parser.Bool {
			importBoolean(cl)

			content += " = { value: " + compiledExpr + " }"
		} else {
			content += " = " + compiledExpr
		}

		content += ";\n"

		// Add variable to scope
		cl.currentScope.vars = append(cl.currentScope.vars, scopeVar{
			varName:      name,
			varType:      varType,
			varConstant:  statement.Constant,
			varValue:     compiledExpr,
			varAllocated: true,
		})
	}

	return content, nil
}

func compileExpression(cl *compiler, statement parser.Statement) (string, error) {
	if statement.Type == parser.NumberExpression || statement.Type == parser.IdentifierExpression {
		if statement.Type == parser.IdentifierExpression {
			variable := cl.currentScope.getVariable(statement.Value)

			if variable != nil && variable.varType.Id == parser.Bool {
				return statement.Value + ".value", nil
			}
		}

		return statement.Value, nil
	}

	if statement.Type == parser.BinaryExpression {
		return compileBinaryExpression(cl, statement, 0)
	}

	if statement.Type == parser.BooleanExpression {
		if statement.Value == "true" {
			return "1", nil
		}
		return "0", nil
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

	// Set scope in compiler
	parentScope := cl.currentScope
	cl.currentScope = scope{
		parent: &parentScope,
	}

	if statement.Type == parser.ScopeDeclaration {
		content += indent(cl) + "{\n"
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

	// Free memory IF not exported, if so pass to exported variable
	for _, variable := range cl.currentScope.vars {
		if variable.varAllocated {
			content += indent(cl) + "// TODO free(" + variable.varName + ");\n"
		}
	}

	cl.indent--

	// Revert scope back to parent, since we left it
	cl.currentScope = parentScope

	if statement.Type == parser.ScopeDeclaration {
		content += indent(cl) + "}\n"
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
