package lexer

import (
	"bufio"
	"fmt"
	"os"
	"unicode"

	"github.com/yonedash/comet/analysis"
)

type TokenizeError struct {
	message string
}

func (e TokenizeError) Error() string {
	return e.message
}

type tokenReader struct {
	text   []rune
	length int
	index  int
}

func (r tokenReader) at(i int) rune {
	if i < 0 || i >= r.length {
		return 0
	}

	return r.text[i]
}

func (r tokenReader) before() rune {
	return r.at(r.index - 1)
}

func (r tokenReader) after() rune {
	return r.at(r.index + 1)
}

func (r tokenReader) current() rune {
	return r.at(r.index)
}

func (r *tokenReader) consume() rune {
	i := r.index
	r.index = i + 1
	return r.at(i)
}

func (r tokenReader) isDone() bool {
	return r.index >= r.length
}

func getTokenReader(path string) (tokenReader, error) {
	file, err := os.Open(path)
	if err != nil {
		return tokenReader{}, err
	}

	defer file.Close()

	bufReader := bufio.NewReader(file)

	text := []rune{}

	for {
		r, _, err := bufReader.ReadRune()
		if err != nil {
			break
		}

		text = append(text, r)
	}

	reader := tokenReader{
		text:  text,
		index: 0,
	}

	reader.length = len(reader.text)

	return reader, nil
}

func Tokenize(path string) ([]Token, error) {
	reader, err := getTokenReader(path)

	if err != nil {
		return nil, err
	}

	tokens := []Token{}

	identifier := ""

	isComment := 0

	for {
		if reader.isDone() {
			break
		}

		ch := reader.current()

		// Skip comments
		if ch == '/' && reader.after() == '/' {
			safelyEndIdentifier(&identifier, &tokens, reader.index)

			reader.index += 2
			isComment = 1
			continue
		}

		if ch == '/' && reader.after() == '*' {
			safelyEndIdentifier(&identifier, &tokens, reader.index)

			reader.index += 2
			isComment = 2
			continue
		}

		if isComment == 1 && ch == '\n' {
			isComment = 0
			reader.consume()
			continue
		}

		if isComment == 2 && ch == '*' && reader.after() == '/' {
			isComment = 0
			reader.index += 2
			continue
		}

		if isComment > 0 {
			reader.consume()
			continue
		}

		fmt.Println(string(ch), ch, reader.index, "idf->", identifier, "<-", len(identifier))

		// Check for string
		if ch == '"' {
			safelyEndIdentifier(&identifier, &tokens, reader.index)
			tokens = append(tokens, str(&reader, reader.index))
			continue
		}

		if ch == '-' && reader.after() == '>' {
			appendType(ArrowRight, &identifier, &tokens, reader.index, string(reader.consume())+string(reader.consume()))
			continue
		}

		// Only make token a number if there was no identifier started
		if len(identifier) == 0 && (unicode.IsDigit(ch) || ch == '.' || ch == '-') {
			tokens = append(tokens, number(&reader, reader.index))
			continue
		}

		if ch == '\n' {
			appendType(LF, &identifier, &tokens, reader.index, string(reader.consume()))
			continue
		}

		if ch == ';' {
			appendType(Semicolon, &identifier, &tokens, reader.index, string(reader.consume()))
			continue
		}

		if ch == ':' {
			appendType(Colon, &identifier, &tokens, reader.index, string(reader.consume()))
			continue
		}

		if ch == ',' {
			appendType(Comma, &identifier, &tokens, reader.index, string(reader.consume()))
			continue
		}

		if ch == '(' {
			appendType(OpenParenthesis, &identifier, &tokens, reader.index, string(reader.consume()))
			continue
		}

		if ch == ')' {
			appendType(CloseParenthesis, &identifier, &tokens, reader.index, string(reader.consume()))
			continue
		}

		if ch == '{' {
			appendType(OpenCurlyBracket, &identifier, &tokens, reader.index, string(reader.consume()))
			continue
		}

		if ch == '}' {
			appendType(CloseCurlyBracket, &identifier, &tokens, reader.index, string(reader.consume()))
			continue
		}

		if ch == '[' {
			appendType(OpenSquareBracket, &identifier, &tokens, reader.index, string(reader.consume()))
			continue
		}

		if ch == ']' {
			appendType(CloseSquareBracket, &identifier, &tokens, reader.index, string(reader.consume()))
			continue
		}

		if ch == '+' {
			appendType(Addition, &identifier, &tokens, reader.index, string(reader.consume()))
			continue
		}

		if ch == '-' {
			appendType(Subtraction, &identifier, &tokens, reader.index, string(reader.consume()))
			continue
		}

		if ch == '*' {
			appendType(Multiplication, &identifier, &tokens, reader.index, string(reader.consume()))
			continue
		}

		if ch == '/' {
			appendType(Division, &identifier, &tokens, reader.index, string(reader.consume()))
			continue
		}

		if ch == '%' {
			appendType(Modulus, &identifier, &tokens, reader.index, string(reader.consume()))
			continue
		}

		if ch == '=' && reader.after() == '=' {
			appendType(CompareEquals, &identifier, &tokens, reader.index, string(reader.consume())+string(reader.consume()))
			continue
		}

		if ch == '=' && reader.after() == '<' {
			appendType(CompareSmaller, &identifier, &tokens, reader.index, string(reader.consume())+string(reader.consume()))
			continue
		}

		if ch == '=' && reader.after() == '>' {
			appendType(CompareBigger, &identifier, &tokens, reader.index, string(reader.consume())+string(reader.consume()))
			continue
		}

		if ch == '=' {
			appendType(Equals, &identifier, &tokens, reader.index, string(reader.consume()))
			continue
		}

		if ch == '.' && reader.after() == '.' && reader.at(reader.index+2) == '.' {
			appendType(Variadic, &identifier, &tokens, reader.index, string(reader.consume())+string(reader.consume())+string(reader.consume()))
			continue
		}

		if ch == '.' && reader.after() == '.' && reader.at(reader.index+2) == '?' {
			appendType(VariadicNoValidate, &identifier, &tokens, reader.index, string(reader.consume())+string(reader.consume())+string(reader.consume()))
			continue
		}

		if isWhitespace(ch) {
			safelyEndIdentifier(&identifier, &tokens, reader.index)

			reader.consume() // Ignore white space
			continue
		}

		// Indentifiers must start with letter and can then contain digit or .
		if unicode.IsLetter(ch) || (len(identifier) != 0 && (unicode.IsDigit(ch) || ch == '.')) {
			identifier += string(reader.consume())
			continue
		}

		// Return error if unknown character is in source
		lineFeeds := getLineFeeds(reader)
		row, col := getLocationOfIndex(reader.index, lineFeeds)

		msg := fmt.Sprintf("Unknown character @ %d:%d '%s'", row, col, string(ch))

		return nil, TokenizeError{message: msg}
	}

	// End possible missing identifier
	safelyEndIdentifier(&identifier, &tokens, reader.length)

	tokens = append(tokens, Token{
		Type: EOF,
		Trace: &analysis.SourceTrace{
			Index: reader.length - 1,
		},
	})

	fillTraces(tokens, reader)

	return tokens, nil
}

func fillTraces(tokens []Token, reader tokenReader) {
	lineFeeds := getLineFeeds(reader)

	for _, token := range tokens {
		row, col := getLocationOfIndex(token.Trace.Index, lineFeeds)
		trace := token.Trace
		trace.Row = row
		trace.Column = col
	}

}

func getLineFeeds(reader tokenReader) []int {
	lineFeeds := []int{}

	lineFeeds = append(lineFeeds, 0)

	for i := 0; i < reader.length; i++ {
		ch := reader.text[i]

		if ch == '\n' {
			lineFeeds = append(lineFeeds, i)
		}

	}

	return lineFeeds
}

func getLocationOfIndex(index int, lineFeeds []int) (int, int) {
	len := len(lineFeeds)
	for i := 0; i < len; i++ {
		lf := lineFeeds[i]

		if lf <= index && (i == len-1 || lineFeeds[i+1] > index) {
			col := index - lf

			if i == 0 {
				col++
			}

			return i + 1, col
		}
	}
	return -1, -1
}

func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\n' || ch == '\r' || ch == '\t'
}

func appendType(tokenType TokenType, identifier *string, tokens *[]Token, index int, value string) {
	safelyEndIdentifier(identifier, tokens, index)

	*tokens = append(*tokens, Token{
		Type:  tokenType,
		Value: value,
		Trace: &analysis.SourceTrace{
			Index: index,
		},
	})
}

func safelyEndIdentifier(identifier *string, tokens *[]Token, index int) {
	if identifier == nil || len(*identifier) == 0 {
		return
	}

	identifierDeref := *identifier

	// Check for keywords in identifier
	if tokenType, found := Keywords[identifierDeref]; found {
		appendType(tokenType, nil, tokens, index-len(identifierDeref), identifierDeref)
		*identifier = ""
		return
	}

	*tokens = append(*tokens, Token{
		Type:  Identifier,
		Value: identifierDeref,
		Trace: &analysis.SourceTrace{
			Index: index - len(identifierDeref),
		},
	})
	*identifier = ""
}

func str(reader *tokenReader, index int) Token {
	// Consume "
	reader.consume()

	value := ""

	for {
		ch := reader.current()

		if reader.isDone() {
			break
		}

		// \" \\"
		if (ch == '"' && reader.before() != '\\') || (ch == '"' && reader.before() == '\\' && reader.at(reader.index-2) == '\\') {
			reader.consume()
			break
		}

		value += string(ch)
		reader.consume()
	}

	token := Token{
		Type:  String,
		Value: value,
		Trace: &analysis.SourceTrace{
			Index: index,
		},
	}
	return token
}

func number(reader *tokenReader, index int) Token {
	value := ""
	dots := 0
	i := 0
	for {
		ch := reader.current()

		if i == 0 && ch == '-' {
			value += string(reader.consume())
		} else if dots == 0 && ch == '.' {
			value += string(reader.consume())
			dots++
		} else if unicode.IsDigit(ch) {
			value += string(reader.consume())
		} else {
			break
		}
		i++
	}
	token := Token{
		Type:  Number,
		Value: value,
		Trace: &analysis.SourceTrace{
			Index: index,
		},
	}
	return token
}
