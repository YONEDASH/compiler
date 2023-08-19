package main

type StatementType int

const (
	Root StatementType = iota
	BinaryExpression
)

type Statement struct {
	Type StatementType
}

type RootStatement struct {
	Statement
	Children *[]Statement
}

type ExpressionStatement struct {
	Statement
}

type BinaryExpressionStatement struct {
	ExpressionStatement
	Left     Statement
	Right    Statement
	Operator string
}

func b() {
	bes := BinaryExpressionStatement{}

	a(bes)
}

func a(a ExpressionStatement) {
	// This should be possible
}
