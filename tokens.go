package main

type TokenType int

var _tokenTypeIncrementor TokenType = 0

func _tokenTypeGet() TokenType {
	_tokenTypeIncrementor++
	return _tokenTypeIncrementor - 1
}

const (
	Number TokenType = iota
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
	Plus
	Minus
	Multiply
	Divide
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
