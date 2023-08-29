package context

import (
	"fmt"
	"strings"

	"github.com/yonedash/comet/analysis"
	"github.com/yonedash/comet/parser"
)

type StaticError struct {
	message string
	trace   analysis.SourceTrace
}

func (e StaticError) Error() string {
	return e.message
}

func fail(statement *parser.Statement, message string) error {
	// Return error if unknown character is in source
	trace := statement.Trace

	row, col := trace.Row, trace.Column
	msg := fmt.Sprintf("%s @ %d:%d >> %v", message, row, col, statement)

	return StaticError{message: msg, trace: trace}
}

type staticAnalyzer struct {
	statements   []*parser.Statement
	currentScope parser.Scope
	hints        []Hint
	length       int
	index        int
}

func (r staticAnalyzer) at(i int) *parser.Statement {
	if i < 0 || i >= r.length {
		return &parser.Statement{}
	}

	return r.statements[i]
}

func (r staticAnalyzer) before() *parser.Statement {
	return r.at(r.index - 1)
}

func (r staticAnalyzer) after() *parser.Statement {
	return r.at(r.index + 1)
}

func (r staticAnalyzer) current() *parser.Statement {
	return r.at(r.index)
}

func (r *staticAnalyzer) consume() *parser.Statement {
	i := r.index
	r.index = i + 1
	return r.at(i)
}

func (r staticAnalyzer) isDone() bool {
	return r.index >= r.length
}

type Hint struct {
	Message   string
	Statement parser.Statement
}

type insertOrder struct {
	index     int
	statement parser.Statement
}

func (i insertOrder) insert(parent *parser.Statement) {
	if i.index >= len(parent.Children) {
		parent.Children = append(parent.Children, &i.statement)
		return
	}

	parent.Children = append(parent.Children[:i.index+1], parent.Children[i.index:]...)
	parent.Children[i.index] = &i.statement
}

func Grow(statement *parser.Statement) ([]Hint, error) {
	analyzer, err := analyzeInstance(statement, parser.Scope{})
	return analyzer.hints, err
}

func analyzeInstance(root *parser.Statement, scope parser.Scope) (staticAnalyzer, error) {
	children := root.Children

	analyzer := staticAnalyzer{
		currentScope: scope,
		statements:   children,
		length:       len(children),
	}

	return analyzer, analyzeTree(&analyzer, root)
}

func analyzeTree(analyzer *staticAnalyzer, parent *parser.Statement) error {
	for {
		if analyzer.isDone() {
			break
		}

		current := analyzer.consume()
		err := analyzeStatement(analyzer, current)

		if err != nil {
			return err
		}
	}

	err := generateAndCleanUp(analyzer, parent)

	return err
}

func analyzeStatement(analyzer *staticAnalyzer, statement *parser.Statement) error {
	// TODO check for unused vars, check for types, check for memory

	switch statement.Type {

	case parser.Root:
		return analyzeRoot(analyzer, statement)

	case parser.ScopeDeclaration:
		return analyzeScopeDeclaration(analyzer, statement)

	case parser.FunctionDeclaration:
		return analyzeFunctionDeclaration(analyzer, statement)

	case parser.VariableDeclaration:
		return analyzeVariableDeclaration(analyzer, statement)

	case parser.VariableAssignment:
		return analyzeVariableAssignment(analyzer, statement)

	case parser.IdentifierExpression:
		return analyzeIdentifierExpression(analyzer, statement)

	case parser.FunctionExpression:
		return analyzeFunctionExpression(analyzer, statement)

	}

	return nil
}

func analyzeRoot(analyzer *staticAnalyzer, statement *parser.Statement) error {
	a, err := analyzeInstance(statement, analyzer.currentScope)
	if err != nil {
		return err
	}
	analyzer.hints = append(analyzer.hints, a.hints...)

	return nil
}

func analyzeScopeDeclaration(analyzer *staticAnalyzer, statement *parser.Statement) error {
	initialScope := analyzer.currentScope
	newScope := parser.Scope{Parent: &initialScope}

	caller := statement.RunCaller
	if caller.Type == parser.FunctionDeclaration {
		// Define variables of function in new scope
		function := initialScope.GetFunction(caller.Value)

		if function == nil {
			return fail(statement, fmt.Sprintf("Could not get function %s within scope", caller.Value))
		}

		argCount := len(function.FnArgNames)
		for i := 0; i < argCount; i++ {
			argName := function.FnArgNames[i]
			argType := function.FnArgTypes[i]

			newScope.Vars = append(newScope.Vars, parser.ScopeVar{
				VarType:       argType,
				VarName:       argName,
				VarConstant:   true,
				VarOfFunction: true,
			})
		}
	}

	analyzer.currentScope = newScope

	a, err := analyzeInstance(statement, analyzer.currentScope)
	if err != nil {
		return err
	}
	analyzer.hints = append(analyzer.hints, a.hints...)

	analyzer.currentScope = initialScope

	// Set context
	statement.Context = newScope

	return nil
}

func analyzeFunctionDeclaration(analyzer *staticAnalyzer, statement *parser.Statement) error {
	name := statement.Value

	if analyzer.currentScope.Parent != nil {
		return fail(statement, "Cannot declare function outside of root scope")
	}

	if analyzer.currentScope.GetFunction(name) != nil {
		return fail(statement, fmt.Sprintf("Function %s is already declared", name))
	}

	newFn := parser.ScopeFn{
		FnTypes:    statement.Types,
		FnArgNames: statement.ArgNames,
		FnArgTypes: statement.ArgTypes,
		FnName:     name,
	}

	analyzer.currentScope.Fns = append(analyzer.currentScope.Fns, newFn)

	// Set context
	statement.Context = analyzer.currentScope
	statement.ContextFunction = &newFn

	runScope := statement.RunScope
	runScope.RunCaller = statement
	err := analyzeStatement(analyzer, runScope)

	if err != nil {
		return err
	}

	return nil
}

func analyzeVariableDeclaration(analyzer *staticAnalyzer, statement *parser.Statement) error {
	assignCount := len(statement.Expressions)

	for i := 0; i < assignCount; i++ {
		identifier := statement.Identifiers[i]
		name := identifier.Value

		// Check if variable is defined
		variable := analyzer.currentScope.GetVariable(name)
		if variable != nil {
			return fail(statement, fmt.Sprintf("Variable %s is already declared", name))
		}

		//
		// !!! TODO Check if (re-)allocation needed, always true for testing right now
		//

		expr := statement.Expressions[i]
		varType := statement.Types[i]

		inferredType, err := inferType(analyzer, expr, statement)
		if err != nil {
			return err
		}

		if varType.Id > 0 && varType.Id != inferredType.Id {
			return fail(statement, fmt.Sprintf("Variable type of %s does not match value", name))
		}

		if varType.Id == 0 {
			varType = inferredType
			statement.Types[i] = inferredType
		}

		// Add variable to scope
		newVar := parser.ScopeVar{
			VarName:            name,
			VarType:            varType,
			VarConstant:        statement.Constant,
			VarValueExpression: expr,
			// VarAllocated:       true, ! no ! compiler will decide, always expect to de-allocate
		}

		analyzer.currentScope.Vars = append(analyzer.currentScope.Vars, newVar)

		// Set context
		statement.Context = analyzer.currentScope
		statement.ContextVariable = &newVar
	}

	return nil
}

func analyzeVariableAssignment(analyzer *staticAnalyzer, statement *parser.Statement) error {
	assignCount := len(statement.Expressions)

	for i := 0; i < assignCount; i++ {
		identifier := statement.Identifiers[i]
		name := identifier.Value

		// Check if variable is defined
		variable := analyzer.currentScope.GetVariable(name)
		if variable == nil {
			return fail(statement, fmt.Sprintf("Variable %s is not defined", name))
		}

		// Check if variable is constant
		if variable.VarConstant {
			return fail(statement, fmt.Sprintf("Variable %s is immutable", name))
		}

		expr := statement.Expressions[i]

		inferredType, err := inferType(analyzer, expr, statement)

		if err != nil {
			return err
		}

		if inferredType.Id != variable.VarType.Id {
			return fail(statement, fmt.Sprintf("Value of variable %s has an mismatched type", name))
		}

		// Set context
		statement.Context = analyzer.currentScope
		statement.ContextVariable = variable
	}

	return nil
}

func analyzeIdentifierExpression(analyzer *staticAnalyzer, statement *parser.Statement) error {
	// check if last in scope, to free var

	name := statement.Value
	variable := analyzer.currentScope.GetVariable(name)

	if variable == nil {
		return fail(statement, fmt.Sprintf("Undefined identifier %s", name))
	}

	return nil
}

func analyzeFunctionExpression(analyzer *staticAnalyzer, statement *parser.Statement) error {
	name := statement.Value
	function := analyzer.currentScope.GetFunction(name)

	if function == nil {
		return fail(statement, fmt.Sprintf("Undefined function %s", name))
	}

	functionArgTypes := function.FnArgTypes
	argTypeCount := len(functionArgTypes)
	inputArgs := statement.Expressions
	argInputCount := len(inputArgs)

	if argInputCount != argTypeCount && (argTypeCount == 0 || !functionArgTypes[argTypeCount-1].Variadic) {
		return fail(statement, "Invalid argument count")
	}

	for i := 0; i < argInputCount; i++ {
		var expectedType parser.ActualType

		if i >= argTypeCount {
			expectedType = functionArgTypes[argTypeCount-1]
		} else {
			expectedType = functionArgTypes[i]
		}

		expression := inputArgs[i]
		inferredType, err := inferType(analyzer, expression, statement)

		if err != nil {
			return err
		}

		if expectedType.Id != inferredType.Id {
			return fail(statement, fmt.Sprintf("Invalid type in argument #%d in function call %s (%d != %d)", i, name, expectedType.Id, inferredType.Id))
		}
	}

	return nil
}

func isUsingVariable(statement parser.Statement, variable parser.ScopeVar) bool {
	switch statement.Type {

	case parser.VariableDeclaration, parser.VariableAssignment:
		for _, identifier := range statement.Identifiers {
			if isUsingVariable(*identifier, variable) {
				return true
			}
		}
		for _, expr := range statement.Expressions {
			if isUsingVariable(*expr, variable) {
				return true
			}
		}

	case parser.BinaryExpression:
		leftUsing := isUsingVariable(*statement.Left, variable)
		rightUsing := isUsingVariable(*statement.Right, variable)

		if leftUsing {
			return true
		}

		if rightUsing {
			return true
		}

	case parser.IdentifierExpression:
		return variable.VarName == statement.Value
	}

	return false
}

func generateAndCleanUp(analyzer *staticAnalyzer, parent *parser.Statement) error {
	scope := analyzer.currentScope

	offset := 1 // De-allocate after index

	children := parent.Children

	for _, variable := range scope.Vars {
		cv := variable

		lastUsageIndex := -1

		usageCount := 0
		firstUsage := parser.Statement{}

		for i := 0; i < len(children); i++ {
			child := children[i]

			if isUsingVariable(*child, variable) {
				lastUsageIndex = i
				usageCount++

				if usageCount == 1 {
					firstUsage = *child
				}
			}
		}

		fmt.Println(variable.VarName, usageCount)

		if usageCount <= 1 && !variable.VarOfFunction {
			return fail(&firstUsage, fmt.Sprintf("Unused variable %s", variable.VarName))
		}

		// Always append freeing statement, compiler needs to decide whether to act on it or not!

		stmt := &parser.Statement{
			Type:            parser.MemoryDeAllocation,
			Context:         analyzer.currentScope,
			ContextVariable: &cv,
		}

		index := lastUsageIndex + offset
		offset++

		if index == len(parent.Children) {
			parent.Children = append(parent.Children, stmt)
		} else {
			parent.Children = append(parent.Children[:index+1], parent.Children[index:]...)
			parent.Children[index] = stmt
		}

	}

	return nil
}

func inferType(analyzer *staticAnalyzer, expression *parser.Statement, statement *parser.Statement) (parser.ActualType, error) {
	switch expression.Type {
	case parser.NumberLiteral:
		value := expression.Value

		floating := strings.Contains(value, ".")
		// TODO get number type by MAX_SIZE

		// Check for unsigned ints
		if value[0] != '-' {

		}

		if floating {
			return parser.ActualType{Id: parser.Float32}, nil
		}

		return parser.ActualType{Id: parser.Int32}, nil

	case parser.BooleanLiteral:
		return parser.ActualType{Id: parser.Bool}, nil

	case parser.StringLiteral:
		return parser.ActualType{Id: parser.String}, nil

	case parser.IdentifierExpression:
		value := expression.Value

		scopeVariable := analyzer.currentScope.GetVariable(value)

		if scopeVariable == nil {
			return parser.ActualType{}, fail(statement, fmt.Sprintf("Undefined identifier %s", value))
		}

		return scopeVariable.VarType, nil

	case parser.BinaryExpression:
		return inferBinaryType(analyzer, expression)

	case parser.FunctionExpression:
		value := expression.Value

		function := analyzer.currentScope.GetFunction(value)

		if function == nil {
			return parser.ActualType{}, fail(statement, fmt.Sprintf("Undefined function %s", value))
		}

		types := function.FnTypes
		typeCount := len(types)

		if typeCount == 0 {
			return parser.ActualType{}, fail(statement, fmt.Sprintf("Function %s does not return any value", value))
		}

		if typeCount > 1 {
			return parser.ActualType{}, fail(statement, fmt.Sprintf("Function %s returns multiple values, can only accept one", value))
		}

		return types[0], nil
	}

	return parser.ActualType{}, fail(statement, "Undefined type")
}

func inferBinaryType(analyzer *staticAnalyzer, statement *parser.Statement) (parser.ActualType, error) {
	if statement.Left == nil {
		return parser.ActualType{}, fail(statement, fmt.Sprintf("Left side could not be dereferenced %v", statement))
	}

	leftType, err := inferType(analyzer, statement.Left, statement)

	if err != nil {
		return parser.ActualType{}, err
	}

	if statement.Right == nil {
		return parser.ActualType{}, fail(statement, fmt.Sprintf("Left side could not be dereferenced %v", statement))
	}

	rightType, err := inferType(analyzer, statement.Right, statement)

	if err != nil {
		return parser.ActualType{}, err
	}

	if leftType.Id != rightType.Id {
		return parser.ActualType{}, fail(statement, "Cannot combine types TODO ADD SUPPORT LATER")
	}

	combinedType := leftType

	return combinedType, nil
}
