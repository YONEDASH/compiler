package lexer

import "github.com/yonedash/comet/analysis"

type TokenType int

const (
	EOF TokenType = iota
	LF
	Null // There will be no null in this language?
	Number
	String
	Identifier
	Boolean
	Equals // add plus equals etc..
	Colon
	Semicolon
	Comma
	CompareEquals
	CompareSmaller
	CompareBigger
	OpenParenthesis
	CloseParenthesis
	OpenCurlyBracket
	CloseCurlyBracket
	OpenSquareBracket
	CloseSquareBracket
	Addition
	Subtraction
	Multiplication
	Division
	Modulus
	ArrowRight
	Var // Keywords
	Const
	Function
	Import
	Native
)

var Keywords = map[string]TokenType{
	"null":   Null,
	"var":    Var,
	"const":  Const,
	"fn":     Function,
	"true":   Boolean,
	"false":  Boolean,
	"import": Import,
	"native": Native,
}

type Token struct {
	Type  TokenType
	Value string
	Trace *analysis.SourceTrace
}
