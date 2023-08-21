package compiler

import (
	"fmt"

	"github.com/yonedash/comet/parser"
)

type compiler struct {
	head    string
	prepend string
	indent  int
}

func CompileC(root parser.Statement) string {
	cl := &compiler{indent: -1}
	content := compile(cl, root)
	return cl.head + cl.prepend + content
}

func compile(cl *compiler, statement parser.Statement) string {
	switch statement.Type {
	case parser.Root, parser.ScopeDeclaration:
		return compileScope(cl, statement)
	case parser.FunctionDeclaration:
		return compileFunction(cl, statement)
	}

	return indent(cl) + fmt.Sprintf("// UNKNOWN STATEMENT %v", statement)
}

var internalTypes = map[parser.TypeId]string{
	parser.Void:  "void",
	parser.Bool:  "int8_t",
	parser.Int8:  "int8_t",
	parser.Int16: "int16_t",
	parser.Int32: "int32_t",
	parser.Int64: "int64_t",
}

func getTypeOfC(aType parser.ActualType) string {
	if aType.Id != parser.Custom {
		return internalTypes[aType.Id]
	}

	return aType.CustomName
}

func compileFunction(cl *compiler, statement parser.Statement) string {
	content := ""

	functionName := statement.Value
	returnTypeC := "void"

	typeCount := len(statement.Types)

	if typeCount > 1 {
		returnTypeC = ""

		// Build a struct for return
		structName := "RS_" + functionName
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
		content += compile(cl, child) + "\n"
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
