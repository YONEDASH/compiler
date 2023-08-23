package lexer

import "github.com/yonedash/comet/analysis"

type TokenType int

var _tokenTypeIncrementor TokenType = 0

func _tokenTypeGet() TokenType {
	_tokenTypeIncrementor++
	return _tokenTypeIncrementor - 1
}

const (
	EOF TokenType = iota
	LF
	Null // There will be no null in this language?
	Number
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
)

var Keywords = map[string]TokenType{
	"null":  Null,
	"var":   Var,
	"const": Const,
	"fn":    Function,
	"true":  Boolean,
	"false": Boolean,
}

type Token struct {
	Type  TokenType
	Value string
	Trace *analysis.SourceTrace
}
