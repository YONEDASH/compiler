package compiler

import (
	"fmt"

	"github.com/yonedash/comet/parser"
)

type compiler struct {
	head            string
	prepend         string
	indent          int
	booleanImported bool
}

func CompileC(root parser.Statement) string {
	cl := &compiler{indent: -1}
	content := compile(cl, root)
	return cl.head + cl.prepend + content
}

func compile(cl *compiler, statement parser.Statement) string {
	switch statement.Type {
	case -1: // skip LF -> TODO: fix in parser to not be passed here
		return ""
	case parser.Root, parser.ScopeDeclaration:
		return compileScope(cl, statement)
	case parser.FunctionDeclaration:
		return compileFunction(cl, statement)
	case parser.VariableDeclaration:
		return compileVariableDeclaration(cl, statement)
	case parser.BinaryExpression:
		return genBinaryExpression(cl, statement, 0)
	case parser.NumberExpression:
		return statement.Value
	}

	return indent(cl) + fmt.Sprintf("// UNKNOWN STATEMENT %v", statement)
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

func compileVariableDeclaration(cl *compiler, statement parser.Statement) string {
	varType := getTypeOfC(statement.ArgTypes[0])
	varName := statement.ArgNames[0]

	value := statement.Value

	constant := ""

	if statement.Constant {
		constant = "const "
	}

	if len(value) == 0 {
		return indent(cl) + constant + varType + " " + varName + ";"
	}

	return indent(cl) + constant + varType + " " + varName + " = " + value + ";"
}

func compileExpression(cl *compiler, statement parser.Statement) string {
	if statement.Type == parser.NumberExpression || statement.Type == parser.IdentifierExpression {
		return statement.Value
	}

}

func genBinaryExpression(cl *compiler, statement parser.Statement, i int) string {
	left := statement.Left
	right := statement.Right
	operator := statement.Operator

	content := ""

	prioritized := operator != parser.AdditionOperation && operator != parser.SubtractionOperation

	if i > 0 && !prioritized {
		content += "("
	}

	if left.Type == parser.BinaryExpression {
		content += genBinaryExpression(cl, *left, i+1)
	} else {
		content += compile(cl, *left)
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
		content += genBinaryExpression(cl, *right, i+1)
	} else {
		content += compile(cl, *right)
	}

	if i > 0 && !prioritized {
		content += ")"
	}

	return content
}

func compileFunction(cl *compiler, statement parser.Statement) string {
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
		return content
	}

	content += compileScope(cl, *scope)

	return content
}

func compileScope(cl *compiler, statement parser.Statement) string {
	content := ""
	indent := indent(cl)

	if statement.Type == parser.ScopeDeclaration {
		content += indent + "{\n"
	}

	cl.indent++

	for _, child := range statement.Children {
		code := compile(cl, child)
		if len(code) > 0 {
			content += code + "\n"
		}
	}

	cl.indent--

	if statement.Type == parser.ScopeDeclaration {
		content += indent + "}\n"
	}

	return content
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
