package main

type StatementType int

const (
	Root StatementType = iota
	BinaryExpression
)

type Statement struct {
	Type     StatementType
	Children []Statement // Root
	Left     *Statement  // Binary Expression
	Right    *Statement  // ^
}
