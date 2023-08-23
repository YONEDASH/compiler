package parser

import (
	"fmt"

	"github.com/yonedash/comet/analysis"
)

type StatementType int

const (
	Root StatementType = iota
	NullExpression
	NumberExpression
	IdentifierExpression
	BooleanExpression
	BinaryExpression
	FunctionDeclaration
	VariableDeclaration
	ScopeDeclaration
	VariableAssignment
	// for context builder
	MemoryDeAllocation
)

type BinaryOperation int

const (
	AdditionOperation BinaryOperation = iota
	SubtractionOperation
	MultiplicationOperation
	DivisionOperation
	ModulusOperation
)

type TypeId int

type ActualType struct {
	Id         TypeId
	CustomName string
	_ALLOCATED bool // true if to deallocate in c compiler!
	// Parent *ActualType // for something like: typedef number int32
}

const (
	Void TypeId = iota
	Bool
	Custom
	Int8 // Numbers ordered by byte count / max size
	UnsignedInt8
	Int16
	UnsignedInt16
	Float32
	Int32
	UnsignedInt32
	Float64
	Complex64
	Complex128
	Int64
	UnsignedInt64
)

func getCommonTypeId(t1 ActualType, t2 ActualType) TypeId {
	id1, id2 := t1.Id, t2.Id

	// Types match
	if id1 == id2 {
		return id1
	}

	smallest := min(id1, id2)
	biggest := max(id1, id2)

	// Custom, Bool cannot be combined
	if smallest <= Custom {
		return Void
	}

	return biggest
}

type Statement struct {
	Type        StatementType
	Children    []*Statement    // Root
	Left        *Statement      // Binary Expression
	Right       *Statement      // ^
	Operator    BinaryOperation // ^
	Range       string          // Range of NumberExpression (int, float etc)
	Value       string          // NumberExpression: num value | IdentifierExpression: name | BinaryExpression: operator
	RunScope    *Statement      // Function Declaration
	ArgTypes    []ActualType    // ^
	ArgNames    []string        // ^ & Assignment
	Types       []ActualType    // ^ & Variable Declaration (EMPTY if no vars declared)
	Expressions []Statement     // Variable Declaration & Assignment
	Identifiers []Statement     // ^
	Constant    bool            // Variable Declaration
	ArraySizes  []int           // Identifier Expression of array
	Trace       analysis.SourceTrace
}

type StatementScope struct {
	Parent   *StatementScope
	VarNames *[]string
}

// Creates new child of scope
func (s StatementScope) GetChild() StatementScope {
	child := StatementScope{
		Parent: &s,
	}
	return child
}

func (s StatementScope) DefineVariable(name string) {
	names := *s.VarNames
	names = append(names, name)
	s.VarNames = &names
}

func (s StatementScope) IsVariableDefined(name string) bool {
	for _, varName := range *s.VarNames {
		if varName == name {
			return true
		}
	}

	if s.Parent != nil {
		return s.Parent.IsVariableDefined(name)
	}

	return false
}

// Debug

func PrintAST(statement Statement, i int) {
	// Cap to depth of 10
	if i > 10 {
		return
	}

	prefix := ""
	for j := 0; j < i; j++ {
		prefix += " "
	}

	fmt.Println(prefix, "Type:", statement.Type)
	fmt.Println(prefix, "Value:", statement.Value)

	if statement.Type == FunctionDeclaration {
		fmt.Println(prefix, "ArgNames:", statement.ArgNames)
		fmt.Println(prefix, "ArgTypes:", statement.ArgTypes)
		fmt.Println(prefix, "(Return)Types:", statement.Types)
		fmt.Println(prefix, "RunScope:")
		PrintAST(*statement.RunScope, i+1)
	}

	if statement.Type == BinaryExpression {
		fmt.Println(prefix, "Operator:", statement.Operator)
	}

	if statement.Left != nil {
		fmt.Println(prefix, "Left: ")

		if statement.Left == &statement {
			fmt.Println(prefix, "Itself??")
		} else {
			PrintAST(*statement.Left, i+1)
		}
	}
	if statement.Right != nil {
		fmt.Println(prefix, "Right: ")

		if statement.Right == &statement {
			fmt.Println(prefix, "Itself??")
		} else {
			PrintAST(*statement.Right, i+1)
		}
	}
	if len(statement.Children) > 0 {
		fmt.Println(prefix, "Children: ")
		for _, child := range statement.Children {
			if child == &statement {
				fmt.Println(prefix, "Itself??")
				continue
			}

			PrintAST(*child, i+1)
		}
	}
}
