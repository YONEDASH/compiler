package analysis

import (
	"github.com/yonedash/comet/lexer"
	"github.com/yonedash/comet/parser"
)

type Statement parser.Statement

type Hint struct {
	Message string
	Trace   lexer.SourceTrace
}

func AnalyseAST(root parser.Statement) ([]Hint, error) {
	return []Hint{}, nil
}
