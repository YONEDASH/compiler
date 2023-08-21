package lexer

type TokenType int

var _tokenTypeIncrementor TokenType = 0

func _tokenTypeGet() TokenType {
	_tokenTypeIncrementor++
	return _tokenTypeIncrementor - 1
}

const (
	EOF  TokenType = iota
	Null           // There will be no null in this language?
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
	ArrowRight
	Let // Keywords
	Const
	Function
)

var Keywords = map[string]TokenType{
	"null":  Null,
	"let":   Let,
	"const": Const,
	"fn":    Function,
}

type Token struct {
	Type  TokenType
	Value string
	Trace *SourceTrace
}
