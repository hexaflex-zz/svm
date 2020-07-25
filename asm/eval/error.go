package eval

import (
	"fmt"

	"github.com/hexaflex/svm/asm/parser"
)

// Error defines a evaluation error with source context.
type Error struct {
	Pos parser.Position
	Msg string
}

// NewError creates a new, formatted error message with the given source context.
func NewError(pos parser.Position, f string, argv ...interface{}) *Error {
	return &Error{
		Pos: pos,
		Msg: fmt.Sprintf(f, argv...),
	}
}

func (e *Error) Error() string {
	return e.Pos.String() + " " + e.Msg
}
