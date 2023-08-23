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
	msg := fmt.Sprintf("%s @ %d:%d >> %+v", message, row, col, statement)

	return StaticError{message: msg, trace: trace}
}

type scope struct {
	parent *scope
	vars   []scopeVar
	fns    []scopeFn
	types  []scopeType
}

type scopeVar struct {
	varType            parser.ActualType
	varName            string
	varConstant        bool
	varValueExpression parser.Statement
	varAllocated       bool
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

type staticAnalyzer struct {
	statements   []*parser.Statement
	currentScope scope
	hints        []Hint
	length       int
	index        int
}

func (r *staticAnalyzer) insert(statement *parser.Statement) {
	if r.isDone() {
		return
	}

	r.statements = append(r.statements[:r.index+1], r.statements[r.index:]...)
	r.statements[r.index] = statement
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
	analyzer, err := analyzeInstance(statement, scope{})
	return analyzer.hints, err
}

func analyzeInstance(root *parser.Statement, scope scope) (staticAnalyzer, error) {
	children := root.Children

	analyzer := staticAnalyzer{
		currentScope: scope,
		statements:   children,
		length:       len(children),
	}

	return analyzer, analyzeTree(&analyzer)
}

func analyzeTree(analyzer *staticAnalyzer) error {
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

	return nil
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
	newScope := scope{parent: &initialScope}
	analyzer.currentScope = newScope

	a, err := analyzeInstance(statement, analyzer.currentScope)
	if err != nil {
		return err
	}
	analyzer.hints = append(analyzer.hints, a.hints...)

	analyzer.currentScope = initialScope

	return nil
}

func analyzeFunctionDeclaration(analyzer *staticAnalyzer, statement *parser.Statement) error {
	err := analyzeStatement(analyzer, statement.RunScope)

	if err != nil {
		return err
	}

	name := statement.Value

	if analyzer.currentScope.getFunction(name) != nil {
		return fail(statement, fmt.Sprintf("Function %s is already declared", name))
	}

	analyzer.currentScope.fns = append(analyzer.currentScope.fns, scopeFn{
		fnTypes: statement.Types,
		fnName:  name,
	})

	return nil
}

func analyzeVariableDeclaration(analyzer *staticAnalyzer, statement *parser.Statement) error {
	assignCount := len(statement.Expressions)

	for i := 0; i < assignCount; i++ {
		identifier := statement.Identifiers[i]
		name := identifier.Value

		// Check if variable is defined
		variable := analyzer.currentScope.getVariable(name)
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
			return fail(statement, fmt.Sprintf("Value of variable %s does not match it's type", name))
		}

		if varType.Id == 0 {
			statement.Types[i] = inferredType
		}

		// Add variable to scope
		analyzer.currentScope.vars = append(analyzer.currentScope.vars, scopeVar{
			varName:            name,
			varType:            varType,
			varConstant:        statement.Constant,
			varValueExpression: expr,
			varAllocated:       true,
		})
	}

	return nil
}

func declareVariable(analyzer *staticAnalyzer, statement *parser.Statement) error {
	return nil
}

func analyzeVariableAssignment(analyzer *staticAnalyzer, statement *parser.Statement) error {
	return nil
}

func analyzeIdentifierExpression(analyzer *staticAnalyzer, statement *parser.Statement) error {
	// check if last in scope, to free var
	return nil
}

func inferType(analyzer *staticAnalyzer, expression parser.Statement, statement *parser.Statement) (parser.ActualType, error) {
	switch expression.Type {
	case parser.NumberExpression:
		value := expression.Value

		floating := strings.Contains(value, ".")

		// Check for unsigned ints
		if value[0] != '-' {

		}

		if floating {
			return parser.ActualType{Id: parser.Float32}, nil
		}

		// TODO get number type by MAX_SIZE
		return parser.ActualType{Id: parser.Int32}, nil
	case parser.BooleanExpression:
		return parser.ActualType{Id: parser.Bool}, nil
	case parser.IdentifierExpression:
		value := expression.Value

		scopeVariable := analyzer.currentScope.getVariable(value)

		if scopeVariable == nil {
			return parser.ActualType{}, fail(statement, fmt.Sprintf("Undefined identifier %s", value))
		}

		return scopeVariable.varType, nil
	}

	return parser.ActualType{}, fail(statement, "Undefined type")
}

func declareMemoryDeallocations() {

}
