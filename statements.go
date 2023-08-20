package main

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
