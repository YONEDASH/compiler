package main

type TokenType int

var _tokenTypeIncrementor TokenType = 0

func _tokenTypeGet() TokenType {
	_tokenTypeIncrementor++
	return _tokenTypeIncrementor - 1
}

const (
	EOF TokenType = iota
	Number
	Identifier
	Equals // add plus equals etc..
	Colon
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
	Let // Keywords
	Function
)

var Keywords = map[string]TokenType{
	"let": Let,
	"fn":  Function,
}

type Token struct {
	Type  TokenType
	Value string
	Trace *SourceTrace
}