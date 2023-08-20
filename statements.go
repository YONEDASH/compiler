package main

import "fmt"

type StatementType int

const (
	Root StatementType = iota
	NumberExpression
	IdentifierExpression
	BinaryExpression
)

type BinaryOperation int

const (
	AdditionOperation BinaryOperation = iota
	SubtractionOperation
	MultiplicationOperation
	DivisionOperation
	ModulusOperation
)

type Statement struct {
	Type     StatementType
	Children []Statement     // Root
	Left     *Statement      // Binary Expression
	Right    *Statement      // ^
	Operator BinaryOperation // ^
	Range    string          // Range of NumberExpression (int, float etc)
	Value    string          // NumberExpression: num value | IdentifierExpression: name | BinaryExpression: operator
}

// Debug

func PrintAST(statement Statement, i int) {
	if i > 100 {
		return
	}

	prefix := ""
	for j := 0; j < i; j++ {
		prefix += "    "
	}

	fmt.Println(prefix, "Type:", statement.Type)

	if statement.Type == NumberExpression || statement.Type == IdentifierExpression {
		fmt.Println(prefix, "Value:", statement.Value)
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
			if &child == &statement {
				fmt.Println(prefix, "Itself??")
				continue
			}

			PrintAST(child, i+1)
		}
	}
}
