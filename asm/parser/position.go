package parser

import "fmt"

// Position defines the source position for a token or AST node.
type Position struct {
	File   string // File in which token was defined.
	Line   int    // Line number at which token was defined.
	Col    int    // Column number at which token was defined.
	Offset int    // Byte offset at which token was defined.
}

func (p *Position) String() string {
	return fmt.Sprintf("%s:%d:%d", p.File, p.Line, p.Col)
}
