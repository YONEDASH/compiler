package compiler

import (
	"fmt"

	"github.com/yonedash/comet/analysis"
	"github.com/yonedash/comet/parser"
)

type CompileError struct {
	message string
	trace   analysis.SourceTrace
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

func CompileC(root parser.Statement) (string, error) {
	cl := &compiler{indent: -1}
	content, err := compile(cl, root, nil)

	if err != nil {
		return "", err
	}

	imports := ""

	for _, i := range cl.imports {
		imports += "#include \"" + i + "\"\n"
	}

	return imports + cl.head + cl.prepend + content, nil
}

func compile(cl *compiler, statement parser.Statement, context *parser.Scope) (string, error) {
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
		return compileExpression(cl, statement, context)
	case parser.MemoryDeAllocation:
		return compileMemoryDeAllocation(cl, statement)
	}

	return indent(cl) + fmt.Sprintf("// UNKNOWN STATEMENT %v", statement), nil
}

func compileMemoryDeAllocation(cl *compiler, statement parser.Statement) (string, error) {
	variable := statement.ContextVariable

	if variable.ALLOCATED { // todo flip logic
		return "", nil
	}

	cl.cImportLib("stdlib.h")

	return indent(cl) + "free(" + variable.VarName + ");", nil
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
		compiledIdentifier, err := compileExpression(cl, *identifier, &statement.Context)

		if err != nil {
			return "", err
		}

		expr := statement.Expressions[i]
		compiledExpr, err := compile(cl, *expr, &statement.Context)

		if err != nil {
			return "", err
		}

		content += indent(cl) + compiledIdentifier + " = " + compiledExpr + ";"

		if i != assignCount-1 {
			content += "\n"
		}
	}

	return content, nil
}

func compileVariableDeclaration(cl *compiler, statement parser.Statement) (string, error) {
	cl.cImportLib("sys/types.h")

	content := ""

	assignCount := len(statement.Expressions)

	for i := 0; i < assignCount; i++ {
		identifier := statement.Identifiers[i]

		//
		// !!! TODO Check if (re-)allocation needed, always true for testing right now
		//

		compiledIdentifier, err := compileExpression(cl, *identifier, &statement.Context)

		if err != nil {
			return "", err
		}

		expr := statement.Expressions[i]
		varType := statement.Types[i]
		compiledExpr, err := compile(cl, *expr, &statement.Context)

		if err != nil {
			return "", err
		}

		constant := ""

		if statement.Constant {
			constant = "const "
		}

		// Don't use b.value
		if varType.Id == parser.Bool {
			compiledIdentifier = identifier.Value
		}

		content += indent(cl) + constant + getTypeOfC(varType) + " " + compiledIdentifier

		if varType.Id == parser.Bool {
			importBoolean(cl)

			content += " = { value: " + compiledExpr + " }"
		} else {
			content += " = " + compiledExpr
		}

		content += ";"

		if i != assignCount-1 {
			content += "\n"
		}
	}

	return content, nil
}

func compileExpression(cl *compiler, statement parser.Statement, context *parser.Scope) (string, error) {
	if statement.Type == parser.NumberExpression || statement.Type == parser.IdentifierExpression {
		if statement.Type == parser.IdentifierExpression {
			variable := context.GetVariable(statement.Value)

			if variable != nil && variable.VarType.Id == parser.Bool {
				return statement.Value + ".value", nil
			}
		}

		return statement.Value, nil
	}

	if statement.Type == parser.BinaryExpression {
		return compileBinaryExpression(cl, statement, 0, context)
	}

	if statement.Type == parser.BooleanExpression {
		if statement.Value == "true" {
			return "1", nil
		}
		return "0", nil
	}

	return indent(cl) + fmt.Sprintf("// UNKNOWN EXPRESSION %v", statement), nil
}

func compileBinaryExpression(cl *compiler, statement parser.Statement, i int, context *parser.Scope) (string, error) {
	left := statement.Left
	right := statement.Right
	operator := statement.Operator

	content := ""

	prioritized := operator != parser.AdditionOperation && operator != parser.SubtractionOperation

	if i > 0 && !prioritized {
		content += "("
	}

	if left.Type == parser.BinaryExpression {
		compiled, err := compileBinaryExpression(cl, *left, i+1, context)

		if err != nil {
			return "", nil
		}

		content += compiled
	} else {
		compiled, err := compile(cl, *left, context)

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
		compiled, err := compileBinaryExpression(cl, *right, i+1, context)

		if err != nil {
			return "", nil
		}

		content += compiled
	} else {
		compiled, err := compile(cl, *right, context)

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

	if statement.Type == parser.ScopeDeclaration {
		content += indent(cl) + "{\n"
	}

	cl.indent++

	for _, child := range statement.Children {
		code, err := compile(cl, *child, &statement.Context)

		if err != nil {
			return "", err
		}

		if len(code) > 0 {
			content += code + "\n"
		}
	}

	cl.indent--

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
